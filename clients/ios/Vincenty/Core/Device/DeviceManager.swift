import Foundation
import UIKit

/// Manages device identity for the current iOS client.
///
/// Mirrors the web client's `ensureDevice` logic in `WebSocketProvider`:
/// 1. Check Keychain for stored device_id (fast path)
/// 2. POST /users/me/devices/resolve — server-side cookie/UA heuristic
/// 3. If no existing devices → auto-create silently with device model name
/// 4. If existing devices but no match → set `pendingEnrolment` to show sheet
///
/// The iOS version uses `device_type: "ios"` and derives the default name
/// from `UIDevice.current.name` (e.g. "Matt's iPhone").
@Observable @MainActor
final class DeviceManager {

    /// Non-nil when the user needs to pick or create a device via the enrolment sheet.
    private(set) var pendingEnrolment: [Device]?

    /// The resolved device ID, stored in Keychain.
    private(set) var deviceId: String?

    /// Whether device resolution is currently in progress.
    private(set) var isResolving = false

    private let api = APIClient.shared
    private let keychain = KeychainStore.shared

    // MARK: - Resolution

    /// Attempt to resolve the current device. Returns the device ID if immediately
    /// available, or nil if the enrolment sheet needs to be shown.
    @discardableResult
    func ensureDevice() async -> String? {
        isResolving = true
        defer { isResolving = false }

        // 1. Fast path: Keychain
        if let stored = keychain.deviceId {
            deviceId = stored
            return stored
        }

        // 2. Server-side resolve
        do {
            let result: DeviceResolveResponse = try await api.post(Endpoints.usersMeDevicesResolve)

            if result.matched, let device = result.device {
                // Cookie or UA matched — use silently
                store(device.id)
                return device.id
            }

            let existing = result.existingDevices ?? []

            if existing.isEmpty {
                // 3. First login — no devices at all, auto-create
                let device = try await createDevice(name: defaultDeviceName)
                store(device.id)
                return device.id
            }

            // 4. User has existing devices but none matched — prompt
            pendingEnrolment = existing
            return nil

        } catch {
            print("[DeviceManager] Failed to resolve device: \(error)")
            return nil
        }
    }

    /// Called by the enrolment sheet when the user selects an existing device.
    func claimExistingDevice(id: String) async throws -> Device {
        let device: Device = try await api.post(Endpoints.usersMeDeviceClaim(id))
        store(device.id)
        pendingEnrolment = nil
        return device
    }

    /// Called by the enrolment sheet when the user creates a new device.
    func registerNewDevice(name: String?) async throws -> Device {
        let deviceName = (name?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
            ? defaultDeviceName
            : name!.trimmingCharacters(in: .whitespacesAndNewlines)

        let body = CreateDeviceRequest.ios(name: deviceName)
        let device: Device = try await api.post(Endpoints.usersMeDevices, body: body)
        store(device.id)
        pendingEnrolment = nil
        return device
    }

    /// Complete enrolment with a device ID (called from the sheet).
    func resolveEnrolment(_ id: String) {
        store(id)
        pendingEnrolment = nil
    }

    /// Clear device state (on logout or stale device).
    func clearDevice() {
        keychain.deviceId = nil
        deviceId = nil
        pendingEnrolment = nil
    }

    // MARK: - Device CRUD (for settings page)

    /// Fetch the current user's devices.
    func fetchMyDevices() async throws -> [Device] {
        try await api.get(Endpoints.usersMeDevices)
    }

    /// Update a device (e.g. rename).
    func updateDevice(id: String, name: String) async throws -> Device {
        try await api.put(Endpoints.device(id), body: UpdateDeviceRequest(name: name))
    }

    /// Set a device as primary.
    func setPrimary(id: String) async throws -> Device {
        try await api.put(Endpoints.usersMeDevicePrimary(id))
    }

    /// Delete a device.
    func deleteDevice(id: String) async throws {
        try await api.delete(Endpoints.device(id))
        // If we just deleted our own device, clear local state
        if id == deviceId {
            clearDevice()
        }
    }

    // MARK: - Private

    private func store(_ id: String) {
        keychain.deviceId = id
        deviceId = id
    }

    private func createDevice(name: String) async throws -> Device {
        let body = CreateDeviceRequest.ios(name: name)
        return try await api.post(Endpoints.usersMeDevices, body: body)
    }

    /// Default device name derived from the device model.
    private var defaultDeviceName: String {
        let name = UIDevice.current.name
        // UIDevice.current.name returns the user-configured name (e.g. "Matt's iPhone")
        // If it's generic, append the model
        if name.lowercased() == "iphone" || name.lowercased() == "ipad" {
            return "\(name) (\(UIDevice.current.model))"
        }
        return name
    }
}

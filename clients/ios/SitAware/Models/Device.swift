import Foundation

// MARK: - Device

struct Device: Codable, Sendable, Identifiable {
    let id: String
    let userId: String
    let name: String
    let deviceType: String
    let deviceUid: String
    var userAgent: String?
    var appVersion: String?
    let isPrimary: Bool
    var lastSeenAt: String?
    let createdAt: String
    let updatedAt: String
}

// MARK: - Device Resolution

/// Response from `POST /users/me/devices/resolve` — heuristic device matching.
struct DeviceResolveResponse: Codable, Sendable {
    let matched: Bool
    var device: Device?
    var existingDevices: [Device]?
}

// MARK: - Requests

struct CreateDeviceRequest: Codable, Sendable {
    let name: String
    let deviceType: String
    let appVersion: String

    static func ios(name: String = "iPhone") -> CreateDeviceRequest {
        CreateDeviceRequest(name: name, deviceType: "ios", appVersion: BuildInfo.version)
    }
}

struct UpdateDeviceRequest: Codable, Sendable {
    var name: String?
}

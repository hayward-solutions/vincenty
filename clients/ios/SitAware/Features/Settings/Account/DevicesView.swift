import SwiftUI

/// Devices management — list, rename, remove, set primary.
///
/// Mirrors the web client's `settings/account/devices/page.tsx`:
/// - Table of all user's devices
/// - Current device badge
/// - Set primary, rename, remove actions
struct DevicesView: View {
    @Environment(DeviceManager.self) private var deviceManager

    @State private var devices: [Device] = []
    @State private var isLoading = true
    @State private var errorMessage: String?

    // Rename
    @State private var deviceToRename: Device?
    @State private var renameText = ""
    @State private var showRenameSheet = false
    @State private var isRenaming = false

    // Remove
    @State private var deviceToRemove: Device?
    @State private var showRemoveAlert = false
    @State private var isRemovingDevice = false

    private let api = APIClient.shared

    var body: some View {
        List {
            if isLoading {
                Section {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                }
            } else if devices.isEmpty {
                Section {
                    Text("No devices found")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("Your Devices") {
                    ForEach(devices) { device in
                        deviceRow(device)
                    }
                }
            }

            if let error = errorMessage {
                Section {
                    Text(error)
                        .foregroundStyle(.red)
                        .font(.caption)
                }
            }
        }
        .navigationTitle("Devices")
        .task { await loadDevices() }
        .refreshable { await loadDevices() }
        .sheet(isPresented: $showRenameSheet) { renameSheet }
        .alert("Remove Device", isPresented: $showRemoveAlert) {
            Button("Cancel", role: .cancel) {}
            Button("Remove", role: .destructive) {
                if let device = deviceToRemove {
                    Task { await removeDevice(device) }
                }
            }
        } message: {
            Text("Are you sure you want to remove \"\(deviceToRemove?.name ?? "this device")\"?")
        }
    }

    // MARK: - Device Row

    @ViewBuilder
    private func deviceRow(_ device: Device) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            // Name + badges
            HStack(spacing: 8) {
                Image(systemName: iconForType(device.deviceType))
                    .foregroundStyle(.secondary)
                    .accessibilityHidden(true)

                VStack(alignment: .leading, spacing: 2) {
                    HStack(spacing: 6) {
                        Text(device.name)
                            .font(.subheadline.weight(.medium))

                        if isCurrentDevice(device) {
                            BadgeView(text: "This device", color: .accentColor)
                        }

                        if device.isPrimary {
                            BadgeView(text: "Primary", color: .green)
                        }
                    }

                    HStack(spacing: 8) {
                        Text(device.deviceType.uppercased())
                            .font(.caption2)
                            .foregroundStyle(.secondary)

                        if let lastSeen = device.lastSeenAt {
                            Text(relativeTime(lastSeen))
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }

                        Text("Registered \(formatDate(device.createdAt))")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }

                Spacer()
            }

            // Action buttons
            HStack(spacing: 12) {
                // Rename
                Button {
                    deviceToRename = device
                    renameText = device.name
                    showRenameSheet = true
                } label: {
                    Label("Rename", systemImage: "pencil")
                        .font(.caption)
                }

                // Set primary (only for non-primary devices)
                if !device.isPrimary {
                    Button {
                        Task { await setPrimary(device) }
                    } label: {
                        Label("Set Primary", systemImage: "star")
                            .font(.caption)
                    }
                }

                Spacer()

                // Remove (not allowed for current device)
                if !isCurrentDevice(device) {
                    Button(role: .destructive) {
                        deviceToRemove = device
                        showRemoveAlert = true
                    } label: {
                        Label("Remove", systemImage: "trash")
                            .font(.caption)
                    }
                }
            }
        }
        .padding(.vertical, 4)
        .accessibilityElement(children: .contain)
        .accessibilityLabel("\(device.name), \(device.deviceType) device\(isCurrentDevice(device) ? ", this device" : "")\(device.isPrimary ? ", primary" : "")")
    }

    // MARK: - Rename Sheet

    @ViewBuilder
    private var renameSheet: some View {
        NavigationStack {
            Form {
                Section("Device Name") {
                    TextField("Name", text: $renameText)
                        .textInputAutocapitalization(.words)
                }
            }
            .navigationTitle("Rename Device")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showRenameSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await renameDevice() }
                    }
                    .disabled(renameText.trimmingCharacters(in: .whitespaces).isEmpty || isRenaming)
                }
            }
        }
    }

    // MARK: - API Actions

    private func loadDevices() async {
        isLoading = true
        errorMessage = nil

        do {
            devices = try await api.get(Endpoints.usersMeDevices)
        } catch {
            errorMessage = "Failed to load devices"
        }

        isLoading = false
    }

    private func renameDevice() async {
        guard let device = deviceToRename else { return }
        isRenaming = true

        do {
            let body = UpdateDeviceRequest(name: renameText.trimmingCharacters(in: .whitespaces))
            let _: Device = try await api.put(Endpoints.device(device.id), body: body)
            showRenameSheet = false
            await loadDevices()
        } catch {
            errorMessage = "Failed to rename device"
        }

        isRenaming = false
    }

    private func removeDevice(_ device: Device) async {
        isRemovingDevice = true

        do {
            try await api.delete(Endpoints.device(device.id))
            await loadDevices()
        } catch {
            errorMessage = "Failed to remove device"
        }

        isRemovingDevice = false
    }

    private func setPrimary(_ device: Device) async {
        do {
            let _: Device = try await api.put(Endpoints.usersMeDevicePrimary(device.id))
            await loadDevices()
        } catch {
            errorMessage = "Failed to set primary device"
        }
    }

    // MARK: - Helpers

    private func isCurrentDevice(_ device: Device) -> Bool {
        device.id == deviceManager.deviceId
    }

    private func iconForType(_ type: String) -> String {
        switch type.lowercased() {
        case "ios": return "iphone"
        case "android": return "candybarphone"
        case "web": return "globe"
        default: return "desktopcomputer"
        }
    }

    private func relativeTime(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: iso) else { return "" }
        let seconds = -date.timeIntervalSinceNow

        if seconds < 60 { return "Just now" }
        if seconds < 3600 { return "\(Int(seconds / 60))m ago" }
        if seconds < 86400 { return "\(Int(seconds / 3600))h ago" }
        if seconds < 2_592_000 { return "\(Int(seconds / 86400))d ago" }
        if seconds < 31_536_000 { return "\(Int(seconds / 2_592_000))mo ago" }
        return "\(Int(seconds / 31_536_000))y ago"
    }

    private func formatDate(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: iso) else { return iso }
        return date.formatted(date: .abbreviated, time: .omitted)
    }
}

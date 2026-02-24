import SwiftUI

/// Non-dismissible sheet shown when the server can't match this device
/// to any existing registration. The user must either claim an existing
/// device or register a new one before the WebSocket can connect.
///
/// Mirrors the web client's `DeviceEnrolmentDialog`.
struct DeviceEnrolmentSheet: View {
    let existingDevices: [Device]
    let onResolved: (String) -> Void

    @Environment(DeviceManager.self) private var deviceManager

    @State private var deviceName = ""
    @State private var isCreating = false
    @State private var claimingId: String?
    @State private var errorMessage: String?

    private var isBusy: Bool { isCreating || claimingId != nil }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    // Header
                    VStack(spacing: 8) {
                        Image(systemName: "iphone.slash")
                            .font(.system(size: 40))
                            .foregroundStyle(.secondary)

                        Text("Device Not Recognised")
                            .font(.title3.bold())

                        Text("We don't recognise this device. Would you like to register it as a new device, or re-use an existing one?")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding(.top, 8)

                    // Error banner
                    if let errorMessage {
                        HStack(spacing: 6) {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .font(.caption)
                            Text(errorMessage)
                                .font(.caption)
                        }
                        .foregroundStyle(.red)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(.red.opacity(0.1))
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    }

                    // Existing devices
                    if !existingDevices.isEmpty {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Your existing devices")
                                .font(.subheadline.weight(.semibold))

                            VStack(spacing: 0) {
                                ForEach(existingDevices) { device in
                                    DeviceRow(
                                        device: device,
                                        isBusy: isBusy,
                                        isClaiming: claimingId == device.id,
                                        onClaim: { claimDevice(device) })

                                    if device.id != existingDevices.last?.id {
                                        Divider()
                                    }
                                }
                            }
                            .background(.regularMaterial)
                            .clipShape(RoundedRectangle(cornerRadius: 10))
                        }
                    }

                    // Register new
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Register as new device")
                            .font(.subheadline.weight(.semibold))

                        TextField("Device name", text: $deviceName)
                            .textFieldStyle(.roundedBorder)
                            .autocorrectionDisabled()
                            .submitLabel(.done)
                            .onSubmit { registerNew() }

                        Text("Give this device a name, or leave blank to use the default.")
                            .font(.caption)
                            .foregroundStyle(.secondary)

                        Button {
                            registerNew()
                        } label: {
                            if isCreating {
                                ProgressView()
                                    .frame(maxWidth: .infinity)
                            } else {
                                Text("Register")
                                    .frame(maxWidth: .infinity)
                            }
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(isBusy)
                    }
                }
                .padding()
            }
            .navigationBarTitleDisplayMode(.inline)
            .interactiveDismissDisabled()
        }
    }

    // MARK: - Actions

    private func claimDevice(_ device: Device) {
        guard !isBusy else { return }
        claimingId = device.id
        errorMessage = nil

        Task {
            do {
                let claimed = try await deviceManager.claimExistingDevice(id: device.id)
                onResolved(claimed.id)
            } catch {
                errorMessage = (error as? APIError)?.message ?? "Failed to claim device"
            }
            claimingId = nil
        }
    }

    private func registerNew() {
        guard !isBusy else { return }
        isCreating = true
        errorMessage = nil

        Task {
            do {
                let device = try await deviceManager.registerNewDevice(
                    name: deviceName.isEmpty ? nil : deviceName)
                onResolved(device.id)
            } catch {
                errorMessage = (error as? APIError)?.message ?? "Failed to register device"
            }
            isCreating = false
        }
    }
}

// MARK: - Device Row

private struct DeviceRow: View {
    let device: Device
    let isBusy: Bool
    let isClaiming: Bool
    let onClaim: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    Text(device.name)
                        .font(.subheadline.weight(.medium))

                    Text(device.deviceType)
                        .font(.caption2.weight(.medium))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(.secondary.opacity(0.15))
                        .clipShape(Capsule())
                }

                Text(lastSeenText)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            Button {
                onClaim()
            } label: {
                if isClaiming {
                    ProgressView()
                        .controlSize(.small)
                } else {
                    Text("Use this")
                        .font(.subheadline)
                }
            }
            .buttonStyle(.bordered)
            .disabled(isBusy)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
    }

    private var lastSeenText: String {
        guard let lastSeen = device.lastSeenAt else { return "Never seen" }

        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: lastSeen) else { return "Never seen" }

        let seconds = Int(Date.now.timeIntervalSince(date))
        if seconds < 60 { return "Just now" }
        let minutes = seconds / 60
        if minutes < 60 { return "\(minutes)m ago" }
        let hours = minutes / 60
        if hours < 24 { return "\(hours)h ago" }
        let days = hours / 24
        if days < 30 { return "\(days)d ago" }
        let months = days / 30
        if months < 12 { return "\(months)mo ago" }
        return "\(months / 12)y ago"
    }
}

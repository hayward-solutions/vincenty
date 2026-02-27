import SwiftUI

/// Root view — gates between onboarding, login, and the main app.
/// Also manages the device resolution + WebSocket connection lifecycle,
/// mirroring the web client's `WebSocketProvider`.
struct ContentView: View {
    @Environment(AuthManager.self) private var auth
    @Environment(NetworkMonitor.self) private var network
    @Environment(DeviceManager.self) private var deviceManager
    @Environment(WebSocketService.self) private var webSocket
    @Environment(SyncManager.self) private var syncManager

    var body: some View {
        ZStack {
            if auth.isLoading {
                // Launch screen equivalent
                VStack(spacing: 16) {
                    Image(systemName: "antenna.radiowaves.left.and.right")
                        .font(.system(size: 48))
                        .foregroundStyle(.tint)
                    ProgressView()
                }
            } else if !auth.hasServerURL {
                ServerURLView()
            } else if !auth.isAuthenticated {
                LoginView()
            } else {
                MainTabView()
            }
        }
        .animation(.default, value: auth.isAuthenticated)
        .animation(.default, value: auth.isLoading)
        // Offline banner
        .overlay(alignment: .top) {
            if !network.isConnected {
                StatusBanner(icon: "wifi.slash", message: "No connection", color: .red)
                    .padding(.top, 4)
                    .transition(.move(edge: .top).combined(with: .opacity))
            }
        }
        // WebSocket connection + sync status indicators
        .overlay(alignment: .bottom) {
            VStack(spacing: 6) {
                if auth.isAuthenticated && webSocket.connectionState == .connecting {
                    StatusBanner(
                        icon: nil, message: "Connecting...",
                        color: .orange, showSpinner: true)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                }

                if syncManager.isSyncing {
                    StatusBanner(
                        icon: nil, message: "Syncing offline changes...",
                        color: .blue, showSpinner: true)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                } else if syncManager.pendingCount != 0 && !network.isConnected {
                    StatusBanner(
                        icon: "arrow.triangle.2.circlepath",
                        message: "\(syncManager.pendingCount) pending",
                        color: .gray)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                }
            }
            .padding(.bottom, 56)
        }
        // Device enrolment sheet — non-dismissible
        .sheet(isPresented: showEnrolmentSheet) {
            if let devices = deviceManager.pendingEnrolment {
                DeviceEnrolmentSheet(existingDevices: devices) { deviceId in
                    deviceManager.resolveEnrolment(deviceId)
                    webSocket.connect(deviceId: deviceId)
                }
                .environment(deviceManager)
                .interactiveDismissDisabled()
            }
        }
        // Auth lifecycle: connect/disconnect WS + resolve device
        .onChange(of: auth.isAuthenticated) { oldValue, isAuthenticated in
            if isAuthenticated {
                startConnection()
            } else {
                // Clean disconnect on logout
                webSocket.disconnect()
                deviceManager.clearDevice()
            }
        }
        // Sync offline actions on network reconnect
        .onChange(of: network.isConnected) { wasConnected, isConnected in
            if !wasConnected && isConnected && auth.isAuthenticated {
                Task { await syncManager.syncPendingActions() }
            }
        }
    }

    // MARK: - Computed

    private var showEnrolmentSheet: Binding<Bool> {
        Binding(
            get: { deviceManager.pendingEnrolment != nil },
            set: { _ in } // Non-dismissible — resolved via callback
        )
    }

    // MARK: - Connection Lifecycle

    /// Mirrors the web client's `useEffect` in `WebSocketProvider`:
    /// 1. Resolve device (may show enrolment sheet)
    /// 2. Connect WebSocket with resolved device ID
    /// 3. Wire up stale device re-resolution callback
    private func startConnection() {
        // Set up the stale device callback on the WS service
        webSocket.onDeviceNeedsResolve = { [deviceManager] () -> String? in
            return await deviceManager.ensureDevice()
        }

        Task {
            let devId = await deviceManager.ensureDevice()
            if let devId {
                webSocket.connect(deviceId: devId)
            }
            // If devId is nil, the enrolment sheet is showing.
            // Connection will happen via resolveEnrolment callback.
        }
    }
}

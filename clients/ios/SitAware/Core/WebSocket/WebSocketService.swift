import Foundation
#if os(iOS)
import ActivityKit
#endif

/// Message handler closure type — receives (type, raw payload dictionary).
typealias WSMessageHandler = @Sendable (String, AnyCodable?) -> Void

/// Connection state for the WebSocket.
enum WSConnectionState: Sendable, Equatable {
    case disconnected
    case connecting
    case connected
}

/// Manages the WebSocket connection to the SitAware server.
///
/// Mirrors the web client's `WebSocketProvider`:
/// - Connects with JWT + device_id as query params
/// - Exponential backoff reconnect (1s → 30s max)
/// - Stale device_id detection with one retry (clear + re-resolve)
/// - Fan-out message dispatch to registered subscribers
/// - Sends/receives `WSEnvelope` JSON envelopes
@Observable @MainActor
final class WebSocketService {

    // MARK: - Public State

    private(set) var connectionState: WSConnectionState = .disconnected
    private(set) var deviceId: String?

    // MARK: - Configuration

    private static let maxBackoff: TimeInterval = 30.0
    private static let initialBackoff: TimeInterval = 1.0

    // MARK: - Private

    private var webSocketTask: URLSessionWebSocketTask?
    private var handlers: [UUID: WSMessageHandler] = [:]
    private var backoff: TimeInterval = WebSocketService.initialBackoff
    private var reconnectTask: Task<Void, Never>?
    private var receiveTask: Task<Void, Never>?
    private var isMounted = false
    /// Guard: only attempt device re-resolution once per connect cycle.
    private var retriedDevice = false

    // MARK: - Live Activity

    #if os(iOS)
    private var connectionActivity: Activity<ConnectionActivityAttributes>?
    #endif

    /// Callback invoked when the WS needs a fresh device ID (stale ID rejected).
    /// Set by the DeviceManager. Returns nil if user needs to choose via enrolment sheet.
    var onDeviceNeedsResolve: (@MainActor () async -> String?)?

    private let decoder: JSONDecoder = {
        let d = JSONDecoder()
        d.keyDecodingStrategy = .convertFromSnakeCase
        return d
    }()

    private let encoder: JSONEncoder = {
        let e = JSONEncoder()
        e.keyEncodingStrategy = .convertToSnakeCase
        return e
    }()

    // MARK: - Lifecycle

    /// Start the WebSocket service. Call after authentication + device resolution.
    func connect(deviceId: String) {
        guard isMounted else {
            isMounted = true
            self.deviceId = deviceId
            connectInternal(deviceId: deviceId)
            return
        }
        self.deviceId = deviceId
        connectInternal(deviceId: deviceId)
    }

    /// Cleanly disconnect and stop all reconnection attempts.
    func disconnect() {
        isMounted = false
        reconnectTask?.cancel()
        reconnectTask = nil
        receiveTask?.cancel()
        receiveTask = nil
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        connectionState = .disconnected
        backoff = Self.initialBackoff
        retriedDevice = false
        AppLogger.shared.log(.info, .ws, "Disconnected (clean)")
        endLiveActivity()
    }

    // MARK: - Subscribe / Send

    /// Register a handler for incoming messages. Returns an unsubscribe closure.
    @discardableResult
    func subscribe(_ handler: @escaping @Sendable (String, AnyCodable?) -> Void) -> () -> Void {
        let id = UUID()
        handlers[id] = handler
        return { [weak self] in
            Task { @MainActor in
                self?.handlers.removeValue(forKey: id)
            }
        }
    }

    /// Send a typed message to the server.
    func send(type: String, payload: (any Encodable & Sendable)? = nil) {
        guard let webSocketTask, webSocketTask.state == .running else { return }

        // Build envelope as dictionary and encode
        var envelope: [String: Any] = ["type": type]
        if let payload {
            // Encode payload to JSON, then decode as dictionary
            if let data = try? encoder.encode(AnyEncodableValue(payload)),
               let dict = try? JSONSerialization.jsonObject(with: data) {
                envelope["payload"] = dict
            }
        }

        guard let data = try? JSONSerialization.data(withJSONObject: envelope),
              let string = String(data: data, encoding: .utf8)
        else { return }

        webSocketTask.send(.string(string)) { error in
            if let error {
                AppLogger.shared.error(.ws, "Send error: \(error.localizedDescription)")
            }
        }
    }

    // MARK: - Connection

    private func connectInternal(deviceId: String) {
        guard isMounted else { return }
        guard let baseURL = KeychainStore.shared.serverURL, !baseURL.isEmpty,
              let token = KeychainStore.shared.accessToken
        else { return }

        // Cancel existing
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        receiveTask?.cancel()

        connectionState = .connecting
        AppLogger.shared.log(.info, .ws, "Connecting (device: \(deviceId.prefix(8))…)")

        // Build WS URL: ws(s)://host/api/v1/ws?token=<jwt>&device_id=<uuid>
        let wsBase = baseURL
            .replacingOccurrences(of: "https://", with: "wss://")
            .replacingOccurrences(of: "http://", with: "ws://")

        guard var components = URLComponents(string: "\(wsBase)\(Endpoints.ws)") else {
            connectionState = .disconnected
            return
        }

        components.queryItems = [
            URLQueryItem(name: "token", value: token),
            URLQueryItem(name: "device_id", value: deviceId),
            URLQueryItem(name: "app_version", value: BuildInfo.version),
        ]

        guard let url = components.url else {
            connectionState = .disconnected
            return
        }

        let task = URLSession.shared.webSocketTask(with: url)
        self.webSocketTask = task
        task.resume()

        // Track whether we got a successful open
        var didOpen = false

        // Receive loop — connection is confirmed on first successful receive.
        // The server always sends a "connected" ack immediately after upgrade,
        // so the first receive doubles as the open signal.
        receiveTask = Task { [weak self] in
            while !Task.isCancelled {
                do {
                    let message = try await task.receive()

                    // First successful receive confirms the connection is open
                    if !didOpen {
                        didOpen = true
                        await MainActor.run { [weak self] in
                            guard let self, self.isMounted else { return }
                            self.connectionState = .connected
                            self.backoff = Self.initialBackoff
                            self.retriedDevice = false
                            AppLogger.shared.log(.info, .ws, "Connected")
                            self.startOrUpdateLiveActivity(isConnected: true)
                        }
                    }

                    await MainActor.run { [weak self] in
                        self?.handleMessage(message)
                    }
                } catch {
                    // Connection closed or failed to open
                    break
                }
            }

            self?.handleDisconnect(task: task, didOpen: didOpen, deviceId: deviceId)
        }
    }

    private func handleMessage(_ message: URLSessionWebSocketTask.Message) {
        let data: Data
        switch message {
        case .string(let text):
            guard let d = text.data(using: .utf8) else { return }
            data = d
        case .data(let d):
            data = d
        @unknown default:
            return
        }

        guard let envelope: WSEnvelope = try? decoder.decode(WSEnvelope.self, from: data) else {
            AppLogger.shared.log(.warning, .ws, "Failed to decode envelope",
                                 detail: String(data: data, encoding: .utf8))
            return
        }

        // Fan out to all subscribers
        let type = envelope.type
        let payload = envelope.payload
        AppLogger.shared.log(.debug, .ws, "Received: \(type)")
        for handler in handlers.values {
            handler(type, payload)
        }
    }

    // MARK: - Live Activity

    #if os(iOS)
    /// Start a new Live Activity (or update the existing one) to reflect connection state.
    private func startOrUpdateLiveActivity(isConnected: Bool) {
        guard ActivityAuthorizationInfo().areActivitiesEnabled else { return }
        let serverURL = KeychainStore.shared.serverURL ?? ""
        let state = ConnectionActivityAttributes.ContentState(
            isConnected: isConnected,
            connectedSince: isConnected ? Date() : nil)

        if let activity = connectionActivity {
            Task {
                await activity.update(.init(state: state, staleDate: nil))
            }
        } else {
            let attributes = ConnectionActivityAttributes(serverURL: serverURL)
            do {
                let activity = try Activity<ConnectionActivityAttributes>.request(
                    attributes: attributes,
                    content: .init(state: state, staleDate: nil),
                    pushType: nil)
                connectionActivity = activity
                AppLogger.shared.log(.info, .ws, "Live Activity started")
            } catch {
                AppLogger.shared.error(.ws, "Live Activity failed to start: \(error)")
            }
        }
    }

    /// End the Live Activity with a brief "Disconnected" snapshot then dismiss.
    private func endLiveActivity() {
        guard let activity = connectionActivity else { return }
        connectionActivity = nil
        let finalState = ConnectionActivityAttributes.ContentState(
            isConnected: false,
            connectedSince: nil)
        Task {
            await activity.end(.init(state: finalState, staleDate: nil), dismissalPolicy: .after(.now + 3))
            AppLogger.shared.log(.info, .ws, "Live Activity ended")
        }
    }
    #else
    private func startOrUpdateLiveActivity(isConnected: Bool) {}
    private func endLiveActivity() {}
    #endif

    private func handleDisconnect(task: URLSessionWebSocketTask, didOpen: Bool, deviceId: String) {
        guard isMounted else { return }

        // Only handle if this is still our current task
        guard task === webSocketTask else { return }

        webSocketTask = nil
        connectionState = .disconnected
        AppLogger.shared.log(.warning, .ws, didOpen ? "Disconnected" : "Connection failed (never opened)")
        startOrUpdateLiveActivity(isConnected: false)

        // Stale device_id detection:
        // If the server rejected connection before it ever opened, clear stored device
        // and re-resolve once before falling back to normal backoff.
        if !didOpen && !retriedDevice {
            retriedDevice = true
            KeychainStore.shared.deviceId = nil
            AppLogger.shared.log(.info, .ws, "Stale device ID detected — re-resolving")

            Task { @MainActor [weak self] in
                guard let self, self.isMounted else { return }
                if let resolve = self.onDeviceNeedsResolve {
                    if let newDevId = await resolve() {
                        self.deviceId = newDevId
                        self.connectInternal(deviceId: newDevId)
                    }
                }
            }
            return
        }

        // Reconnect with exponential backoff
        let delay = backoff
        backoff = min(backoff * 2, Self.maxBackoff)
        AppLogger.shared.log(.info, .ws, "Reconnecting in \(Int(delay))s…")

        reconnectTask = Task { @MainActor [weak self] in
            try? await Task.sleep(for: .seconds(delay))
            guard let self, self.isMounted, !Task.isCancelled else { return }
            guard KeychainStore.shared.accessToken != nil else { return }

            let devId = self.deviceId ?? deviceId
            self.connectInternal(deviceId: devId)
        }
    }
}

// MARK: - Encoding Helper

/// Type-erased Encodable for building WS send payloads.
private struct AnyEncodableValue: Encodable {
    private let _encode: (Encoder) throws -> Void

    init(_ value: any Encodable) {
        self._encode = value.encode
    }

    func encode(to encoder: Encoder) throws {
        try _encode(encoder)
    }
}


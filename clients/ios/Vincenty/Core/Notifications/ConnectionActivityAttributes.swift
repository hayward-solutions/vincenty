#if os(iOS)
import ActivityKit
import Foundation

/// Attributes for the persistent "Connected to Vincenty" Live Activity.
///
/// This type is compiled into both the main app target and the
/// VincentyWidgets extension so that the app can start/update activities
/// and the widget extension can render them.
struct ConnectionActivityAttributes: ActivityAttributes {

    // MARK: - Static data (set once at activity start)

    /// The server base URL displayed in the Live Activity UI.
    let serverURL: String

    // MARK: - Dynamic state (updated as connection status changes)

    struct ContentState: Codable, Hashable {
        /// Whether the WebSocket is currently connected.
        let isConnected: Bool
        /// When the current connection was established (nil when disconnected).
        let connectedSince: Date?
    }
}
#endif

import CoreLocation
import Foundation

/// Manages GPS location updates and broadcasts them to the server via WebSocket.
///
/// iOS-specific implementation — the web client uses `navigator.geolocation.watchPosition`.
/// This uses `CLLocationManager` with:
/// - "When In Use" permission for foreground use
/// - "Always" permission + `allowsBackgroundLocationUpdates` for background tracking
/// - Activity type `.otherNavigation` for good accuracy with power savings
/// - Distance filter of 5m to avoid spamming trivial updates
/// - Broadcasts location as `WSLocationUpdate` via the WebSocket service
@Observable @MainActor
final class LocationSharingManager: NSObject, @unchecked Sendable {

    // MARK: - Public State

    /// Whether location sharing is currently active.
    private(set) var isSharing = false

    /// Current authorization status.
    private(set) var authorizationStatus: CLAuthorizationStatus = .notDetermined

    /// The latest known position.
    private(set) var currentPosition: (lat: Double, lng: Double, heading: Double?)?

    /// Any location error message.
    private(set) var error: String?

    // MARK: - Configuration

    /// Minimum distance (meters) between updates to trigger a broadcast.
    var distanceFilter: CLLocationDistance = 5.0

    /// Desired accuracy.
    var desiredAccuracy: CLLocationAccuracy = kCLLocationAccuracyBest

    // MARK: - Private

    private let locationManager = CLLocationManager()
    private weak var webSocket: WebSocketService?
    private var deviceId: String?

    // MARK: - Init

    override init() {
        super.init()
        locationManager.delegate = self
        locationManager.desiredAccuracy = desiredAccuracy
        locationManager.distanceFilter = distanceFilter
        locationManager.activityType = .otherNavigation
        locationManager.pausesLocationUpdatesAutomatically = false

        authorizationStatus = locationManager.authorizationStatus
    }

    // MARK: - Public API

    /// Start sharing location. Requests permission if needed.
    func startSharing(webSocket: WebSocketService, deviceId: String) {
        self.webSocket = webSocket
        self.deviceId = deviceId
        self.error = nil

        switch authorizationStatus {
        case .notDetermined:
            AppLogger.shared.log(.info, .location, "Requesting location permission")
            locationManager.requestWhenInUseAuthorization()
        case .authorizedWhenInUse, .authorizedAlways:
            beginUpdates()
        case .denied, .restricted:
            self.error = "Location access denied. Enable in Settings."
            AppLogger.shared.log(.warning, .location, "Location permission denied")
        @unknown default:
            self.error = "Unknown authorization status."
            AppLogger.shared.log(.warning, .location, "Unknown location authorization status")
        }
    }

    /// Stop sharing location.
    func stopSharing() {
        locationManager.stopUpdatingLocation()
        isSharing = false
        currentPosition = nil
        AppLogger.shared.log(.info, .location, "Location sharing stopped")
    }

    /// Request "Always" authorization for background tracking.
    /// Should only be called after user explicitly enables background mode.
    func requestAlwaysAuthorization() {
        locationManager.requestAlwaysAuthorization()
    }

    /// Enable background location updates.
    /// Must be called after receiving "Always" authorization.
    func enableBackgroundUpdates() {
        locationManager.allowsBackgroundLocationUpdates = true
        locationManager.showsBackgroundLocationIndicator = true
    }

    /// Disable background location updates.
    func disableBackgroundUpdates() {
        locationManager.allowsBackgroundLocationUpdates = false
        locationManager.showsBackgroundLocationIndicator = false
    }

    // MARK: - Private

    private func beginUpdates() {
        locationManager.desiredAccuracy = desiredAccuracy
        locationManager.distanceFilter = distanceFilter
        locationManager.startUpdatingLocation()
        isSharing = true
        AppLogger.shared.log(.info, .location, "Location sharing started (filter: \(Int(distanceFilter))m)")
    }

    /// Send the current location to the server via WebSocket.
    private func broadcastLocation(_ location: CLLocation) {
        guard let deviceId else { return }

        let update = WSLocationUpdate(
            deviceId: deviceId,
            lat: location.coordinate.latitude,
            lng: location.coordinate.longitude,
            altitude: location.altitude > -1 ? location.altitude : nil,
            heading: location.course >= 0 ? location.course : nil,
            speed: location.speed >= 0 ? location.speed : nil,
            accuracy: location.horizontalAccuracy >= 0 ? location.horizontalAccuracy : nil)

        webSocket?.send(type: WSMessageType.locationUpdate, payload: update)
    }
}

// MARK: - CLLocationManagerDelegate

extension LocationSharingManager: CLLocationManagerDelegate {

    nonisolated func locationManager(
        _ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]
    ) {
        guard let latest = locations.last else { return }

        Task { @MainActor in
            // Update local state
            let heading: Double? = latest.course >= 0 ? latest.course : nil
            self.currentPosition = (
                lat: latest.coordinate.latitude,
                lng: latest.coordinate.longitude,
                heading: heading
            )

            // Broadcast to server
            self.broadcastLocation(latest)
            let acc = latest.horizontalAccuracy >= 0 ? String(format: "±%.0fm", latest.horizontalAccuracy) : ""
            AppLogger.shared.log(.debug, .location,
                String(format: "Broadcast: %.5f, %.5f \(acc)",
                       latest.coordinate.latitude, latest.coordinate.longitude))
        }
    }

    nonisolated func locationManager(
        _ manager: CLLocationManager, didFailWithError error: Error
    ) {
        Task { @MainActor in
            if let clError = error as? CLError {
                switch clError.code {
                case .denied:
                    self.error = "Location access denied."
                    AppLogger.shared.log(.warning, .location, "Location access denied by user")
                    self.stopSharing()
                case .locationUnknown:
                    // Transient — ignore
                    break
                default:
                    self.error = "Location error: \(clError.localizedDescription)"
                    AppLogger.shared.log(.error, .location, "Location error",
                                        detail: clError.localizedDescription)
                }
            }
        }
    }

    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        // Capture needed state before hopping to the main actor to avoid sending non-Sendable values.
        let status = manager.authorizationStatus

        Task { @MainActor in
            self.authorizationStatus = status

            switch status {
            case .authorizedWhenInUse, .authorizedAlways:
                let label = status == .authorizedAlways ? "Always" : "WhenInUse"
                AppLogger.shared.log(.info, .location, "Location permission granted: \(label)")
                if self.webSocket != nil && !self.isSharing {
                    self.beginUpdates()
                }
            case .denied, .restricted:
                self.error = "Location access denied."
                AppLogger.shared.log(.warning, .location, "Location permission denied/restricted")
                self.stopSharing()
            default:
                break
            }
        }
    }
}

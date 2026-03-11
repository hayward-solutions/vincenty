import Foundation
import MapLibre
import SwiftUI

/// Central view model for the map screen.
///
/// Mirrors the web client's `page.tsx` state + the hooks it orchestrates:
/// - Fetches map settings from the API
/// - Subscribes to WebSocket for real-time location updates
/// - Manages filtered/displayed locations
/// - Tracks camera state (bearing, pitch, zoom) for controls
/// - Manages panel toggles (filter, measure, draw, replay)
@Observable @MainActor
final class MapViewModel {

    // MARK: - Map State

    /// The underlying MapLibre map view reference (set after style loads).
    private(set) var mapView: MLNMapView?
    private(set) var isMapReady = false

    /// Server map configuration.
    private(set) var mapSettings: MapSettings?
    private(set) var isLoadingSettings = true

    // MARK: - Camera

    private(set) var bearing: Double = 0
    private(set) var pitch: Double = 0
    private(set) var zoomLevel: Double = 10

    // MARK: - Locations

    /// All live locations keyed by device ID (from WS + REST).
    private(set) var allLocations: [String: UserLocation] = [:]

    /// Filtered locations for display.
    var displayLocations: [UserLocation] {
        var result = Array(allLocations.values)

        // Filter by primary-only
        if primaryOnly {
            result = result.filter(\.isPrimary)
        }

        // Filter by selected groups
        if !selectedGroupIds.isEmpty {
            result = result.filter { selectedGroupIds.contains($0.groupId) }
        }

        // Filter by selected users
        if !selectedUserIds.isEmpty {
            result = result.filter { selectedUserIds.contains($0.userId) }
        }

        return result
    }

    // MARK: - User's Position

    /// The current user's self-reported position (from iOS location).
    var selfPosition: (lat: Double, lng: Double, heading: Double?)? = nil
    var showSelf = true

    // MARK: - Groups / Users

    private(set) var groups: [Group] = []
    private(set) var users: [User] = []

    // MARK: - Filter State

    var showFilterPanel = false
    var showDrawings = true
    var primaryOnly = false
    var selectedGroupIds: Set<String> = []
    var selectedUserIds: Set<String> = []

    // MARK: - Panel Toggles

    var showReplayPanel = false
    var showMeasurePanel = false
    var showDrawPanel = false

    // MARK: - Tracking

    var isTracking = false

    // MARK: - Terrain

    var terrainEnabled = false
    var terrainAvailable: Bool { !(mapSettings?.terrainUrl.isEmpty ?? true) }

    // MARK: - Dependencies

    private let api = APIClient.shared
    private var wsUnsubscribe: (() -> Void)?

    // MARK: - Lifecycle

    /// Load map settings and group/user lists from the API.
    /// On first load, shows a loading spinner. On re-entry (tab switch back),
    /// refreshes data silently so the existing map is not destroyed.
    func loadInitialData() async {
        if mapSettings == nil {
            isLoadingSettings = true
        }
        async let settingsTask: MapSettings? = {
            do { return try await api.get(Endpoints.mapSettings) }
            catch { return nil }
        }()
        async let groupsTask: [Group] = {
            do {
                let response: ListResponse<Group> = try await api.get(Endpoints.groups)
                return response.data
            } catch { return [] }
        }()
        async let usersTask: [User] = {
            do {
                let response: ListResponse<User> = try await api.get(
                    Endpoints.users, params: ["page_size": "200"])
                return response.data
            } catch { return [] }
        }()

        let (settings, groupsList, usersList) = await (settingsTask, groupsTask, usersTask)
        mapSettings = settings
        groups = groupsList
        users = usersList
        isLoadingSettings = false
    }

    /// Subscribe to WebSocket location updates.
    func subscribeToLocations(webSocket: WebSocketService) {
        wsUnsubscribe = webSocket.subscribe { [weak self] type, data in
            Task { @MainActor [weak self] in
                self?.handleWSMessage(type: type, data: data)
            }
        }
    }

    /// Clean up subscriptions.
    func unsubscribe() {
        wsUnsubscribe?()
        wsUnsubscribe = nil
    }

    // MARK: - Map Ready Callback

    func onMapReady(_ map: MLNMapView) {
        self.mapView = map
        self.isMapReady = true
    }

    func onCameraChanged(bearing: Double, pitch: Double, zoom: Double) {
        self.bearing = bearing
        self.pitch = pitch
        self.zoomLevel = zoom
    }

    /// Called when the user begins a gesture-driven camera move. Disables tracking
    /// so the camera doesn't snap back on the next GPS update.
    func onUserDragBegan() {
        isTracking = false
    }

    // MARK: - Camera Controls

    func zoomIn() {
        guard let mapView else { return }
        mapView.setZoomLevel(mapView.zoomLevel + 1, animated: true)
    }

    func zoomOut() {
        guard let mapView else { return }
        mapView.setZoomLevel(mapView.zoomLevel - 1, animated: true)
    }

    func resetNorth() {
        guard let mapView else { return }
        mapView.setDirection(0, animated: true)
        let camera = mapView.camera
        camera.pitch = 0
        mapView.setCamera(camera, animated: true)
    }

    func toggleTerrain() {
        guard let mapView, terrainAvailable else { return }
        terrainEnabled.toggle()

        if terrainEnabled {
            // Enable terrain with exaggeration
            // Note: terrain API usage depends on MapLibre Native version
            let camera = mapView.camera
            camera.pitch = 50
            mapView.fly(to: camera, withDuration: 0.5, completionHandler: nil)
        } else {
            let camera = mapView.camera
            camera.pitch = 0
            mapView.fly(to: camera, withDuration: 0.5, completionHandler: nil)
        }
    }

    func flyToSelf() {
        guard let mapView, let pos = selfPosition else { return }
        let camera = MLNMapCamera(
            lookingAtCenter: CLLocationCoordinate2D(latitude: pos.lat, longitude: pos.lng),
            altitude: mapView.camera.altitude,
            pitch: mapView.camera.pitch,
            heading: mapView.direction)
        let zoom = max(mapView.zoomLevel, 14)
        mapView.setZoomLevel(zoom, animated: false)
        mapView.fly(to: camera, withDuration: 1.0, completionHandler: nil)
        isTracking = true
    }

    /// Update camera to follow self position while tracking.
    func updateTrackingIfNeeded() {
        guard isTracking, let mapView, let pos = selfPosition else { return }
        let camera = MLNMapCamera(
            lookingAtCenter: CLLocationCoordinate2D(latitude: pos.lat, longitude: pos.lng),
            altitude: mapView.camera.altitude,
            pitch: mapView.camera.pitch,
            heading: mapView.direction)
        mapView.setCamera(camera, withDuration: 0.3, animationTimingFunction: nil)
    }

    // MARK: - Panel Toggles (mutual exclusion)

    func toggleFilter() {
        showFilterPanel.toggle()
        if showFilterPanel {
            showReplayPanel = false
            showMeasurePanel = false
            showDrawPanel = false
        }
    }

    func toggleReplay() {
        showReplayPanel.toggle()
        if showReplayPanel {
            showFilterPanel = false
            showMeasurePanel = false
            showDrawPanel = false
        }
    }

    func toggleMeasure() {
        showMeasurePanel.toggle()
        if showMeasurePanel {
            showFilterPanel = false
            showReplayPanel = false
            showDrawPanel = false
        }
    }

    func toggleDraw() {
        showDrawPanel.toggle()
        if showDrawPanel {
            showFilterPanel = false
            showReplayPanel = false
            showMeasurePanel = false
        }
    }

    // MARK: - WebSocket Message Handling

    // Typed envelope wrappers — one per message type — for direct single-pass decoding.
    private struct BroadcastEnvelope: Decodable {
        let payload: WSLocationBroadcast
    }
    private struct SnapshotEnvelope: Decodable {
        let payload: WSLocationSnapshot
    }

    private func handleWSMessage(type: String, data: Data) {
        switch type {
        case WSMessageType.locationBroadcast:
            do {
                let envelope = try JSONDecoder.snakeCase.decode(BroadcastEnvelope.self, from: data)
                let location = UserLocation(from: envelope.payload)
                allLocations[location.deviceId] = location
            } catch {
                AppLogger.shared.error(.ws, "location_broadcast decode failed: \(error)")
            }

        case WSMessageType.locationSnapshot:
            do {
                let envelope = try JSONDecoder.snakeCase.decode(SnapshotEnvelope.self, from: data)
                for broadcast in envelope.payload.locations {
                    let location = UserLocation(from: broadcast)
                    allLocations[location.deviceId] = location
                }
            } catch {
                AppLogger.shared.error(.ws, "location_snapshot decode failed: \(error)")
            }

        case WSMessageType.connected:
            break

        default:
            break
        }
    }
}



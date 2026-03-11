import Foundation

// MARK: - WebSocket Envelope

/// All WebSocket messages use this envelope: `{ type: string, payload: ... }`.
struct WSEnvelope: Codable, Sendable {
    let type: String
    let payload: AnyCodable?
}

// MARK: - Client -> Server

/// Client -> Server: send current GPS position.
struct WSLocationUpdate: Codable, Sendable {
    let deviceId: String
    let lat: Double
    let lng: Double
    var altitude: Double?
    var heading: Double?
    var speed: Double?
    var accuracy: Double?
}

// MARK: - Server -> Client

/// Server -> Client: another device's real-time position update.
struct WSLocationBroadcast: Codable, Sendable {
    let userId: String
    let username: String
    let displayName: String
    let deviceId: String
    let deviceName: String
    let isPrimary: Bool
    let groupId: String
    let lat: Double
    let lng: Double
    var altitude: Double?
    var heading: Double?
    var speed: Double?
    let timestamp: String
}

/// Server -> Client: bulk snapshot of all group member positions on connect.
struct WSLocationSnapshot: Codable, Sendable {
    let groupId: String
    let locations: [WSLocationBroadcast]
}

/// Server -> Client: connection acknowledgement with group list.
struct WSConnected: Codable, Sendable {
    let userId: String
    let groups: [WSGroupInfo]
}

struct WSGroupInfo: Codable, Sendable {
    let id: String
    let name: String
}

/// Server -> Client: error message.
struct WSError: Codable, Sendable {
    let message: String
}

// MARK: - WebSocket Message Types

/// All known WebSocket message type strings.
enum WSMessageType {
    // Client -> Server
    static let locationUpdate = "location_update"

    // Server -> Client
    static let connected = "connected"
    static let locationBroadcast = "location_broadcast"
    static let locationSnapshot = "location_snapshot"
    static let messageNew = "message_new"
    static let drawingUpdated = "drawing_updated"
    static let error = "error"
}

// MARK: - User Location (runtime model)

/// Consolidated live location for a device, used by the map view.
/// Built from WSLocationBroadcast data.
struct UserLocation: Sendable, Identifiable, Equatable {
    var id: String { deviceId }

    let userId: String
    let username: String
    let displayName: String
    let deviceId: String
    let deviceName: String
    let isPrimary: Bool
    let groupId: String
    let lat: Double
    let lng: Double
    var altitude: Double?
    var heading: Double?
    var speed: Double?
    let timestamp: Date

    init(from broadcast: WSLocationBroadcast) {
        self.userId = broadcast.userId
        self.username = broadcast.username
        self.displayName = broadcast.displayName
        self.deviceId = broadcast.deviceId
        self.deviceName = broadcast.deviceName
        self.isPrimary = broadcast.isPrimary
        self.groupId = broadcast.groupId
        self.lat = broadcast.lat
        self.lng = broadcast.lng
        self.altitude = broadcast.altitude
        self.heading = broadcast.heading
        self.speed = broadcast.speed
        self.timestamp = ISO8601DateFormatter().date(from: broadcast.timestamp) ?? Date()
    }

    init(from entry: LatestLocationEntry, groupId: String = "") {
        self.userId = entry.userId
        self.username = entry.username
        self.displayName = entry.displayName
        self.deviceId = entry.deviceId
        self.deviceName = entry.deviceName
        self.isPrimary = entry.isPrimary
        self.groupId = groupId
        self.lat = entry.lat
        self.lng = entry.lng
        self.altitude = entry.altitude
        self.heading = entry.heading
        self.speed = entry.speed
        self.timestamp = ISO8601DateFormatter().date(from: entry.recordedAt) ?? Date()
    }
}

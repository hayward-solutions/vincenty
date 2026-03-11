import Foundation

// MARK: - Location History Entry

/// A single recorded location point from the history API.
struct LocationHistoryEntry: Codable, Sendable, Identifiable, Equatable {
    var id: String { "\(userId):\(deviceId):\(recordedAt)" }

    let userId: String
    let deviceId: String
    let deviceName: String
    let username: String
    let displayName: String
    let lat: Double
    let lng: Double
    var altitude: Double?
    var heading: Double?
    var speed: Double?
    let recordedAt: String
}

// MARK: - Latest Location Entry

/// Latest known position for a device from `GET /api/v1/locations`.
struct LatestLocationEntry: Codable, Sendable {
    let userId: String
    let deviceId: String
    let deviceName: String
    let isPrimary: Bool
    let username: String
    let displayName: String
    let lat: Double
    let lng: Double
    var altitude: Double?
    var heading: Double?
    var speed: Double?
    let recordedAt: String
}

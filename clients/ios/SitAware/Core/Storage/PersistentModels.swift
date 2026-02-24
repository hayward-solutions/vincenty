import Foundation
import SwiftData

// MARK: - SwiftData Persistent Models
//
// These mirror the API Codable models but are stored locally for offline support.
// Each model tracks `lastSyncedAt` for staleness detection and uses the server's
// `id` as the primary key. On sync, server-wins conflict resolution is used:
// if the server's `updatedAt` is newer, the local record is overwritten.

/// Cached user profile for offline display.
@Model
final class CachedUser {
    @Attribute(.unique) var id: String
    var username: String
    var email: String
    var displayName: String
    var avatarUrl: String
    var markerIcon: String
    var markerColor: String
    var isAdmin: Bool
    var isActive: Bool
    var mfaEnabled: Bool
    var createdAt: String
    var updatedAt: String
    var lastSyncedAt: Date

    init(from user: User) {
        self.id = user.id
        self.username = user.username
        self.email = user.email
        self.displayName = user.displayName
        self.avatarUrl = user.avatarUrl
        self.markerIcon = user.markerIcon
        self.markerColor = user.markerColor
        self.isAdmin = user.isAdmin
        self.isActive = user.isActive
        self.mfaEnabled = user.mfaEnabled
        self.createdAt = user.createdAt
        self.updatedAt = user.updatedAt
        self.lastSyncedAt = Date()
    }

    /// Update from a fresher server record.
    func update(from user: User) {
        self.username = user.username
        self.email = user.email
        self.displayName = user.displayName
        self.avatarUrl = user.avatarUrl
        self.markerIcon = user.markerIcon
        self.markerColor = user.markerColor
        self.isAdmin = user.isAdmin
        self.isActive = user.isActive
        self.mfaEnabled = user.mfaEnabled
        self.updatedAt = user.updatedAt
        self.lastSyncedAt = Date()
    }

    /// Convert back to the API model for view consumption.
    func toUser() -> User {
        User(
            id: id, username: username, email: email,
            displayName: displayName, avatarUrl: avatarUrl,
            markerIcon: markerIcon, markerColor: markerColor,
            isAdmin: isAdmin, isActive: isActive, mfaEnabled: mfaEnabled,
            createdAt: createdAt, updatedAt: updatedAt)
    }
}

/// Cached group for offline display.
@Model
final class CachedGroup {
    @Attribute(.unique) var id: String
    var name: String
    var groupDescription: String
    var markerIcon: String
    var markerColor: String
    var memberCount: Int
    var createdAt: String
    var updatedAt: String
    var lastSyncedAt: Date

    init(from group: Group) {
        self.id = group.id
        self.name = group.name
        self.groupDescription = group.description
        self.markerIcon = group.markerIcon
        self.markerColor = group.markerColor
        self.memberCount = group.memberCount
        self.createdAt = group.createdAt
        self.updatedAt = group.updatedAt
        self.lastSyncedAt = Date()
    }

    func update(from group: Group) {
        self.name = group.name
        self.groupDescription = group.description
        self.markerIcon = group.markerIcon
        self.markerColor = group.markerColor
        self.memberCount = group.memberCount
        self.updatedAt = group.updatedAt
        self.lastSyncedAt = Date()
    }

    func toGroup() -> Group {
        Group(
            id: id, name: name, description: groupDescription,
            markerIcon: markerIcon, markerColor: markerColor,
            memberCount: memberCount, createdAt: createdAt, updatedAt: updatedAt)
    }
}

/// Cached message for offline display and optimistic insertion.
@Model
final class CachedMessage {
    @Attribute(.unique) var id: String
    var content: String
    var senderId: String
    var senderUsername: String
    var senderDisplayName: String
    var groupId: String?
    var recipientId: String?
    var messageType: String
    var lat: Double?
    var lng: Double?
    var createdAt: String
    var lastSyncedAt: Date
    /// Optimistic messages pending server confirmation.
    var isPending: Bool

    init(from message: MessageResponse) {
        self.id = message.id
        self.content = message.content
        self.senderId = message.senderId
        self.senderUsername = message.senderUsername
        self.senderDisplayName = message.senderDisplayName
        self.groupId = message.groupId
        self.recipientId = message.recipientId
        self.messageType = message.messageType
        self.lat = message.lat
        self.lng = message.lng
        self.createdAt = message.createdAt
        self.lastSyncedAt = Date()
        self.isPending = false
    }

    /// Create an optimistic pending message.
    init(optimisticId: String, content: String, senderId: String, senderUsername: String,
         senderDisplayName: String, groupId: String?, recipientId: String?)
    {
        self.id = optimisticId
        self.content = content
        self.senderId = senderId
        self.senderUsername = senderUsername
        self.senderDisplayName = senderDisplayName
        self.groupId = groupId
        self.recipientId = recipientId
        self.messageType = "text"
        self.createdAt = ISO8601DateFormatter().string(from: Date())
        self.lastSyncedAt = Date()
        self.isPending = true
    }
}

/// Cached drawing for offline display.
@Model
final class CachedDrawing {
    @Attribute(.unique) var id: String
    var name: String
    var userId: String
    var username: String
    var displayName: String
    /// GeoJSON stored as serialized JSON data.
    var geoJsonData: Data?
    var createdAt: String
    var updatedAt: String
    var lastSyncedAt: Date

    init(from drawing: DrawingResponse) {
        self.id = drawing.id
        self.name = drawing.name
        self.userId = drawing.userId
        self.username = drawing.username
        self.displayName = drawing.displayName
        self.createdAt = drawing.createdAt
        self.updatedAt = drawing.updatedAt
        self.lastSyncedAt = Date()

        // Serialize GeoJSON
        if let fc = drawing.geojson {
            self.geoJsonData = try? JSONEncoder().encode(fc)
        }
    }

    func update(from drawing: DrawingResponse) {
        self.name = drawing.name
        self.updatedAt = drawing.updatedAt
        self.lastSyncedAt = Date()
        if let fc = drawing.geojson {
            self.geoJsonData = try? JSONEncoder().encode(fc)
        }
    }
}

/// Cached location entry for offline replay and history.
@Model
final class CachedLocationEntry {
    @Attribute(.unique) var entryId: String
    var userId: String
    var deviceId: String
    var lat: Double
    var lng: Double
    var altitude: Double?
    var speed: Double?
    var heading: Double?
    var accuracy: Double?
    var timestamp: String
    var lastSyncedAt: Date

    init(from entry: LocationHistoryEntry) {
        // Composite key: userId + timestamp for uniqueness
        self.entryId = "\(entry.userId)_\(entry.timestamp)"
        self.userId = entry.userId
        self.deviceId = entry.deviceId
        self.lat = entry.lat
        self.lng = entry.lng
        self.altitude = entry.altitude
        self.speed = entry.speed
        self.heading = entry.heading
        self.accuracy = entry.accuracy
        self.timestamp = entry.timestamp
        self.lastSyncedAt = Date()
    }
}

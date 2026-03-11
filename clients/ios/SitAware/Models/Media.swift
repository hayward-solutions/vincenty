import Foundation

// MARK: - Media Rooms / Calls

struct MediaRoom: Codable, Sendable, Identifiable, Equatable {
    let id: String
    let name: String
    let roomType: String
    let groupId: String?
    let createdBy: String
    let livekitRoom: String
    let isActive: Bool
    let maxParticipants: Int
    let createdAt: String
    let endedAt: String?
}

struct JoinRoomResponse: Codable, Sendable {
    let room: MediaRoom
    let token: String
    let url: String
}

struct CreateCallRequest: Codable, Sendable {
    var groupId: String?
    var recipientId: String?
    var name: String?
    var videoEnabled: Bool?
}

// MARK: - Video Feeds

struct VideoFeed: Codable, Sendable, Identifiable, Equatable {
    let id: String
    let name: String
    let feedType: String
    let sourceUrl: String?
    let groupId: String
    let createdBy: String
    let streamKey: String?
    let isActive: Bool
    let createdAt: String
    let updatedAt: String
}

struct CreateVideoFeedRequest: Codable, Sendable {
    var name: String
    var feedType: String
    var sourceUrl: String?
    var groupId: String
}

struct VideoFeedStartResponse: Codable, Sendable {
    let feed: VideoFeed
    let ingestUrl: String?
    let streamKey: String?
    let token: String?
    let url: String?
}

// MARK: - Recordings

struct Recording: Codable, Sendable, Identifiable, Equatable {
    let id: String
    let roomId: String?
    let feedId: String?
    let fileType: String
    let durationSecs: Int?
    let fileSizeBytes: Int?
    let status: String
    let playbackUrl: String?
    let startedAt: String
    let endedAt: String?
}

// MARK: - PTT Channels

struct PTTChannel: Codable, Sendable, Identifiable, Equatable {
    let id: String
    let groupId: String
    let roomId: String
    let name: String
    let isDefault: Bool
    let createdAt: String
}

struct CreatePTTChannelRequest: Codable, Sendable {
    var name: String
    var isDefault: Bool?
}

struct JoinPTTChannelResponse: Codable, Sendable {
    let channel: PTTChannel
    let token: String
    let url: String
}

// MARK: - WebSocket Events

struct WSCallEvent: Codable, Sendable {
    let roomId: String
    let roomName: String
    let roomType: String
    let groupId: String?
    let callerId: String
    let eventType: String
}

struct WSFeedEvent: Codable, Sendable {
    let feedId: String
    let feedName: String
    let groupId: String
    let eventType: String
}

struct WSPTTFloorEvent: Codable, Sendable {
    let channelId: String
    let eventType: String
    let holderId: String?
    let holderName: String?
}

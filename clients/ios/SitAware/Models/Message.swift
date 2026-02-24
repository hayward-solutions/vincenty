import Foundation

// MARK: - Attachment

struct Attachment: Codable, Sendable, Identifiable {
    let id: String
    let filename: String
    let contentType: String
    let sizeBytes: Int
    let createdAt: String

    /// Whether this attachment is an image type.
    var isImage: Bool {
        contentType.hasPrefix("image/")
    }

    /// Human-readable file size.
    var formattedSize: String {
        let formatter = ByteCountFormatter()
        formatter.countStyle = .file
        return formatter.string(fromByteCount: Int64(sizeBytes))
    }
}

// MARK: - EXIF Location

/// GPS location extracted from an image attachment's EXIF data.
struct ExifLocation: Codable, Sendable {
    let attachmentId: String
    let lat: Double
    let lng: Double
    var altitude: Double?
    var takenAt: String?
}

// MARK: - Message Metadata

/// Typed message metadata — shape depends on message_type.
struct MessageMetadata: Codable, Sendable {
    var exifLocations: [ExifLocation]?
    var drawingId: String?

    // Catch-all for unknown keys (GPX data stored as GeoJSON, etc.)
    private var additionalProperties: [String: AnyCodable]?

    enum CodingKeys: String, CodingKey {
        case exifLocations
        case drawingId
    }
}

// MARK: - Message

struct MessageResponse: Codable, Sendable, Identifiable {
    let id: String
    let senderId: String
    let username: String
    let displayName: String
    var groupId: String?
    var recipientId: String?
    let content: String
    let messageType: String
    var lat: Double?
    var lng: Double?
    var metadata: MessageMetadata?
    let attachments: [Attachment]
    let createdAt: String
}

// MARK: - WebSocket Message

/// Server -> Client: a new message via WebSocket (same shape as MessageResponse).
typealias WSMessageNew = MessageResponse

// MARK: - Conversations

/// A user the caller has DM history with.
struct DMConversationPartner: Codable, Sendable {
    let userId: String
    let username: String
    let displayName: String
}

/// Conversation list item — either a group or a DM partner.
struct Conversation: Sendable, Identifiable, Equatable {
    let id: String
    let type: ConversationType
    let name: String
    var lastMessage: MessageResponse?

    static func == (lhs: Conversation, rhs: Conversation) -> Bool {
        lhs.id == rhs.id && lhs.type == rhs.type
    }
}

enum ConversationType: String, Sendable {
    case group
    case direct
}

// MARK: - AnyCodable Helper

/// Minimal type-erased Codable wrapper for arbitrary JSON values.
struct AnyCodable: Codable, Sendable {
    let value: Any

    init(_ value: Any) {
        self.value = value
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let string = try? container.decode(String.self) {
            value = string
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let bool = try? container.decode(Bool.self) {
            value = bool
        } else if let array = try? container.decode([AnyCodable].self) {
            value = array.map(\.value)
        } else if let dict = try? container.decode([String: AnyCodable].self) {
            value = dict.mapValues(\.value)
        } else {
            value = NSNull()
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch value {
        case let string as String: try container.encode(string)
        case let int as Int: try container.encode(int)
        case let double as Double: try container.encode(double)
        case let bool as Bool: try container.encode(bool)
        default: try container.encodeNil()
        }
    }
}

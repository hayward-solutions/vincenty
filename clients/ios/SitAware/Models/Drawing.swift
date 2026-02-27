import Foundation

// MARK: - GeoJSON Types

/// Minimal GeoJSON FeatureCollection — stores drawing geometry and overlays.
struct GeoJSONFeatureCollection: Codable, Sendable {
    let type: String // "FeatureCollection"
    var features: [GeoJSONFeature]

    init(features: [GeoJSONFeature] = []) {
        self.type = "FeatureCollection"
        self.features = features
    }

    static var empty: GeoJSONFeatureCollection {
        GeoJSONFeatureCollection(features: [])
    }
}

struct GeoJSONFeature: Codable, Sendable {
    let type: String // "Feature"
    let geometry: GeoJSONGeometry
    var properties: [String: AnyCodable]?

    init(geometry: GeoJSONGeometry, properties: [String: AnyCodable]? = nil) {
        self.type = "Feature"
        self.geometry = geometry
        self.properties = properties
    }
}

/// GeoJSON geometry types used by SitAware (Point, LineString, Polygon).
enum GeoJSONGeometry: Codable, Sendable {
    case point([Double])                // [lng, lat]
    case lineString([[Double]])         // [[lng, lat], ...]
    case polygon([[[Double]]])          // [[[lng, lat], ...], ...]

    enum CodingKeys: String, CodingKey {
        case type
        case coordinates
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        let type = try container.decode(String.self, forKey: .type)

        switch type {
        case "Point":
            let coords = try container.decode([Double].self, forKey: .coordinates)
            self = .point(coords)
        case "LineString":
            let coords = try container.decode([[Double]].self, forKey: .coordinates)
            self = .lineString(coords)
        case "Polygon":
            let coords = try container.decode([[[Double]]].self, forKey: .coordinates)
            self = .polygon(coords)
        default:
            throw DecodingError.dataCorruptedError(
                forKey: .type, in: container,
                debugDescription: "Unknown GeoJSON geometry type: \(type)")
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        switch self {
        case .point(let coords):
            try container.encode("Point", forKey: .type)
            try container.encode(coords, forKey: .coordinates)
        case .lineString(let coords):
            try container.encode("LineString", forKey: .type)
            try container.encode(coords, forKey: .coordinates)
        case .polygon(let coords):
            try container.encode("Polygon", forKey: .type)
            try container.encode(coords, forKey: .coordinates)
        }
    }
}

// MARK: - Drawing

struct DrawingResponse: Codable, Sendable, Identifiable {
    let id: String
    let ownerId: String
    let username: String
    let displayName: String
    let name: String
    let geojson: GeoJSONFeatureCollection
    let createdAt: String
    let updatedAt: String
}

// MARK: - Requests

struct CreateDrawingRequest: Codable, Sendable {
    let name: String
    let geojson: GeoJSONFeatureCollection
}

struct UpdateDrawingRequest: Codable, Sendable {
    var name: String?
    var geojson: GeoJSONFeatureCollection?
}

struct ShareDrawingRequest: Codable, Sendable {
    var groupId: String?
    var recipientId: String?
}

// MARK: - Share Info

struct DrawingShareInfo: Codable, Sendable, Identifiable {
    var id: String { messageId }
    let type: String // "group" or "user"
    let name: String
    let sharedAt: String
    let messageId: String

    enum CodingKeys: String, CodingKey {
        case type, name, sharedAt, messageId
        // `id` in the JSON maps to the share's own identifier (group_id or user_id),
        // but we use messageId as our Identifiable id to avoid conflict.
        case shareTargetId = "id"
    }

    let shareTargetId: String
}

// MARK: - WebSocket Drawing Update

/// Server -> Client: a drawing was updated by its owner.
typealias WSDrawingUpdated = DrawingResponse

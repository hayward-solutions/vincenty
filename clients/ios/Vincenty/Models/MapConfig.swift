import Foundation

// MARK: - Map Settings

/// Combined map configuration returned by `GET /api/v1/map/settings`.
struct MapSettings: Codable, Sendable {
    let tileUrl: String
    var styleJson: [String: AnyCodable]?
    let centerLat: Double
    let centerLng: Double
    let zoom: Double
    let minZoom: Int
    let maxZoom: Int
    let terrainUrl: String
    let terrainEncoding: String
    var mapboxAccessToken: String?
    var googleMapsApiKey: String?
    let configs: [MapConfigResponse]
}

// MARK: - Map Tile Config

struct MapConfigResponse: Codable, Sendable, Identifiable {
    let id: String
    let name: String
    let sourceType: String
    let tileUrl: String
    var styleJson: [String: AnyCodable]?
    let minZoom: Int
    let maxZoom: Int
    let isDefault: Bool
    let isBuiltin: Bool
    let isEnabled: Bool
    var createdBy: String?
    let createdAt: String
    let updatedAt: String
}

struct CreateMapConfigRequest: Codable, Sendable {
    let name: String
    var sourceType: String?
    var tileUrl: String?
    var styleJson: [String: AnyCodable]?
    var minZoom: Int?
    var maxZoom: Int?
    var isDefault: Bool?
}

struct UpdateMapConfigRequest: Codable, Sendable {
    var name: String?
    var sourceType: String?
    var tileUrl: String?
    var styleJson: [String: AnyCodable]?
    var minZoom: Int?
    var maxZoom: Int?
    var isDefault: Bool?
    var isEnabled: Bool?
}

// MARK: - Terrain Config

struct TerrainConfigResponse: Codable, Sendable, Identifiable {
    let id: String
    let name: String
    let sourceType: String
    let terrainUrl: String
    let terrainEncoding: String
    let isDefault: Bool
    let isBuiltin: Bool
    let isEnabled: Bool
    var createdBy: String?
    let createdAt: String
    let updatedAt: String
}

struct CreateTerrainConfigRequest: Codable, Sendable {
    let name: String
    var sourceType: String?
    let terrainUrl: String
    var terrainEncoding: String?
    var isDefault: Bool?
}

struct UpdateTerrainConfigRequest: Codable, Sendable {
    var name: String?
    var sourceType: String?
    var terrainUrl: String?
    var terrainEncoding: String?
    var isDefault: Bool?
    var isEnabled: Bool?
}

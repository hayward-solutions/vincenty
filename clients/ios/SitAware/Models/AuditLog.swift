import Foundation

// MARK: - Audit Log

struct AuditLogResponse: Codable, Sendable, Identifiable {
    let id: String
    let userId: String
    let username: String
    let displayName: String
    var deviceId: String?
    let action: String
    let resourceType: String
    var resourceId: String?
    var groupId: String?
    var metadata: [String: AnyCodable]?
    var lat: Double?
    var lng: Double?
    let ipAddress: String
    let createdAt: String
}

// MARK: - Filters

struct AuditFilters: Sendable, Equatable {
    var from: String?
    var to: String?
    var action: String?
    var resourceType: String?
    var page: Int?
    var pageSize: Int?

    /// Convert to query parameters dictionary for the API.
    var queryParams: [String: String] {
        var params: [String: String] = [:]
        if let from { params["from"] = from }
        if let to { params["to"] = to }
        if let action { params["action"] = action }
        if let resourceType { params["resource_type"] = resourceType }
        if let page { params["page"] = String(page) }
        if let pageSize { params["page_size"] = String(pageSize) }
        return params
    }

    /// Known audit log actions (matching web's 16 options).
    static let knownActions = [
        "auth.login", "auth.logout",
        "user.create", "user.update", "user.delete",
        "group.create", "group.update", "group.delete",
        "group.member.add", "group.member.remove",
        "message.send", "message.delete",
        "map_config.create", "map_config.update", "map_config.delete",
    ]

    /// Known resource types.
    static let knownResourceTypes = [
        "session", "user", "device", "group", "message", "map_config",
    ]
}

import Foundation

// MARK: - Group

struct Group: Codable, Sendable, Identifiable {
    let id: String
    let name: String
    let description: String
    let markerIcon: String
    let markerColor: String
    var createdBy: String?
    let memberCount: Int
    let createdAt: String
    let updatedAt: String
}

// MARK: - Group Member

struct GroupMember: Codable, Sendable, Identifiable {
    let id: String
    let groupId: String
    let userId: String
    let username: String
    let displayName: String
    let canRead: Bool
    let canWrite: Bool
    let isGroupAdmin: Bool
    let createdAt: String
    let updatedAt: String
}

// MARK: - Requests

struct CreateGroupRequest: Codable, Sendable {
    let name: String
    var description: String?
    var markerIcon: String?
    var markerColor: String?
}

struct UpdateGroupRequest: Codable, Sendable {
    var name: String?
    var description: String?
    var markerIcon: String?
    var markerColor: String?
}

struct UpdateGroupMarkerRequest: Codable, Sendable {
    var markerIcon: String?
    var markerColor: String?
}

struct AddGroupMemberRequest: Codable, Sendable {
    let userId: String
    var canRead: Bool?
    var canWrite: Bool?
    var isGroupAdmin: Bool?
}

struct UpdateGroupMemberRequest: Codable, Sendable {
    var canRead: Bool?
    var canWrite: Bool?
    var isGroupAdmin: Bool?
}

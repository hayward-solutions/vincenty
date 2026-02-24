import Foundation

// MARK: - User

struct User: Codable, Sendable, Identifiable, Equatable {
    let id: String
    let username: String
    let email: String
    let displayName: String
    let avatarUrl: String
    let markerIcon: String
    let markerColor: String
    let isAdmin: Bool
    let isActive: Bool
    let mfaEnabled: Bool
    let createdAt: String
    let updatedAt: String
}

// MARK: - Requests

struct CreateUserRequest: Codable, Sendable {
    let username: String
    let email: String
    let password: String
    var displayName: String?
    var isAdmin: Bool?
}

struct UpdateUserRequest: Codable, Sendable {
    var email: String?
    var displayName: String?
    var password: String?
    var isAdmin: Bool?
    var isActive: Bool?
}

struct UpdateMeRequest: Codable, Sendable {
    var email: String?
    var displayName: String?
    var markerIcon: String?
    var markerColor: String?
}

struct ChangePasswordRequest: Codable, Sendable {
    let currentPassword: String
    let newPassword: String
}

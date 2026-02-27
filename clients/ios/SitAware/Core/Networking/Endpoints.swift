import Foundation

/// All API v1 endpoint path constants.
enum Endpoints {
    // MARK: - Auth
    static let login = "/api/v1/auth/login"
    static let refresh = "/api/v1/auth/refresh"
    static let logout = "/api/v1/auth/logout"

    // MARK: - MFA Challenge (login-time)
    static let mfaTOTP = "/api/v1/auth/mfa/totp"
    static let mfaRecovery = "/api/v1/auth/mfa/recovery"
    static let mfaWebAuthnBegin = "/api/v1/auth/mfa/webauthn/begin"
    static let mfaWebAuthnFinish = "/api/v1/auth/mfa/webauthn/finish"

    // MARK: - Passkey (passwordless)
    static let passkeyBegin = "/api/v1/auth/passkey/begin"
    static let passkeyFinish = "/api/v1/auth/passkey/finish"

    // MARK: - Users (current user)
    static let usersMe = "/api/v1/users/me"
    static let usersMePassword = "/api/v1/users/me/password"
    static let usersMeAvatar = "/api/v1/users/me/avatar"
    static let usersMeDevices = "/api/v1/users/me/devices"
    static let usersMeDevicesResolve = "/api/v1/users/me/devices/resolve"
    static let usersMeGroups = "/api/v1/users/me/groups"
    static let usersMeLocationsHistory = "/api/v1/users/me/locations/history"
    static let usersMeLocationsExport = "/api/v1/users/me/locations/export"

    static func usersMeDeviceClaim(_ deviceId: String) -> String {
        "/api/v1/users/me/devices/\(deviceId)/claim"
    }
    static func usersMeDevicePrimary(_ deviceId: String) -> String {
        "/api/v1/users/me/devices/\(deviceId)/primary"
    }

    // MARK: - MFA Management (current user)
    static let usersMeMFAMethods = "/api/v1/users/me/mfa/methods"
    static let usersMeMFATOTPSetup = "/api/v1/users/me/mfa/totp/setup"
    static let usersMeMFATOTPVerify = "/api/v1/users/me/mfa/totp/verify"
    static let usersMeMFAWebAuthnRegisterBegin = "/api/v1/users/me/mfa/webauthn/register/begin"
    static let usersMeMFAWebAuthnRegisterFinish = "/api/v1/users/me/mfa/webauthn/register/finish"
    static let usersMeMFARecoveryCodes = "/api/v1/users/me/mfa/recovery-codes"

    static func usersMeMFAMethod(_ methodId: String) -> String {
        "/api/v1/users/me/mfa/methods/\(methodId)"
    }
    static func usersMeMFAWebAuthnPasswordless(_ credentialId: String) -> String {
        "/api/v1/users/me/mfa/webauthn/\(credentialId)/passwordless"
    }

    // MARK: - Users (admin)
    static let users = "/api/v1/users"

    static func user(_ id: String) -> String { "/api/v1/users/\(id)" }
    static func userAvatar(_ id: String) -> String { "/api/v1/users/\(id)/avatar" }
    static func userMFA(_ id: String) -> String { "/api/v1/users/\(id)/mfa" }
    static func userLocationsHistory(_ id: String) -> String {
        "/api/v1/users/\(id)/locations/history"
    }

    // MARK: - Devices
    static func device(_ id: String) -> String { "/api/v1/devices/\(id)" }

    // MARK: - Groups
    static let groups = "/api/v1/groups"

    static func group(_ id: String) -> String { "/api/v1/groups/\(id)" }
    static func groupMembers(_ id: String) -> String { "/api/v1/groups/\(id)/members" }
    static func groupMember(_ groupId: String, _ userId: String) -> String {
        "/api/v1/groups/\(groupId)/members/\(userId)"
    }
    static func groupMarker(_ id: String) -> String { "/api/v1/groups/\(id)/marker" }
    static func groupMessages(_ id: String) -> String { "/api/v1/groups/\(id)/messages" }
    static func groupLocationsHistory(_ id: String) -> String {
        "/api/v1/groups/\(id)/locations/history"
    }
    static func groupAuditLogs(_ id: String) -> String { "/api/v1/groups/\(id)/audit-logs" }

    // MARK: - Locations
    static let locations = "/api/v1/locations"
    static let locationsHistory = "/api/v1/locations/history"

    // MARK: - Messages
    static let messages = "/api/v1/messages"
    static let messagesConversations = "/api/v1/messages/conversations"

    static func message(_ id: String) -> String { "/api/v1/messages/\(id)" }
    static func directMessages(_ userId: String) -> String {
        "/api/v1/messages/direct/\(userId)"
    }

    // MARK: - Attachments
    static func attachmentDownload(_ id: String) -> String {
        "/api/v1/attachments/\(id)/download"
    }

    // MARK: - Drawings
    static let drawings = "/api/v1/drawings"
    static let drawingsShared = "/api/v1/drawings/shared"

    static func drawing(_ id: String) -> String { "/api/v1/drawings/\(id)" }
    static func drawingShares(_ id: String) -> String { "/api/v1/drawings/\(id)/shares" }
    static func drawingShare(_ id: String) -> String { "/api/v1/drawings/\(id)/share" }
    static func drawingUnshare(_ drawingId: String, _ messageId: String) -> String {
        "/api/v1/drawings/\(drawingId)/shares/\(messageId)"
    }

    // MARK: - Map Config
    static let mapSettings = "/api/v1/map/settings"
    static let mapConfigs = "/api/v1/map-configs"
    static func mapConfig(_ id: String) -> String { "/api/v1/map-configs/\(id)" }

    // MARK: - Terrain Config
    static let terrainConfigs = "/api/v1/terrain-configs"
    static func terrainConfig(_ id: String) -> String { "/api/v1/terrain-configs/\(id)" }

    // MARK: - Audit Logs
    static let auditLogs = "/api/v1/audit-logs"
    static let auditLogsMe = "/api/v1/audit-logs/me"
    static let auditLogsMeExport = "/api/v1/audit-logs/me/export"
    static let auditLogsExport = "/api/v1/audit-logs/export"

    // MARK: - Server Settings
    static let serverSettings = "/api/v1/server/settings"

    // MARK: - WebSocket
    static let ws = "/api/v1/ws"

    // MARK: - Health
    static let healthz = "/healthz"
    static let readyz = "/readyz"
}

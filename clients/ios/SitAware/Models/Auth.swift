import Foundation

// MARK: - Auth Response

/// Returned on successful authentication (login, MFA verify, passkey, token refresh).
struct AuthResponse: Codable, Sendable {
    let accessToken: String
    let refreshToken: String
    let user: User
}

// MARK: - MFA Challenge

/// Returned by login when MFA is required — client must complete an MFA challenge.
struct MFAChallengeResponse: Codable, Sendable {
    let mfaRequired: Bool
    let mfaToken: String
    let methods: [String] // e.g. ["totp", "webauthn", "recovery"]
}

// MARK: - Login Result

/// Discriminated result from the login endpoint — either full auth or MFA challenge.
enum LoginResult: Sendable {
    case authenticated(AuthResponse)
    case mfaRequired(MFAChallengeResponse)
}

extension LoginResult: Decodable {
    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        // Try MFA challenge first (has the `mfa_required` key)
        if let challenge = try? container.decode(MFAChallengeResponse.self),
           challenge.mfaRequired {
            self = .mfaRequired(challenge)
            return
        }

        // Otherwise decode as full auth response
        let auth = try container.decode(AuthResponse.self)
        self = .authenticated(auth)
    }
}

// MARK: - Token Refresh

struct RefreshTokenRequest: Codable, Sendable {
    let refreshToken: String
}

struct LogoutRequest: Codable, Sendable {
    let refreshToken: String
}

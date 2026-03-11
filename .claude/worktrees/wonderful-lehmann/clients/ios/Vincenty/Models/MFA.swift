import Foundation

// MARK: - MFA Method

/// A single enrolled MFA method (TOTP or WebAuthn).
struct MFAMethod: Codable, Sendable, Identifiable {
    let id: String
    let type: MFAMethodType
    let name: String
    let verified: Bool
    var passwordlessEnabled: Bool?
    var lastUsedAt: String?
    let createdAt: String
}

enum MFAMethodType: String, Codable, Sendable {
    case totp
    case webauthn
}

// MARK: - TOTP Setup

/// Returned when beginning TOTP setup — contains the QR code URI and secret.
struct TOTPSetupResponse: Codable, Sendable {
    let methodId: String
    let secret: String
    let uri: String
    let issuer: String
    let account: String
}

/// Returned when TOTP is verified (first method triggers recovery codes).
struct TOTPVerifyResponse: Codable, Sendable {
    let verified: Bool
    var recoveryCodes: [String]?
}

// MARK: - WebAuthn Registration

/// Returned when WebAuthn registration finishes.
struct WebAuthnRegisterResponse: Codable, Sendable {
    let registered: Bool
    var recoveryCodes: [String]?
}

// MARK: - Recovery Codes

struct RecoveryCodesResponse: Codable, Sendable {
    let codes: [String]
}

// MARK: - Server Settings

struct ServerSettings: Codable, Sendable {
    var mfaRequired: Bool
    var mapboxAccessToken: String
    var googleMapsApiKey: String
}

// MARK: - WebAuthn / Passkey Options (generic JSON containers)

/// Options returned by the server for WebAuthn assertion/registration.
/// We pass these through to ASAuthorization — the exact shape varies, so we
/// decode only the fields we need and keep the rest as raw data.
struct WebAuthnOptions: Codable, Sendable {
    let challenge: String
    var timeout: Int?
    var rpId: String?
    var allowCredentials: [WebAuthnCredentialDescriptor]?
    var userVerification: String?
}

struct WebAuthnCredentialDescriptor: Codable, Sendable {
    let type: String
    let id: String
    var transports: [String]?
}

struct WebAuthnCreationOptions: Codable, Sendable {
    let challenge: String
    let rp: WebAuthnRP
    let user: WebAuthnUser
    var pubKeyCredParams: [WebAuthnPubKeyParam]?
    var timeout: Int?
    var excludeCredentials: [WebAuthnCredentialDescriptor]?
    var authenticatorSelection: WebAuthnAuthenticatorSelection?
    var attestation: String?
}

struct WebAuthnRP: Codable, Sendable {
    let name: String
    let id: String
}

struct WebAuthnUser: Codable, Sendable {
    let name: String
    let displayName: String
    let id: String
}

struct WebAuthnPubKeyParam: Codable, Sendable {
    let type: String
    let alg: Int
}

struct WebAuthnAuthenticatorSelection: Codable, Sendable {
    var authenticatorAttachment: String?
    var residentKey: String?
    var requireResidentKey: Bool?
    var userVerification: String?
}

/// Passkey begin response from the server.
struct PasskeyBeginResponse: Codable, Sendable {
    let options: PasskeyOptionsWrapper
    let sessionId: String
}

/// The server wraps options inside a `publicKey` key.
struct PasskeyOptionsWrapper: Codable, Sendable {
    let publicKey: WebAuthnOptions
}

/// WebAuthn registration begin response.
struct WebAuthnRegisterBeginResponse: Codable, Sendable {
    let publicKey: WebAuthnCreationOptions
}

/// MFA WebAuthn challenge begin response.
struct MFAWebAuthnBeginResponse: Codable, Sendable {
    let options: PasskeyOptionsWrapper
}

// MARK: - WebAuthn Outbound Models (finish requests)

/// Encoder for WebAuthn request bodies.
///
/// The W3C WebAuthn spec requires exact camelCase field names (`clientDataJSON`,
/// `authenticatorData`, `rawId`, etc.), so we cannot use APIClient's default
/// `convertToSnakeCase` encoder. Encode finish request bodies with this encoder
/// and send via `APIClient.postRawJSON`.
enum WebAuthnJSON {
    static let encoder: JSONEncoder = {
        let encoder = JSONEncoder()
        return encoder
    }()
}

/// Assertion response fields (used by passkey login and WebAuthn MFA finish).
struct WebAuthnAssertionResponseData: Encodable, Sendable {
    let authenticatorData: String
    let clientDataJSON: String
    let signature: String
    let userHandle: String?
}

/// Passkey (passwordless) finish request body.
/// Merges `session_id` with the W3C credential assertion fields.
struct PasskeyFinishRequest: Encodable, Sendable {
    let sessionId: String
    let id: String
    let rawId: String
    let type: String
    let response: WebAuthnAssertionResponseData

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case id, rawId, type, response
    }
}

/// MFA WebAuthn finish request body.
/// Merges `mfa_token` with the W3C credential assertion fields.
struct MFAWebAuthnFinishRequest: Encodable, Sendable {
    let mfaToken: String
    let id: String
    let rawId: String
    let type: String
    let response: WebAuthnAssertionResponseData

    enum CodingKeys: String, CodingKey {
        case mfaToken = "mfa_token"
        case id, rawId, type, response
    }
}

/// Attestation response fields (used by WebAuthn registration finish).
struct WebAuthnAttestationResponseData: Encodable, Sendable {
    let attestationObject: String
    let clientDataJSON: String
}

/// WebAuthn registration finish request body.
struct WebAuthnRegistrationFinishRequest: Encodable, Sendable {
    let id: String
    let rawId: String
    let type: String
    let response: WebAuthnAttestationResponseData
}

/// Response from passwordless toggle endpoint.
struct PasswordlessToggleResponse: Decodable, Sendable {
    let passwordlessEnabled: Bool
}

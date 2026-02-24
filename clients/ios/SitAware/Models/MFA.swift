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

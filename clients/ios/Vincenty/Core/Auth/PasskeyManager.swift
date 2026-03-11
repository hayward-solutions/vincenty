import AuthenticationServices
import Foundation
import UIKit

/// Wraps ASAuthorizationController for WebAuthn/passkey ceremonies.
///
/// Provides async/await methods for:
/// - Discoverable assertion (passkey login — passwordless, uses Face ID)
/// - Standard assertion (WebAuthn MFA challenge)
/// - Registration (enroll new security key/passkey)
///
/// All ASAuthorization UI is presented as a system sheet over the app's key window.
@MainActor
final class PasskeyManager: NSObject {
    static let shared = PasskeyManager()

    private var continuation: CheckedContinuation<ASAuthorizationCredential, any Error>?

    private override init() {
        super.init()
    }

    // MARK: - Result Types

    /// Result of a WebAuthn assertion ceremony (login or MFA).
    struct AssertionResult: Sendable {
        let credentialID: Data
        let rawClientDataJSON: Data
        let rawAuthenticatorData: Data
        let signature: Data
        let userID: Data
    }

    /// Result of a WebAuthn registration ceremony (enroll new credential).
    struct RegistrationResult: Sendable {
        let credentialID: Data
        let rawClientDataJSON: Data
        let rawAttestationObject: Data
    }

    // MARK: - Discoverable Assertion (Passkey Login)

    /// Perform a discoverable credential assertion (passkey login — no username required).
    ///
    /// The system presents Face ID / Touch ID and retrieves matching passkeys
    /// for the given relying party.
    func assertDiscoverable(
        challenge: Data,
        relyingPartyID: String
    ) async throws -> AssertionResult {
        let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(
            relyingPartyIdentifier: relyingPartyID)
        let request = provider.createCredentialAssertionRequest(
            challenge: challenge)

        let credential = try await performRequest(request)
        return try extractAssertionResult(from: credential)
    }

    // MARK: - Standard Assertion (WebAuthn MFA)

    /// Perform a credential assertion with allowed credentials (WebAuthn MFA challenge).
    func assert(
        challenge: Data,
        relyingPartyID: String,
        allowCredentials: [Data]
    ) async throws -> AssertionResult {
        let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(
            relyingPartyIdentifier: relyingPartyID)
        let request = provider.createCredentialAssertionRequest(
            challenge: challenge)

        if !allowCredentials.isEmpty {
            request.allowedCredentials = allowCredentials.map {
                ASAuthorizationPlatformPublicKeyCredentialDescriptor(credentialID: $0)
            }
        }

        let credential = try await performRequest(request)
        return try extractAssertionResult(from: credential)
    }

    // MARK: - Registration (Enroll New Credential)

    /// Perform a credential registration (enroll new security key or passkey).
    func register(
        challenge: Data,
        relyingPartyID: String,
        userName: String,
        userID: Data
    ) async throws -> RegistrationResult {
        let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(
            relyingPartyIdentifier: relyingPartyID)
        let request = provider.createCredentialRegistrationRequest(
            challenge: challenge,
            name: userName,
            userID: userID)

        let credential = try await performRequest(request)
        return try extractRegistrationResult(from: credential)
    }

    // MARK: - Private

    private func performRequest(
        _ request: ASAuthorizationRequest
    ) async throws -> ASAuthorizationCredential {
        try await withCheckedThrowingContinuation { continuation in
            self.continuation = continuation
            let controller = ASAuthorizationController(authorizationRequests: [request])
            controller.delegate = self
            controller.presentationContextProvider = self
            controller.performRequests()
        }
    }

    private func extractAssertionResult(
        from credential: ASAuthorizationCredential
    ) throws -> AssertionResult {
        guard let assertion = credential
            as? ASAuthorizationPlatformPublicKeyCredentialAssertion
        else {
            throw PasskeyError.unexpectedCredentialType
        }
        return AssertionResult(
            credentialID: assertion.credentialID,
            rawClientDataJSON: assertion.rawClientDataJSON,
            rawAuthenticatorData: assertion.rawAuthenticatorData,
            signature: assertion.signature,
            userID: assertion.userID
        )
    }

    private func extractRegistrationResult(
        from credential: ASAuthorizationCredential
    ) throws -> RegistrationResult {
        guard let registration = credential
            as? ASAuthorizationPlatformPublicKeyCredentialRegistration
        else {
            throw PasskeyError.unexpectedCredentialType
        }
        guard let attestationObject = registration.rawAttestationObject else {
            throw PasskeyError.missingAttestationObject
        }
        return RegistrationResult(
            credentialID: registration.credentialID,
            rawClientDataJSON: registration.rawClientDataJSON,
            rawAttestationObject: attestationObject
        )
    }
}

// MARK: - ASAuthorizationControllerDelegate

extension PasskeyManager: ASAuthorizationControllerDelegate {
    func authorizationController(
        controller: ASAuthorizationController,
        didCompleteWithAuthorization authorization: ASAuthorization
    ) {
        continuation?.resume(returning: authorization.credential)
        continuation = nil
    }

    func authorizationController(
        controller: ASAuthorizationController,
        didCompleteWithError error: any Error
    ) {
        let mapped: any Error
        if let authError = error as? ASAuthorizationError {
            switch authError.code {
            case .canceled:
                mapped = PasskeyError.canceled
            case .invalidResponse:
                mapped = PasskeyError.invalidResponse
            case .notHandled:
                mapped = PasskeyError.notHandled
            case .notInteractive:
                mapped = PasskeyError.notInteractive
            default:
                mapped = PasskeyError.failed(authError.localizedDescription)
            }
        } else {
            mapped = error
        }

        continuation?.resume(throwing: mapped)
        continuation = nil
    }
}

// MARK: - ASAuthorizationControllerPresentationContextProviding

extension PasskeyManager: ASAuthorizationControllerPresentationContextProviding {
    func presentationAnchor(
        for controller: ASAuthorizationController
    ) -> ASPresentationAnchor {
        guard let scene = UIApplication.shared.connectedScenes
            .compactMap({ $0 as? UIWindowScene })
            .first,
            let window = scene.windows.first(where: { $0.isKeyWindow })
        else {
            return UIWindow()
        }
        return window
    }
}

// MARK: - PasskeyError

/// Errors from WebAuthn/passkey operations.
enum PasskeyError: LocalizedError, Sendable {
    case canceled
    case failed(String)
    case invalidResponse
    case notHandled
    case notInteractive
    case unexpectedCredentialType
    case missingAttestationObject

    var errorDescription: String? {
        switch self {
        case .canceled:
            return nil // User cancellation — caller should not show an error
        case .failed(let detail):
            return "Passkey operation failed: \(detail)"
        case .invalidResponse:
            return "Invalid response from authenticator."
        case .notHandled:
            return "The request was not handled."
        case .notInteractive:
            return "The operation requires user interaction."
        case .unexpectedCredentialType:
            return "Received unexpected credential type."
        case .missingAttestationObject:
            return "Registration response missing attestation data."
        }
    }
}

import Foundation
import AuthenticationServices

/// Central authentication state manager.
/// Mirrors the web client's `AuthContext` — manages login, MFA, passkey, logout, and session state.
@Observable @MainActor
final class AuthManager {
    private(set) var user: User?
    private(set) var isLoading = true
    private(set) var error: String?
    var hasServerURL = false

    var isAuthenticated: Bool { user != nil }
    var isAdmin: Bool { user?.isAdmin ?? false }

    private let api = APIClient.shared

    // MARK: - Bootstrap

    /// Check for existing session on app launch.
    /// Mirrors the web client's `useEffect` in `AuthProvider` that checks `localStorage`.
    func bootstrap() async {
        hasServerURL = KeychainStore.shared.serverURL != nil

        guard KeychainStore.shared.accessToken != nil else {
            AppLogger.shared.log(.info, .auth, "Bootstrap: no stored token")
            isLoading = false
            return
        }

        AppLogger.shared.log(.info, .auth, "Bootstrap: validating stored token")
        do {
            let fetchedUser: User = try await api.get(Endpoints.usersMe)
            self.user = fetchedUser
            AppLogger.shared.log(.info, .auth, "Bootstrap: authenticated as \(fetchedUser.username)")
        } catch {
            // Token invalid — clear and require re-login
            AppLogger.shared.log(.warning, .auth, "Bootstrap: token invalid, clearing session",
                                 detail: error.localizedDescription)
            await api.tokenManager.clearTokens()
        }

        isLoading = false
    }

    // MARK: - Login

    /// Login with username and password. Returns the result which may require MFA.
    func login(username: String, password: String) async throws -> LoginResult {
        self.error = nil

        struct LoginBody: Encodable {
            let username: String
            let password: String
        }

        let result: LoginResult = try await api.post(
            Endpoints.login,
            body: LoginBody(username: username, password: password))

        switch result {
        case .authenticated(let auth):
            await api.tokenManager.setTokens(access: auth.accessToken, refresh: auth.refreshToken)
            self.user = auth.user
            AppLogger.shared.log(.info, .auth, "Login successful: \(auth.user.username)")

        case .mfaRequired:
            AppLogger.shared.log(.info, .auth, "Login: MFA challenge required")
        }

        return result
    }

    /// Complete login after successful MFA verification.
    func completeMFALogin(_ response: AuthResponse) async {
        await api.tokenManager.setTokens(access: response.accessToken, refresh: response.refreshToken)
        self.user = response.user
        AppLogger.shared.log(.info, .auth, "MFA login complete: \(response.user.username)")
    }

    // MARK: - Passkey Login

    /// Login using a passkey (passwordless WebAuthn discoverable credential).
    ///
    /// Triggers Face ID / Touch ID and retrieves a matching passkey for the
    /// server's relying party. On success, tokens are stored and `user` is set.
    func passkeyLogin() async throws {
        self.error = nil

        // 1. Begin — server returns assertion options + session ID
        let beginResponse: PasskeyBeginResponse = try await api.post(Endpoints.passkeyBegin)
        let options = beginResponse.options.publicKey

        guard let challengeData = Data(base64URLEncoded: options.challenge) else {
            throw APIError(status: 0, message: "Invalid challenge from server")
        }

        // 2. Perform discoverable assertion via ASAuthorization (Face ID / passkey)
        let result = try await PasskeyManager.shared.assertDiscoverable(
            challenge: challengeData,
            relyingPartyID: options.rpId ?? "")

        // 3. Build finish body with exact W3C field names
        let credentialID = result.credentialID.base64URLEncodedString()
        let body = PasskeyFinishRequest(
            sessionId: beginResponse.sessionId,
            id: credentialID,
            rawId: credentialID,
            type: "public-key",
            response: WebAuthnAssertionResponseData(
                authenticatorData: result.rawAuthenticatorData.base64URLEncodedString(),
                clientDataJSON: result.rawClientDataJSON.base64URLEncodedString(),
                signature: result.signature.base64URLEncodedString(),
                userHandle: result.userID.isEmpty
                    ? nil : result.userID.base64URLEncodedString()
            )
        )

        let jsonData = try WebAuthnJSON.encoder.encode(body)

        // 4. Finish — server validates assertion and returns auth tokens
        let auth: AuthResponse = try await api.postRawJSON(
            Endpoints.passkeyFinish, jsonData: jsonData)

        await api.tokenManager.setTokens(access: auth.accessToken, refresh: auth.refreshToken)
        self.user = auth.user
        AppLogger.shared.log(.info, .auth, "Passkey login successful: \(auth.user.username)")
    }

    // MARK: - MFA Challenge Verification

    /// Verify TOTP code during MFA challenge.
    func verifyTOTP(mfaToken: String, code: String) async throws -> AuthResponse {
        struct Body: Encodable {
            let mfaToken: String
            let code: String
        }
        return try await api.post(Endpoints.mfaTOTP, body: Body(mfaToken: mfaToken, code: code))
    }

    /// Verify recovery code during MFA challenge.
    func verifyRecoveryCode(mfaToken: String, code: String) async throws -> AuthResponse {
        struct Body: Encodable {
            let mfaToken: String
            let code: String
        }
        return try await api.post(Endpoints.mfaRecovery, body: Body(mfaToken: mfaToken, code: code))
    }

    /// Verify WebAuthn credential during MFA challenge.
    ///
    /// Calls the MFA WebAuthn begin endpoint to get assertion options, then
    /// presents the system authenticator UI, and submits the result to the
    /// MFA WebAuthn finish endpoint.
    func verifyWebAuthn(mfaToken: String) async throws -> AuthResponse {
        // 1. Begin — get assertion options for the user's registered credentials
        struct BeginBody: Encodable {
            let mfaToken: String
        }
        let beginResponse: MFAWebAuthnBeginResponse = try await api.post(
            Endpoints.mfaWebAuthnBegin,
            body: BeginBody(mfaToken: mfaToken))

        let options = beginResponse.options.publicKey

        guard let challengeData = Data(base64URLEncoded: options.challenge) else {
            throw APIError(status: 0, message: "Invalid challenge from server")
        }

        let allowCredentials = (options.allowCredentials ?? []).compactMap {
            Data(base64URLEncoded: $0.id)
        }

        // 2. Perform assertion via ASAuthorization
        let result = try await PasskeyManager.shared.assert(
            challenge: challengeData,
            relyingPartyID: options.rpId ?? "",
            allowCredentials: allowCredentials)

        // 3. Build finish body
        let credentialID = result.credentialID.base64URLEncodedString()
        let body = MFAWebAuthnFinishRequest(
            mfaToken: mfaToken,
            id: credentialID,
            rawId: credentialID,
            type: "public-key",
            response: WebAuthnAssertionResponseData(
                authenticatorData: result.rawAuthenticatorData.base64URLEncodedString(),
                clientDataJSON: result.rawClientDataJSON.base64URLEncodedString(),
                signature: result.signature.base64URLEncodedString(),
                userHandle: result.userID.isEmpty
                    ? nil : result.userID.base64URLEncodedString()
            )
        )

        let jsonData = try WebAuthnJSON.encoder.encode(body)
        return try await api.postRawJSON(Endpoints.mfaWebAuthnFinish, jsonData: jsonData)
    }

    // MARK: - Logout

    /// Revoke refresh token and clear local state.
    func logout() async {
        AppLogger.shared.log(.info, .auth, "Logging out")
        if let refreshToken = KeychainStore.shared.refreshToken {
            do {
                try await api.post(
                    Endpoints.logout,
                    body: LogoutRequest(refreshToken: refreshToken)) as EmptyResponse
            } catch {
                // Ignore errors during logout — clear local state regardless
                AppLogger.shared.log(.warning, .auth, "Logout: server revocation failed",
                                     detail: error.localizedDescription)
            }
        }

        await api.tokenManager.clearTokens()
        self.user = nil
        AppLogger.shared.log(.info, .auth, "Logged out")
    }

    // MARK: - Refresh User

    /// Re-fetch the current user's profile.
    func refreshUser() async {
        do {
            let updated: User = try await api.get(Endpoints.usersMe)
            self.user = updated
        } catch {
            // Silently ignore — user state remains stale
        }
    }
}

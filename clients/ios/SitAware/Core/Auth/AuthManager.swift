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

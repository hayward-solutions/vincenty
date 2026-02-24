import Foundation

/// Thread-safe token management with single-flight refresh deduplication.
/// Swift `actor` ensures only one refresh request is in-flight at a time —
/// this is the equivalent of the web client's `refreshPromise` pattern.
actor TokenManager {
    private let keychain = KeychainStore.shared
    private var refreshTask: Task<Bool, Never>?

    /// The current base URL for API requests.
    var baseURL: String {
        keychain.serverURL ?? ""
    }

    /// Get the current access token (may be expired).
    var accessToken: String? {
        keychain.accessToken
    }

    /// Store new tokens after login or refresh.
    func setTokens(access: String, refresh: String) {
        keychain.accessToken = access
        keychain.refreshToken = refresh
    }

    /// Clear all auth tokens (logout).
    func clearTokens() {
        keychain.clearAuthTokens()
        refreshTask?.cancel()
        refreshTask = nil
    }

    /// Attempt to refresh the access token using the stored refresh token.
    /// Returns `true` if refresh succeeded, `false` otherwise.
    /// Deduplicates concurrent calls — only one refresh request is in-flight at a time.
    func tryRefresh() async -> Bool {
        // If a refresh is already in progress, await it
        if let existing = refreshTask {
            return await existing.value
        }

        let task = Task<Bool, Never> { [self] in
            guard let refreshToken = keychain.refreshToken,
                  let baseURL = keychain.serverURL,
                  !baseURL.isEmpty
            else {
                clearTokensSync()
                return false
            }

            do {
                let url = URL(string: "\(baseURL)\(Endpoints.refresh)")!
                var request = URLRequest(url: url)
                request.httpMethod = "POST"
                request.setValue("application/json", forHTTPHeaderField: "Content-Type")

                let body = RefreshTokenRequest(refreshToken: refreshToken)
                let encoder = JSONEncoder()
                encoder.keyEncodingStrategy = .convertToSnakeCase
                request.httpBody = try encoder.encode(body)

                let (data, response) = try await URLSession.shared.data(for: request)

                guard let httpResponse = response as? HTTPURLResponse,
                      httpResponse.statusCode == 200
                else {
                    clearTokensSync()
                    return false
                }

                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                let authResponse = try decoder.decode(AuthResponse.self, from: data)

                keychain.accessToken = authResponse.accessToken
                keychain.refreshToken = authResponse.refreshToken
                return true
            } catch {
                clearTokensSync()
                return false
            }
        }

        refreshTask = task
        let result = await task.value
        refreshTask = nil
        return result
    }

    /// Synchronous clear (used within Task closures where we're already isolated).
    private func clearTokensSync() {
        keychain.clearAuthTokens()
    }
}

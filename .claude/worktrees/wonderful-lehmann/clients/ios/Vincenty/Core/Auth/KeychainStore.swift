import Foundation
import Security

/// Secure storage for tokens and device identifiers using the iOS Keychain.
/// Uses `kSecAttrAccessibleAfterFirstUnlock` so background location can access tokens.
final class KeychainStore: Sendable {
    static let shared = KeychainStore()

    private let service = "com.vincenty.app"

    private init() {}

    // MARK: - Typed Accessors

    var accessToken: String? {
        get { loadString(key: "access_token") }
        set {
            if let newValue { save(key: "access_token", string: newValue) }
            else { delete(key: "access_token") }
        }
    }

    var refreshToken: String? {
        get { loadString(key: "refresh_token") }
        set {
            if let newValue { save(key: "refresh_token", string: newValue) }
            else { delete(key: "refresh_token") }
        }
    }

    var deviceId: String? {
        get { loadString(key: "device_id") }
        set {
            if let newValue { save(key: "device_id", string: newValue) }
            else { delete(key: "device_id") }
        }
    }

    var serverURL: String? {
        get { loadString(key: "server_url") }
        set {
            if let newValue { save(key: "server_url", string: newValue) }
            else { delete(key: "server_url") }
        }
    }

    /// Clear all auth tokens (logout). Preserves server URL and device ID.
    func clearAuthTokens() {
        delete(key: "access_token")
        delete(key: "refresh_token")
    }

    /// Clear everything (full reset).
    func clearAll() {
        delete(key: "access_token")
        delete(key: "refresh_token")
        delete(key: "device_id")
        delete(key: "server_url")
    }

    // MARK: - Low-Level Keychain Operations

    private func save(key: String, string: String) {
        guard let data = string.data(using: .utf8) else { return }
        save(key: key, data: data)
    }

    private func loadString(key: String) -> String? {
        guard let data = load(key: key) else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private func save(key: String, data: Data) {
        // Delete existing item first (SecItemUpdate is finicky)
        delete(key: key)

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock,
        ]

        SecItemAdd(query as CFDictionary, nil)
    }

    private func load(key: String) -> Data? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess else { return nil }
        return result as? Data
    }

    @discardableResult
    private func delete(key: String) -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]

        return SecItemDelete(query as CFDictionary) == errSecSuccess
    }
}

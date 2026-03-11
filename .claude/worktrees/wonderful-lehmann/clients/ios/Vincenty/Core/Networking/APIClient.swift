import Foundation

/// HTTP client for the Vincenty REST API.
/// Mirrors the web client's `ApiClient` class — auto-injects Bearer tokens,
/// retries on 401 with token refresh, and handles the standard error envelope.
final class APIClient: @unchecked Sendable {
    static let shared = APIClient()

    let tokenManager = TokenManager()

    private let session: URLSession
    private let encoder: JSONEncoder
    private let decoder: JSONDecoder

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 300
        self.session = URLSession(configuration: config)

        self.encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase

        self.decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
    }

    // MARK: - Public Methods

    func get<T: Decodable & Sendable>(
        _ path: String,
        params: [String: String]? = nil
    ) async throws -> T {
        try await request(path: path, method: "GET", params: params)
    }

    func post<T: Decodable & Sendable>(
        _ path: String,
        body: (any Encodable & Sendable)? = nil,
        params: [String: String]? = nil
    ) async throws -> T {
        try await request(path: path, method: "POST", body: body, params: params)
    }

    func put<T: Decodable & Sendable>(
        _ path: String,
        body: (any Encodable & Sendable)? = nil,
        params: [String: String]? = nil
    ) async throws -> T {
        try await request(path: path, method: "PUT", body: body, params: params)
    }

    func delete(_ path: String, params: [String: String]? = nil) async throws {
        let _: EmptyResponse = try await request(
            path: path, method: "DELETE", params: params)
    }

    /// Post pre-encoded JSON data (bypasses the key encoding strategy).
    ///
    /// Used for WebAuthn requests where the W3C spec requires exact camelCase
    /// keys (`clientDataJSON`, `authenticatorData`, `rawId`, etc.) that would be
    /// mangled by the default `convertToSnakeCase` encoder.
    func postRawJSON<T: Decodable & Sendable>(
        _ path: String,
        jsonData: Data
    ) async throws -> T {
        try await request(path: path, method: "POST", rawBody: jsonData)
    }

    /// Upload multipart form data (for file uploads).
    func upload<T: Decodable & Sendable>(
        _ path: String,
        formData: MultipartFormData,
        method: String = "PUT"
    ) async throws -> T {
        let url = try buildURL(path: path)
        var request = URLRequest(url: url)
        request.httpMethod = method

        let (data, contentType) = formData.finalize()
        request.httpBody = data
        request.setValue(contentType, forHTTPHeaderField: "Content-Type")

        // Inject auth token
        if let token = await tokenManager.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let logger = AppLogger.shared
        logger.debug(.api, "\(method) \(path) [multipart]")
        let start = Date()

        let (responseData, response) = try await session.data(for: request)

        let elapsed = Int(Date().timeIntervalSince(start) * 1000)
        let status = (response as? HTTPURLResponse)?.statusCode ?? 0

        // Auto-refresh on 401
        if let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 401 {
            logger.info(.api, "\(method) \(path) [multipart] → 401, refreshing token")
            let refreshed = await tokenManager.tryRefresh()
            if refreshed {
                if let newToken = await tokenManager.accessToken {
                    request.setValue("Bearer \(newToken)", forHTTPHeaderField: "Authorization")
                }
                let (retryData, retryResponse) = try await session.data(for: request)
                do {
                    let result: T = try handleResponse(data: retryData, response: retryResponse)
                    let s2 = (retryResponse as? HTTPURLResponse)?.statusCode ?? 0
                    logger.info(.api, "\(method) \(path) [multipart] → \(s2) (\(elapsed)ms)")
                    return result
                } catch {
                    logger.error(.api, "\(method) \(path) [multipart] → failed after token refresh",
                                 detail: error.localizedDescription)
                    throw error
                }
            }
            logger.warning(.api, "\(method) \(path) [multipart] — token refresh failed")
        }

        do {
            let result: T = try handleResponse(data: responseData, response: response)
            logger.info(.api, "\(method) \(path) [multipart] → \(status) (\(elapsed)ms)")
            return result
        } catch {
            logger.error(.api, "\(method) \(path) [multipart] → \(status) (\(elapsed)ms)",
                         detail: error.localizedDescription)
            throw error
        }
    }

    /// Download raw data (for GPX/CSV/JSON export).
    func download(_ path: String, params: [String: String]? = nil) async throws -> Data {
        let url = try buildURL(path: path, params: params)
        var request = URLRequest(url: url)
        request.httpMethod = "GET"

        if let token = await tokenManager.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let logger = AppLogger.shared
        logger.debug(.api, "GET \(path) [download]")
        let start = Date()

        let (data, response) = try await session.data(for: request)

        let elapsed = Int(Date().timeIntervalSince(start) * 1000)
        let status = (response as? HTTPURLResponse)?.statusCode ?? 0

        guard let httpResponse = response as? HTTPURLResponse,
              (200 ... 299).contains(httpResponse.statusCode)
        else {
            logger.error(.api, "GET \(path) [download] → \(status) (\(elapsed)ms)")
            throw APIError(status: status, message: "Download failed")
        }

        logger.info(.api, "GET \(path) [download] → \(status) (\(elapsed)ms), \(data.count) bytes")
        return data
    }

    /// Build a full URL with auth token as query parameter (for attachments/avatars).
    func authenticatedURL(path: String) async -> URL? {
        guard let baseURL = KeychainStore.shared.serverURL,
              let token = await tokenManager.accessToken
        else { return nil }

        var components = URLComponents(string: "\(baseURL)\(path)")
        components?.queryItems = [URLQueryItem(name: "token", value: token)]
        return components?.url
    }

    // MARK: - Private

    private func request<T: Decodable>(
        path: String,
        method: String,
        body: (any Encodable)? = nil,
        rawBody: Data? = nil,
        params: [String: String]? = nil,
        retry: Bool = true
    ) async throws -> T {
        let url = try buildURL(path: path, params: params)
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        // Inject auth token
        if let token = await tokenManager.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        // Encode body — rawBody takes precedence (used by WebAuthn endpoints)
        if let rawBody {
            request.httpBody = rawBody
        } else if let body {
            request.httpBody = try encoder.encode(AnyEncodable(body))
        }

        let logger = AppLogger.shared
        logger.debug(.api, "\(method) \(path)")
        let start = Date()

        let (data, response) = try await session.data(for: request)

        let elapsed = Int(Date().timeIntervalSince(start) * 1000)
        let status = (response as? HTTPURLResponse)?.statusCode ?? 0

        // Auto-refresh on 401 (once)
        if let httpResponse = response as? HTTPURLResponse,
           httpResponse.statusCode == 401, retry
        {
            logger.info(.api, "\(method) \(path) → 401, refreshing token")
            let refreshed = await tokenManager.tryRefresh()
            if refreshed {
                return try await self.request(
                    path: path, method: method, body: body,
                    rawBody: rawBody, params: params, retry: false)
            }
            logger.warning(.api, "\(method) \(path) — token refresh failed, unauthorized")
        }

        do {
            let result: T = try handleResponse(data: data, response: response)
            logger.info(.api, "\(method) \(path) → \(status) (\(elapsed)ms)")
            return result
        } catch {
            logger.error(.api, "\(method) \(path) → \(status) (\(elapsed)ms)",
                         detail: error.localizedDescription)
            throw error
        }
    }

    private func handleResponse<T: Decodable>(data: Data, response: URLResponse) throws -> T {
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError(status: 0, message: "Invalid response")
        }

        // 204 No Content
        if httpResponse.statusCode == 204 {
            if T.self == EmptyResponse.self {
                return EmptyResponse() as! T
            }
            throw APIError(status: 204, message: "No content")
        }

        // Error responses
        guard (200 ... 299).contains(httpResponse.statusCode) else {
            if let errorResponse = try? decoder.decode(APIErrorResponse.self, from: data) {
                throw APIError(status: httpResponse.statusCode, response: errorResponse)
            }
            throw APIError(
                status: httpResponse.statusCode,
                message: HTTPURLResponse.localizedString(forStatusCode: httpResponse.statusCode))
        }

        // Success
        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw APIError.decodingError(error)
        }
    }

    private func buildURL(path: String, params: [String: String]? = nil) throws -> URL {
        guard let baseURL = KeychainStore.shared.serverURL, !baseURL.isEmpty else {
            throw APIError(status: 0, code: "no_server", message: "Server URL not configured")
        }

        var components = URLComponents(string: "\(baseURL)\(path)")

        if let params, !params.isEmpty {
            components?.queryItems = params.map { URLQueryItem(name: $0.key, value: $0.value) }
        }

        guard let url = components?.url else {
            throw APIError(status: 0, message: "Invalid URL: \(baseURL)\(path)")
        }

        return url
    }
}

// MARK: - Helpers

/// Placeholder for endpoints that return 204 or where we don't care about the body.
struct EmptyResponse: Decodable, Sendable {
    init() {}
    init(from decoder: Decoder) throws {}
}

/// Type-erased Encodable wrapper for dynamic body encoding.
private struct AnyEncodable: Encodable {
    private let encode: (Encoder) throws -> Void

    init(_ value: any Encodable) {
        self.encode = value.encode
    }

    func encode(to encoder: Encoder) throws {
        try encode(encoder)
    }
}

// MARK: - Shared JSONDecoder

extension JSONDecoder {
    /// Shared decoder with snake_case → camelCase key conversion.
    /// Used by WebSocket message handlers to decode server payloads directly from raw bytes.
    static let snakeCase: JSONDecoder = {
        let d = JSONDecoder()
        d.keyDecodingStrategy = .convertFromSnakeCase
        return d
    }()
}

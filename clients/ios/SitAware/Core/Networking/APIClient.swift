import Foundation

/// HTTP client for the SitAware REST API.
/// Mirrors the web client's `ApiClient` class — auto-injects Bearer tokens,
/// retries on 401 with token refresh, and handles the standard error envelope.
final class APIClient: Sendable {
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

        let (responseData, response) = try await session.data(for: request)

        // Auto-refresh on 401
        if let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 401 {
            let refreshed = await tokenManager.tryRefresh()
            if refreshed {
                if let newToken = await tokenManager.accessToken {
                    request.setValue("Bearer \(newToken)", forHTTPHeaderField: "Authorization")
                }
                let (retryData, retryResponse) = try await session.data(for: request)
                return try handleResponse(data: retryData, response: retryResponse)
            }
        }

        return try handleResponse(data: responseData, response: response)
    }

    /// Download raw data (for GPX/CSV/JSON export).
    func download(_ path: String, params: [String: String]? = nil) async throws -> Data {
        let url = try buildURL(path: path, params: params)
        var request = URLRequest(url: url)
        request.httpMethod = "GET"

        if let token = await tokenManager.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let (data, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse,
              (200 ... 299).contains(httpResponse.statusCode)
        else {
            let httpResponse = response as? HTTPURLResponse
            throw APIError(
                status: httpResponse?.statusCode ?? 0,
                message: "Download failed")
        }

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

        // Encode body
        if let body {
            request.httpBody = try encoder.encode(AnyEncodable(body))
        }

        let (data, response) = try await session.data(for: request)

        // Auto-refresh on 401 (once)
        if let httpResponse = response as? HTTPURLResponse,
           httpResponse.statusCode == 401, retry
        {
            let refreshed = await tokenManager.tryRefresh()
            if refreshed {
                return try await self.request(
                    path: path, method: method, body: body,
                    params: params, retry: false)
            }
        }

        return try handleResponse(data: data, response: response)
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

import Foundation

/// Typed API error with HTTP status code and server error message.
struct APIError: Error, LocalizedError, Sendable {
    let status: Int
    let code: String
    let message: String

    var errorDescription: String? { message }

    /// Common HTTP error statuses.
    var isUnauthorized: Bool { status == 401 }
    var isForbidden: Bool { status == 403 }
    var isNotFound: Bool { status == 404 }
    var isConflict: Bool { status == 409 }
    var isServerError: Bool { status >= 500 }

    /// Create from an API error response body.
    init(status: Int, response: APIErrorResponse) {
        self.status = status
        self.code = response.error.code
        self.message = response.error.message
    }

    /// Create with explicit values.
    init(status: Int, code: String = "", message: String) {
        self.status = status
        self.code = code
        self.message = message
    }

    /// Network/transport error (no HTTP status).
    static func networkError(_ error: Error) -> APIError {
        APIError(status: 0, code: "network_error", message: error.localizedDescription)
    }

    /// JSON decoding error.
    static func decodingError(_ error: Error) -> APIError {
        APIError(status: 0, code: "decoding_error", message: "Failed to decode response: \(error.localizedDescription)")
    }
}

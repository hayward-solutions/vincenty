import Foundation

// MARK: - Generic API Envelope

/// Paginated list response matching the API's `{ data, total, page, page_size }` envelope.
struct ListResponse<T: Codable & Sendable>: Codable, Sendable {
    let data: [T]
    let total: Int
    let page: Int
    let pageSize: Int
}

// MARK: - API Error

/// Matches the API's `{ error: { code, message } }` error envelope.
struct APIErrorResponse: Codable, Sendable {
    let error: Detail

    struct Detail: Codable, Sendable {
        let code: String
        let message: String
    }
}

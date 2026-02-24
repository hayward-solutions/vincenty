import Foundation

/// Builds multipart/form-data request bodies for file uploads.
struct MultipartFormData: Sendable {
    private let boundary: String
    private var parts: [Part] = []

    init() {
        self.boundary = "SitAware-\(UUID().uuidString)"
    }

    /// Append a text field.
    mutating func append(name: String, value: String) {
        parts.append(.text(name: name, value: value))
    }

    /// Append a file field.
    mutating func append(name: String, data: Data, filename: String, mimeType: String) {
        parts.append(.file(name: name, data: data, filename: filename, mimeType: mimeType))
    }

    /// Build the final body data and content type header value.
    func finalize() -> (data: Data, contentType: String) {
        var body = Data()

        for part in parts {
            body.append("--\(boundary)\r\n".data(using: .utf8)!)

            switch part {
            case .text(let name, let value):
                body.append("Content-Disposition: form-data; name=\"\(name)\"\r\n".data(using: .utf8)!)
                body.append("\r\n".data(using: .utf8)!)
                body.append(value.data(using: .utf8)!)
                body.append("\r\n".data(using: .utf8)!)

            case .file(let name, let data, let filename, let mimeType):
                body.append(
                    "Content-Disposition: form-data; name=\"\(name)\"; filename=\"\(filename)\"\r\n"
                        .data(using: .utf8)!)
                body.append("Content-Type: \(mimeType)\r\n".data(using: .utf8)!)
                body.append("\r\n".data(using: .utf8)!)
                body.append(data)
                body.append("\r\n".data(using: .utf8)!)
            }
        }

        body.append("--\(boundary)--\r\n".data(using: .utf8)!)

        return (data: body, contentType: "multipart/form-data; boundary=\(boundary)")
    }

    // MARK: - Private

    private enum Part: Sendable {
        case text(name: String, value: String)
        case file(name: String, data: Data, filename: String, mimeType: String)
    }
}

import Foundation

extension Data {
    /// Decode a Base64URL-encoded string (RFC 4648 §5) into Data.
    ///
    /// Base64URL uses `-` and `_` instead of `+` and `/`, and omits padding.
    /// This matches the encoding used by the WebAuthn/FIDO2 spec.
    init?(base64URLEncoded string: String) {
        var base64 = string
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")

        // Pad to a multiple of 4
        let remainder = base64.count % 4
        if remainder > 0 {
            base64 += String(repeating: "=", count: 4 - remainder)
        }

        self.init(base64Encoded: base64)
    }

    /// Encode Data as a Base64URL string (RFC 4648 §5, no padding).
    ///
    /// Produces the same output as the web client's `bufferToBase64URL()`.
    func base64URLEncodedString() -> String {
        base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }
}

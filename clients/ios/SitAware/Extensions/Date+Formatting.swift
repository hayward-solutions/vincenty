import Foundation

extension Date {
    /// Short relative time: "Just now", "5m ago", "2h ago", "3d ago", etc.
    /// Matches the web client's `relativeTime` helper.
    var relativeShort: String {
        let now = Date()
        let interval = now.timeIntervalSince(self)

        if interval < 60 { return "Just now" }
        if interval < 3600 { return "\(Int(interval / 60))m ago" }
        if interval < 86400 { return "\(Int(interval / 3600))h ago" }
        if interval < 2_592_000 { return "\(Int(interval / 86400))d ago" }
        if interval < 31_536_000 { return "\(Int(interval / 2_592_000))mo ago" }
        return "\(Int(interval / 31_536_000))y ago"
    }

    /// Short date + time: "Feb 24, 14:32"
    var shortDateTime: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "MMM d, HH:mm"
        return formatter.string(from: self)
    }

    /// Time only: "14:32:05"
    var timeOnly: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss"
        return formatter.string(from: self)
    }

    /// Locale-aware short date: "Feb 24, 2026"
    var shortDate: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .none
        return formatter.string(from: self)
    }
}

extension String {
    /// Parse an ISO 8601 date string to a Date.
    var iso8601Date: Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: self) { return date }
        // Retry without fractional seconds
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: self)
    }
}

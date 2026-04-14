import Foundation
import SwiftUI

// MARK: - Log Level

enum LogLevel: Int, Comparable, CaseIterable, Sendable {
    case debug = 0
    case info = 1
    case warning = 2
    case error = 3

    var label: String {
        switch self {
        case .debug:   return "DEBUG"
        case .info:    return "INFO"
        case .warning: return "WARN"
        case .error:   return "ERROR"
        }
    }

    var icon: String {
        switch self {
        case .debug:   return "magnifyingglass"
        case .info:    return "info.circle.fill"
        case .warning: return "exclamationmark.triangle.fill"
        case .error:   return "xmark.circle.fill"
        }
    }

    var color: Color {
        switch self {
        case .debug:   return .secondary
        case .info:    return .blue
        case .warning: return .orange
        case .error:   return .red
        }
    }

    static func < (lhs: LogLevel, rhs: LogLevel) -> Bool {
        lhs.rawValue < rhs.rawValue
    }
}

// MARK: - Log Category

enum LogCategory: String, CaseIterable, Sendable {
    case api      = "API"
    case ws       = "WS"
    case auth     = "Auth"
    case location = "Location"
    case sync     = "Sync"
    case app      = "App"
}

// MARK: - Log Entry

struct LogEntry: Identifiable, Sendable {
    let id: UUID
    let timestamp: Date
    let level: LogLevel
    let category: LogCategory
    let message: String
    let detail: String?

    init(
        level: LogLevel,
        category: LogCategory,
        message: String,
        detail: String? = nil
    ) {
        self.id = UUID()
        self.timestamp = Date()
        self.level = level
        self.category = category
        self.message = message
        self.detail = detail
    }
}

// MARK: - AppLogger

/// In-memory ring-buffer logger. All mutation is isolated to MainActor.
/// nonisolated convenience methods allow safe calls from any thread/actor.
@Observable @MainActor
final class AppLogger {

    nonisolated static let shared = AppLogger()

    /// All entries (newest last), capped at `maxEntries`.
    private(set) var entries: [LogEntry] = []

    private let maxEntries = 1_000

    private let exportFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd HH:mm:ss.SSS"
        return f
    }()

    nonisolated private init() {}

    // MARK: - Core (MainActor)

    func log(
        _ level: LogLevel,
        _ category: LogCategory,
        _ message: String,
        detail: String? = nil
    ) {
        let entry = LogEntry(level: level, category: category, message: message, detail: detail)
        entries.append(entry)
        if entries.count > maxEntries {
            entries.removeFirst(entries.count - maxEntries)
        }
    }

    // MARK: - Convenience (nonisolated — safe from any actor or thread)

    nonisolated func debug(_ category: LogCategory, _ message: String, detail: String? = nil) {
        Task { @MainActor in AppLogger.shared.log(.debug, category, message, detail: detail) }
    }

    nonisolated func info(_ category: LogCategory, _ message: String, detail: String? = nil) {
        Task { @MainActor in AppLogger.shared.log(.info, category, message, detail: detail) }
    }

    nonisolated func warning(_ category: LogCategory, _ message: String, detail: String? = nil) {
        Task { @MainActor in AppLogger.shared.log(.warning, category, message, detail: detail) }
    }

    nonisolated func error(_ category: LogCategory, _ message: String, detail: String? = nil) {
        Task { @MainActor in AppLogger.shared.log(.error, category, message, detail: detail) }
    }

    // MARK: - Management

    func clear() {
        entries.removeAll()
    }

    /// Format all current entries as plain text suitable for sharing.
    func export() -> String {
        entries.map { e in
            let ts = exportFormatter.string(from: e.timestamp)
            var line = "[\(ts)] [\(e.level.label)] [\(e.category.rawValue)] \(e.message)"
            if let d = e.detail { line += "\n    \(d)" }
            return line
        }.joined(separator: "\n")
    }
}

import Foundation

/// Replay speed multiplier options.
enum ReplaySpeed: Double, CaseIterable, Sendable {
    case x1 = 1
    case x2 = 2
    case x5 = 5
    case x10 = 10
    case x30 = 30
    case x60 = 60

    var label: String {
        switch self {
        case .x1: return "1x"
        case .x2: return "2x"
        case .x5: return "5x"
        case .x10: return "10x"
        case .x30: return "30x"
        case .x60: return "60x"
        }
    }
}

/// Replay scope: which location history to fetch.
enum ReplayScope: Sendable {
    case all
    case group(String)
    case user(String)
}

/// Manages location history replay with time-based animation.
///
/// Mirrors the web client's replay functionality:
/// - Fetches location history for a time range
/// - Animates through history with configurable speed
/// - Uses a 30fps Timer (1s real = 1min replay at 1x)
/// - Provides current playback time for marker interpolation
@Observable @MainActor
final class ReplayViewModel {

    // MARK: - State

    private(set) var isActive = false
    private(set) var isPlaying = false
    private(set) var isLoading = false
    /// Set when a fetch fails so the UI can show a message.
    private(set) var errorMessage: String? = nil

    var speed: ReplaySpeed = .x1

    /// The replay time range.
    var startDate: Date = Calendar.current.date(byAdding: .hour, value: -1, to: Date()) ?? Date()
    var endDate: Date = Date()

    /// Current playback position (between startDate and endDate).
    private(set) var currentTime: Date = Date()

    /// Progress from 0.0 to 1.0.
    var progress: Double {
        let total = endDate.timeIntervalSince(startDate)
        guard total > 0 else { return 0 }
        let elapsed = currentTime.timeIntervalSince(startDate)
        return min(max(elapsed / total, 0), 1)
    }

    /// Raw history data points (all of them, regardless of playback position).
    private(set) var historyEntries: [LocationHistoryEntry] = []

    /// Pre-parsed dates parallel to `historyEntries` — avoids allocating a
    /// new ISO8601DateFormatter for every entry on every animation frame.
    private(set) var parsedEntryDates: [Date] = []

    /// Entries visible at the current playback time (recorded_at <= currentTime).
    var visibleEntries: [LocationHistoryEntry] {
        zip(historyEntries, parsedEntryDates)
            .filter { _, date in date <= currentTime }
            .map(\.0)
    }

    // MARK: - Private

    private let api = APIClient.shared
    private var displayLink: Timer?
    private static let isoFormatter: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()

    // MARK: - Lifecycle

    /// Start a replay session: fetch history for the configured time range and scope.
    ///
    /// `isActive` is set to `true` only after a successful fetch so the setup
    /// panel's loading spinner stays visible during the network call. On failure,
    /// `isActive` remains `false` and `errorMessage` is populated so the user
    /// can retry without reopening the panel.
    func start(scope: ReplayScope) async {
        isLoading = true
        errorMessage = nil
        currentTime = startDate

        // API expects "from" and "to" as RFC3339 strings.
        let isoStart = Self.isoFormatter.string(from: startDate)
        let isoEnd   = Self.isoFormatter.string(from: endDate)
        let params   = ["from": isoStart, "to": isoEnd]

        do {
            let entries: [LocationHistoryEntry]

            switch scope {
            case .all:
                entries = try await api.get(Endpoints.locationsHistory, params: params)
            case .group(let groupId):
                entries = try await api.get(
                    Endpoints.groupLocationsHistory(groupId), params: params)
            case .user(let userId):
                entries = try await api.get(
                    Endpoints.userLocationsHistory(userId), params: params)
            }

            historyEntries = entries
            // Pre-parse dates once so visibleEntries filter is O(n) with no allocations.
            parsedEntryDates = entries.map {
                Self.isoFormatter.date(from: $0.recordedAt) ?? Date.distantPast
            }
            // Only activate once data is ready — keeps the setup panel visible
            // with a loading spinner for the full duration of the fetch.
            isActive = true
        } catch {
            historyEntries = []
            parsedEntryDates = []
            errorMessage = error.localizedDescription
            // isActive stays false so the setup panel remains visible for retry.
        }

        isLoading = false
    }

    /// Stop replay and clear all data.
    func stop() {
        pause()
        isActive = false
        historyEntries = []
        parsedEntryDates = []
        errorMessage = nil
        currentTime = startDate
    }

    /// Start/resume playback.
    func play() {
        guard isActive, !isPlaying else { return }
        isPlaying = true

        // Timer fires at ~30fps; each tick advances currentTime by (speed * interval * 60)
        // so 1 real second = 1 replay minute at 1x speed.
        displayLink = Timer.scheduledTimer(withTimeInterval: 1.0 / 30.0, repeats: true) {
            [weak self] _ in
            Task { @MainActor [weak self] in
                self?.tick()
            }
        }
    }

    /// Pause playback.
    func pause() {
        isPlaying = false
        displayLink?.invalidate()
        displayLink = nil
    }

    /// Seek to a specific progress value (0.0 – 1.0).
    func seek(to progress: Double) {
        let total = endDate.timeIntervalSince(startDate)
        let offset = total * min(max(progress, 0), 1)
        currentTime = startDate.addingTimeInterval(offset)
    }

    // MARK: - Playback Tick

    private func tick() {
        guard isPlaying else { return }

        // Advance by (1/30 sec * speed * 60) seconds of replay time.
        let replayAdvance = (1.0 / 30.0) * speed.rawValue * 60.0
        currentTime = currentTime.addingTimeInterval(replayAdvance)

        // Auto-stop at end.
        if currentTime >= endDate {
            currentTime = endDate
            pause()
        }
    }
}

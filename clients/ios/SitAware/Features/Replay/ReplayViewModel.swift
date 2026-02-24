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
/// - Uses `CADisplayLink`-equivalent timer (1s real = 1min replay at 1x)
/// - Provides current playback time for marker interpolation
@Observable @MainActor
final class ReplayViewModel {

    // MARK: - State

    private(set) var isActive = false
    private(set) var isPlaying = false
    private(set) var isLoading = false

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

    /// History data points.
    private(set) var historyEntries: [LocationHistoryEntry] = []

    /// Entries visible at the current playback time.
    var visibleEntries: [LocationHistoryEntry] {
        historyEntries.filter { entry in
            guard let date = ISO8601DateFormatter().date(from: entry.recordedAt) else { return false }
            return date <= currentTime
        }
    }

    // MARK: - Private

    private let api = APIClient.shared
    private var displayLink: Timer?

    // MARK: - Lifecycle

    /// Start a replay session: fetch history and begin playback.
    func start(scope: ReplayScope) async {
        isActive = true
        isLoading = true
        currentTime = startDate

        let isoStart = ISO8601DateFormatter().string(from: startDate)
        let isoEnd = ISO8601DateFormatter().string(from: endDate)
        let params = ["start": isoStart, "end": isoEnd]

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
        } catch {
            historyEntries = []
        }

        isLoading = false
    }

    /// Stop replay and clear data.
    func stop() {
        pause()
        isActive = false
        historyEntries = []
        currentTime = startDate
    }

    /// Start/resume playback.
    func play() {
        guard isActive, !isPlaying else { return }
        isPlaying = true

        // Timer fires at ~30fps; each tick advances currentTime by (speed * interval * 60)
        // 1 second real = 1 minute replay at 1x speed
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

    /// Seek to a specific progress (0.0 - 1.0).
    func seek(to progress: Double) {
        let total = endDate.timeIntervalSince(startDate)
        let offset = total * min(max(progress, 0), 1)
        currentTime = startDate.addingTimeInterval(offset)
    }

    // MARK: - Playback Tick

    private func tick() {
        guard isPlaying else { return }

        // Advance by (1/30 sec * speed * 60) seconds of replay time
        let replayAdvance = (1.0 / 30.0) * speed.rawValue * 60.0
        currentTime = currentTime.addingTimeInterval(replayAdvance)

        // Stop at end
        if currentTime >= endDate {
            currentTime = endDate
            pause()
        }
    }
}

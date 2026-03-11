import SwiftUI
import LiveKit

@Observable @MainActor
final class LiveMonitorViewModel {

    // MARK: - State
    private(set) var groups: [Group] = []
    private(set) var allFeeds: [VideoFeed] = []
    private(set) var isLoading = false
    private(set) var error: String?

    // Active watching sessions: feedId -> Room
    private(set) var watchingSessions: [String: Room] = [:]

    private let api = APIClient.shared

    // MARK: - Fetch

    func loadAllFeeds() async {
        isLoading = true
        error = nil
        do {
            groups = try await api.get(Endpoints.usersMeGroups)

            var feeds: [VideoFeed] = []
            for group in groups {
                let groupFeeds: [VideoFeed] = try await api.get(Endpoints.groupFeeds(group.id))
                feeds.append(contentsOf: groupFeeds)
            }
            allFeeds = feeds
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    var activeFeeds: [VideoFeed] {
        allFeeds.filter { $0.isActive }
    }

    // MARK: - Watch a feed

    func watchFeed(_ feedId: String) async {
        guard watchingSessions[feedId] == nil else { return }
        do {
            let response: JoinRoomResponse = try await api.get(Endpoints.feedView(feedId))
            let room = Room()
            try await room.connect(url: response.url, token: response.token)
            watchingSessions[feedId] = room
        } catch {
            self.error = "Failed to connect: \(error.localizedDescription)"
        }
    }

    func stopWatching(_ feedId: String) async {
        guard let room = watchingSessions[feedId] else { return }
        await room.disconnect()
        watchingSessions.removeValue(forKey: feedId)
    }

    func stopAll() async {
        for (_, room) in watchingSessions {
            await room.disconnect()
        }
        watchingSessions.removeAll()
    }

    func groupName(for feed: VideoFeed) -> String {
        groups.first(where: { $0.id == feed.groupId })?.name ?? "Unknown"
    }
}

struct LiveMonitorView: View {
    @State private var viewModel = LiveMonitorViewModel()

    // Adaptive grid: 2 columns on compact, 3 on regular
    @Environment(\.horizontalSizeClass) private var sizeClass

    private var columns: [GridItem] {
        let count = sizeClass == .regular ? 3 : 2
        return Array(repeating: GridItem(.flexible(), spacing: 8), count: count)
    }

    var body: some View {
        Group {
            if viewModel.isLoading && viewModel.allFeeds.isEmpty {
                ProgressView("Loading feeds...")
            } else if viewModel.activeFeeds.isEmpty {
                emptyState
            } else {
                feedGrid
            }
        }
        .navigationTitle("Live")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    Task { await viewModel.loadAllFeeds() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
            }
        }
        .task {
            await viewModel.loadAllFeeds()
            for feed in viewModel.activeFeeds {
                await viewModel.watchFeed(feed.id)
            }
        }
        .onDisappear {
            Task { await viewModel.stopAll() }
        }
    }

    private var emptyState: some View {
        ContentUnavailableView {
            Label("No Active Feeds", systemImage: "video.slash")
        } description: {
            Text("There are no active video feeds in your groups. Add feeds from the Feeds tab to start monitoring.")
        }
    }

    private var feedGrid: some View {
        ScrollView {
            LazyVGrid(columns: columns, spacing: 8) {
                ForEach(viewModel.activeFeeds) { feed in
                    feedTile(feed)
                }
            }
            .padding()
        }
    }

    @ViewBuilder
    private func feedTile(_ feed: VideoFeed) -> some View {
        ZStack(alignment: .topLeading) {
            // Video content
            if let room = viewModel.watchingSessions[feed.id],
               let participant = room.remoteParticipants.values.first,
               let track = participant.firstCameraVideoTrack {
                VideoView(track)
                    .aspectRatio(16/9, contentMode: .fit)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
            } else {
                // Placeholder / connecting
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color(.systemGray6))
                    .aspectRatio(16/9, contentMode: .fit)
                    .overlay {
                        VStack(spacing: 6) {
                            ProgressView()
                            Text("Connecting...")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
            }

            // Overlay: labels
            VStack(alignment: .leading) {
                HStack(spacing: 4) {
                    // Type badge
                    Text(feed.feedType.uppercased())
                        .font(.system(size: 9, weight: .bold))
                        .padding(.horizontal, 5)
                        .padding(.vertical, 2)
                        .background(.black.opacity(0.6))
                        .foregroundStyle(.white)
                        .clipShape(Capsule())

                    // Live indicator
                    HStack(spacing: 3) {
                        Circle()
                            .fill(.red)
                            .frame(width: 5, height: 5)
                        Text("LIVE")
                            .font(.system(size: 9, weight: .bold))
                    }
                    .padding(.horizontal, 5)
                    .padding(.vertical, 2)
                    .background(.black.opacity(0.6))
                    .foregroundStyle(.white)
                    .clipShape(Capsule())
                }
                .padding(6)

                Spacer()

                // Bottom: feed name + group
                VStack(alignment: .leading, spacing: 1) {
                    Text(feed.name)
                        .font(.caption.bold())
                        .foregroundStyle(.white)
                    Text(viewModel.groupName(for: feed))
                        .font(.caption2)
                        .foregroundStyle(.white.opacity(0.8))
                }
                .padding(6)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(.linearGradient(
                    colors: [.clear, .black.opacity(0.7)],
                    startPoint: .top,
                    endPoint: .bottom
                ))
            }
        }
    }
}

import Foundation
import LiveKit

@Observable @MainActor
final class FeedViewModel {

    // MARK: - State
    private(set) var feeds: [VideoFeed] = []
    private(set) var isLoading = false
    private(set) var error: String?

    // Active feed viewing
    private(set) var room: Room?
    private(set) var activeFeed: VideoFeed?
    private(set) var isViewing = false

    private let api = APIClient.shared

    // MARK: - Fetch

    func fetchFeeds(_ groupId: String) async {
        isLoading = true
        error = nil
        do {
            feeds = try await api.get(Endpoints.groupFeeds(groupId))
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    // MARK: - Feed Actions

    func createFeed(name: String, feedType: String, sourceUrl: String? = nil, groupId: String) async -> VideoFeed? {
        let request = CreateVideoFeedRequest(name: name, feedType: feedType, sourceUrl: sourceUrl, groupId: groupId)
        do {
            let feed: VideoFeed = try await api.post(Endpoints.feeds, body: request)
            await fetchFeeds(groupId)
            return feed
        } catch {
            self.error = error.localizedDescription
            return nil
        }
    }

    func startFeed(_ feedId: String) async -> VideoFeedStartResponse? {
        do {
            return try await api.post(Endpoints.feedStart(feedId))
        } catch {
            self.error = error.localizedDescription
            return nil
        }
    }

    func stopFeed(_ feedId: String) async {
        do {
            let _: EmptyResponse = try await api.post(Endpoints.feedStop(feedId))
        } catch {
            self.error = error.localizedDescription
        }
    }

    func deleteFeed(_ feedId: String, groupId: String) async {
        do {
            try await api.delete(Endpoints.feed(feedId))
            await fetchFeeds(groupId)
        } catch {
            self.error = error.localizedDescription
        }
    }

    // MARK: - View Feed (subscribe-only)

    func viewFeed(_ feedId: String) async {
        do {
            let response: JoinRoomResponse = try await api.get(Endpoints.feedView(feedId))
            let newRoom = Room()
            try await newRoom.connect(url: response.url, token: response.token)
            room = newRoom
            if let feed = feeds.first(where: { $0.id == feedId }) {
                activeFeed = feed
            }
            isViewing = true
        } catch {
            self.error = error.localizedDescription
        }
    }

    func stopViewing() async {
        await room?.disconnect()
        room = nil
        activeFeed = nil
        isViewing = false
    }
}

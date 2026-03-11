import Foundation
import LiveKit

@Observable @MainActor
final class PTTViewModel {

    // MARK: - State
    private(set) var channels: [PTTChannel] = []
    private(set) var isLoading = false
    private(set) var error: String?

    // Active PTT session
    private(set) var activeChannel: PTTChannel?
    private(set) var room: Room?
    private(set) var isConnected = false
    private(set) var isTalking = false
    private(set) var floorHolder: (id: String, name: String)?

    private let api = APIClient.shared
    private var wsUnsubscribe: (() -> Void)?

    // MARK: - Fetch

    func fetchChannels(_ groupId: String) async {
        isLoading = true
        error = nil
        do {
            channels = try await api.get(Endpoints.groupPTTChannels(groupId))
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    // MARK: - Channel Actions

    func createChannel(_ groupId: String, name: String, isDefault: Bool = false) async -> PTTChannel? {
        let request = CreatePTTChannelRequest(name: name, isDefault: isDefault)
        do {
            let channel: PTTChannel = try await api.post(Endpoints.groupPTTChannels(groupId), body: request)
            await fetchChannels(groupId)
            return channel
        } catch {
            self.error = error.localizedDescription
            return nil
        }
    }

    func joinChannel(_ groupId: String, channelId: String) async {
        do {
            let response: JoinPTTChannelResponse = try await api.post(Endpoints.pttChannelJoin(groupId, channelId))
            activeChannel = response.channel
            let newRoom = Room()
            try await newRoom.connect(url: response.url, token: response.token)
            // Start with mic disabled (PTT = push to enable)
            try await newRoom.localParticipant.setMicrophone(enabled: false)
            room = newRoom
            isConnected = true
        } catch {
            self.error = error.localizedDescription
        }
    }

    func leaveChannel() async {
        await room?.disconnect()
        room = nil
        activeChannel = nil
        isConnected = false
        isTalking = false
        floorHolder = nil
    }

    func deleteChannel(_ groupId: String, channelId: String) async {
        do {
            try await api.delete(Endpoints.groupPTTChannel(groupId, channelId))
            await fetchChannels(groupId)
        } catch {
            self.error = error.localizedDescription
        }
    }

    // MARK: - PTT Floor Control

    func startTalking(webSocket: WebSocketService) async {
        guard let channel = activeChannel, let room else { return }
        webSocket.send(type: "ptt_floor_request", payload: ["channel_id": channel.id])
        try? await room.localParticipant.setMicrophone(enabled: true)
        isTalking = true
    }

    func stopTalking(webSocket: WebSocketService) async {
        guard let channel = activeChannel, let room else { return }
        try? await room.localParticipant.setMicrophone(enabled: false)
        webSocket.send(type: "ptt_floor_release", payload: ["channel_id": channel.id])
        isTalking = false
    }

    // MARK: - WebSocket

    func subscribeToFloorEvents(webSocket: WebSocketService) {
        wsUnsubscribe = webSocket.subscribe { [weak self] type, data in
            guard let self else { return }
            guard type == "ptt_floor_granted" || type == "ptt_floor_released" else { return }
            guard let event = try? JSONDecoder.snakeCase.decode(WSPTTFloorEventEnvelope.self, from: data) else { return }
            let payload = event.payload
            guard payload.channelId == self.activeChannel?.id else { return }
            Task { @MainActor in
                if payload.eventType == "floor_granted", let holderId = payload.holderId {
                    self.floorHolder = (id: holderId, name: payload.holderName ?? "Unknown")
                } else {
                    self.floorHolder = nil
                }
            }
        }
    }

    func unsubscribe() {
        wsUnsubscribe?()
        wsUnsubscribe = nil
    }
}

private struct WSPTTFloorEventEnvelope: Decodable {
    let payload: WSPTTFloorEvent
}

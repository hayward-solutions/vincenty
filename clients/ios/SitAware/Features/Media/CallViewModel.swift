import Foundation
import LiveKit

@Observable @MainActor
final class CallViewModel {

    // MARK: - State
    private(set) var activeCalls: [MediaRoom] = []
    private(set) var groupCalls: [MediaRoom] = []
    private(set) var isLoading = false
    private(set) var error: String?

    // Current active call
    private(set) var currentRoom: Room?
    private(set) var currentMediaRoom: MediaRoom?
    private(set) var isInCall = false
    private(set) var isVideoEnabled = false

    private let api = APIClient.shared
    private var wsUnsubscribe: (() -> Void)?

    // MARK: - Fetch

    func fetchActiveCalls() async {
        isLoading = true
        error = nil
        do {
            activeCalls = try await api.get(Endpoints.calls)
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    func fetchGroupCalls(_ groupId: String) async {
        isLoading = true
        do {
            groupCalls = try await api.get(Endpoints.groupCalls(groupId))
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    // MARK: - Call Actions

    func createCall(groupId: String? = nil, recipientId: String? = nil, name: String? = nil, videoEnabled: Bool = true) async -> JoinRoomResponse? {
        let request = CreateCallRequest(groupId: groupId, recipientId: recipientId, name: name, videoEnabled: videoEnabled)
        do {
            let response: JoinRoomResponse = try await api.post(Endpoints.calls, body: request)
            isVideoEnabled = videoEnabled
            await connectToRoom(response: response)
            return response
        } catch {
            self.error = error.localizedDescription
            return nil
        }
    }

    func joinCall(_ roomId: String) async {
        do {
            let response: JoinRoomResponse = try await api.post(Endpoints.callJoin(roomId))
            await connectToRoom(response: response)
        } catch {
            self.error = error.localizedDescription
        }
    }

    func leaveCall() async {
        guard let mediaRoom = currentMediaRoom else { return }
        do {
            let _: EmptyResponse = try await api.post(Endpoints.callLeave(mediaRoom.id))
        } catch {
            // Best-effort
        }
        await disconnectFromRoom()
    }

    func endCall(_ roomId: String) async {
        do {
            try await api.delete(Endpoints.call(roomId))
        } catch {
            self.error = error.localizedDescription
        }
        if currentMediaRoom?.id == roomId {
            await disconnectFromRoom()
        }
        await fetchActiveCalls()
    }

    // MARK: - Room Connection

    private func connectToRoom(response: JoinRoomResponse) async {
        let room = Room()
        do {
            try await room.connect(url: response.url, token: response.token)
            try await room.localParticipant.setMicrophone(enabled: true)
            if isVideoEnabled {
                try await room.localParticipant.setCamera(enabled: true)
            }
            currentRoom = room
            currentMediaRoom = response.room
            isInCall = true
        } catch {
            self.error = "Failed to connect: \(error.localizedDescription)"
        }
    }

    private func disconnectFromRoom() async {
        await currentRoom?.disconnect()
        currentRoom = nil
        currentMediaRoom = nil
        isInCall = false
        isVideoEnabled = false
    }

    // MARK: - Media Controls

    func toggleMicrophone() async {
        guard let room = currentRoom else { return }
        let enabled = room.localParticipant.isMicrophoneEnabled()
        try? await room.localParticipant.setMicrophone(enabled: !enabled)
    }

    func toggleCamera() async {
        guard let room = currentRoom else { return }
        let enabled = room.localParticipant.isCameraEnabled()
        try? await room.localParticipant.setCamera(enabled: !enabled)
        isVideoEnabled = !enabled
    }

    // MARK: - WebSocket

    func subscribeToCallEvents(webSocket: WebSocketService) {
        wsUnsubscribe = webSocket.subscribe { [weak self] type, data in
            guard let self else { return }
            guard type == "call_started" || type == "call_ended" else { return }
            Task { @MainActor in
                await self.fetchActiveCalls()
            }
        }
    }

    func unsubscribe() {
        wsUnsubscribe?()
        wsUnsubscribe = nil
    }
}

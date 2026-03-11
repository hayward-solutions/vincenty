import Foundation

/// Central view model for the messaging feature.
///
/// Mirrors the web client's `messages/page.tsx` + `use-messages.ts` + `use-conversations.ts`:
/// - Fetches conversations (groups + DM partners) and builds a unified list
/// - Cursor-based message loading (newest-first, `before=<id>`, limit 50)
/// - WebSocket subscription for real-time `message_new` events
/// - Optimistic message insertion after send
/// - Multipart form-data message sending with file attachments
@Observable @MainActor
final class MessagesViewModel {

    // MARK: - Conversations

    private(set) var conversations: [Conversation] = []
    private(set) var isLoadingConversations = false

    var activeConversation: Conversation?
    var selectedConversationId: String?

    // MARK: - Messages (for active conversation)

    private(set) var messages: [MessageResponse] = []
    private(set) var isLoadingMessages = false
    private(set) var hasMoreMessages = true

    // MARK: - Send State

    private(set) var isSending = false

    // MARK: - Error

    private(set) var error: String?

    // MARK: - Private

    private let api = APIClient.shared
    private var wsUnsubscribe: (() -> Void)?

    // MARK: - Conversations Loading

    /// Fetch the conversation list (groups + DM partners).
    func loadConversations() async {
        isLoadingConversations = true
        defer { isLoadingConversations = false }

        var result: [Conversation] = []

        // Fetch user's groups
        do {
            let groups: [Group] = try await api.get(Endpoints.usersMeGroups)
            for group in groups {
                result.append(Conversation(
                    id: group.id,
                    type: .group,
                    name: group.name))
            }
        } catch {
            // Silently ignore — groups section empty
        }

        // Fetch DM conversation partners
        do {
            let partners: [DMConversationPartner] = try await api.get(
                Endpoints.messagesConversations)
            for partner in partners {
                result.append(Conversation(
                    id: partner.userId,
                    type: .direct,
                    name: partner.displayName.isEmpty ? partner.username : partner.displayName))
            }
        } catch {
            // Silently ignore — DM section empty
        }

        conversations = result
    }

    /// Add a DM conversation for a user (if not already present). Returns the conversation.
    @discardableResult
    func addDmConversation(userId: String, displayName: String) -> Conversation {
        if let existing = conversations.first(where: { $0.id == userId && $0.type == .direct }) {
            return existing
        }
        let conv = Conversation(id: userId, type: .direct, name: displayName)
        conversations.append(conv)
        return conv
    }

    // MARK: - Messages Loading

    /// Load messages for the active conversation (initial load or cursor-based pagination).
    func loadMessages(before: String? = nil) async {
        guard let conversation = activeConversation else { return }

        isLoadingMessages = true
        defer { isLoadingMessages = false }

        var params: [String: String] = ["limit": "50"]
        if let before { params["before"] = before }

        do {
            let endpoint: String
            switch conversation.type {
            case .group:
                endpoint = Endpoints.groupMessages(conversation.id)
            case .direct:
                endpoint = Endpoints.directMessages(conversation.id)
            }

            let result: [MessageResponse] = try await api.get(endpoint, params: params)

            if before != nil {
                // Append older messages
                messages.append(contentsOf: result)
            } else {
                // Initial load
                messages = result
            }

            hasMoreMessages = result.count == 50
        } catch {
            self.error = "Failed to load messages"
        }
    }

    /// Load older messages (cursor pagination).
    func loadMore() async {
        guard hasMoreMessages, !isLoadingMessages, !messages.isEmpty else { return }
        let oldestId = messages.last?.id
        await loadMessages(before: oldestId)
    }

    /// Select a conversation and load its messages.
    func selectConversation(_ conversation: Conversation) {
        activeConversation = conversation
        messages = []
        hasMoreMessages = true
        Task {
            await loadMessages()
        }
    }

    /// Deselect the active conversation (mobile back).
    func clearActiveConversation() {
        activeConversation = nil
        messages = []
    }

    // MARK: - Send Message

    /// Send a message with optional file attachments.
    /// Uses multipart/form-data (same as web's raw `fetch` approach).
    func sendMessage(
        content: String?,
        files: [URL] = [],
        lat: Double? = nil,
        lng: Double? = nil,
        deviceId: String? = nil
    ) async throws -> MessageResponse {
        guard let conversation = activeConversation else {
            throw APIError(status: 0, message: "No active conversation")
        }

        isSending = true
        defer { isSending = false }

        var form = MultipartFormData()

        if let content, !content.isEmpty {
            form.append(name: "content", value: content)
        }

        switch conversation.type {
        case .group:
            form.append(name: "group_id", value: conversation.id)
        case .direct:
            form.append(name: "recipient_id", value: conversation.id)
        }

        if let lat { form.append(name: "lat", value: String(lat)) }
        if let lng { form.append(name: "lng", value: String(lng)) }
        if let deviceId { form.append(name: "device_id", value: deviceId) }

        for fileURL in files {
            if let data = try? Data(contentsOf: fileURL) {
                let filename = fileURL.lastPathComponent
                let mimeType = mimeTypeForExtension(fileURL.pathExtension)
                form.append(name: "files", data: data, filename: filename, mimeType: mimeType)
            }
        }

        let result: MessageResponse = try await api.upload(
            Endpoints.messages,
            formData: form,
            method: "POST")

        // Optimistic insertion
        messages.insert(result, at: 0)

        return result
    }

    // MARK: - WebSocket

    /// Subscribe to real-time messages.
    func subscribeToMessages(webSocket: WebSocketService, currentUserId: String?) {
        struct MessageEnvelope: Decodable { let payload: MessageResponse }

        wsUnsubscribe = webSocket.subscribe { [weak self] type, data in
            guard type == WSMessageType.messageNew else { return }

            Task { @MainActor [weak self] in
                guard let self else { return }

                do {
                    let envelope = try JSONDecoder.snakeCase.decode(MessageEnvelope.self, from: data)
                    let msg = envelope.payload

                    // Skip own messages (already added optimistically)
                    if msg.senderId == currentUserId { return }

                    // Add to active conversation if it matches
                    if let active = self.activeConversation {
                        switch active.type {
                        case .group:
                            if msg.groupId == active.id {
                                self.messages.insert(msg, at: 0)
                            }
                        case .direct:
                            if msg.senderId == active.id || msg.recipientId == active.id {
                                self.messages.insert(msg, at: 0)
                            }
                        }
                    }

                    // Auto-add DM conversation for new senders
                    if msg.groupId == nil {
                        let name = msg.displayName.isEmpty ? msg.username : msg.displayName
                        self.addDmConversation(userId: msg.senderId, displayName: name)
                    }
                } catch {
                    AppLogger.shared.error(.ws, "message_new decode failed: \(error)")
                }
            }
        }
    }

    func unsubscribe() {
        wsUnsubscribe?()
        wsUnsubscribe = nil
    }

    // MARK: - Helpers

    private func mimeTypeForExtension(_ ext: String) -> String {
        switch ext.lowercased() {
        case "jpg", "jpeg": return "image/jpeg"
        case "png": return "image/png"
        case "gif": return "image/gif"
        case "heic": return "image/heic"
        case "pdf": return "application/pdf"
        case "gpx": return "application/gpx+xml"
        case "mp4": return "video/mp4"
        case "mov": return "video/quicktime"
        default: return "application/octet-stream"
        }
    }
}


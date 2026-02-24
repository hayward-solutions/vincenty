import SwiftUI

/// The main messages screen with conversation list + chat thread.
///
/// Uses `NavigationSplitView` on iPad for sidebar + detail,
/// and a navigation-based push on iPhone.
struct MessagesScreen: View {
    @Environment(AuthManager.self) private var auth
    @Environment(WebSocketService.self) private var webSocket
    @Environment(DeviceManager.self) private var deviceManager
    @Environment(LocationSharingManager.self) private var locationSharing

    @State private var viewModel = MessagesViewModel()
    @State private var showNewDM = false

    var body: some View {
        NavigationSplitView {
            // Sidebar: conversation list
            ConversationsListView(
                conversations: viewModel.conversations,
                activeId: viewModel.activeConversation?.id,
                onSelect: { conv in
                    viewModel.selectConversation(conv)
                },
                onNewMessage: {
                    showNewDM = true
                })
                .navigationTitle("Messages")
        } detail: {
            // Detail: chat thread
            if let conversation = viewModel.activeConversation {
                VStack(spacing: 0) {
                    // Thread header
                    HStack(spacing: 8) {
                        Image(
                            systemName: conversation.type == .group ? "number" : "person")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                        Text(conversation.name)
                            .font(.headline)
                    }
                    .padding()
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(.bar)

                    Divider()

                    // Messages
                    ChatThreadView(
                        messages: viewModel.messages,
                        currentUserId: auth.user?.id ?? "",
                        isLoading: viewModel.isLoadingMessages,
                        hasMore: viewModel.hasMoreMessages,
                        onLoadMore: {
                            Task { await viewModel.loadMore() }
                        })

                    // Input
                    MessageInputView(
                        onSend: { content, files in
                            Task {
                                try? await viewModel.sendMessage(
                                    content: content.isEmpty ? nil : content,
                                    files: files,
                                    lat: locationSharing.currentPosition?.lat,
                                    lng: locationSharing.currentPosition?.lng,
                                    deviceId: deviceManager.deviceId)
                            }
                        },
                        disabled: viewModel.isSending)
                }
            } else {
                ContentUnavailableView(
                    "Select a Conversation",
                    systemImage: "message",
                    description: Text("Choose a conversation from the sidebar to start messaging."))
            }
        }
        .task {
            await viewModel.loadConversations()
        }
        .task {
            viewModel.subscribeToMessages(
                webSocket: webSocket,
                currentUserId: auth.user?.id)
        }
        .onDisappear {
            viewModel.unsubscribe()
        }
        .sheet(isPresented: $showNewDM) {
            NewDirectMessageView { userId, displayName in
                let conv = viewModel.addDmConversation(userId: userId, displayName: displayName)
                viewModel.selectConversation(conv)
            }
        }
    }
}

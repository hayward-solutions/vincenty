import SwiftUI

/// Chat message thread with cursor-based infinite scroll.
/// Mirrors the web client's `message-thread.tsx`.
///
/// Messages come from the API newest-first; we reverse for display
/// (oldest at top, newest at bottom).
struct ChatThreadView: View {
    let messages: [MessageResponse]
    let currentUserId: String
    let isLoading: Bool
    let hasMore: Bool
    let onLoadMore: () -> Void

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(spacing: 12) {
                    // Load more at top
                    if hasMore {
                        if isLoading {
                            ProgressView()
                                .padding(.vertical, 8)
                                .accessibilityLabel("Loading older messages")
                        } else {
                            Button("Load older messages") {
                                onLoadMore()
                            }
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.vertical, 8)
                        }
                    }

                    // Messages (reversed: oldest first)
                    ForEach(displayMessages) { msg in
                        MessageBubbleView(
                            message: msg,
                            isOwn: msg.senderId == currentUserId)
                            .id(msg.id)
                    }

                    // Anchor for scroll-to-bottom
                    Color.clear
                        .frame(height: 1)
                        .id("bottom")
                }
                .padding()
            }
            .onChange(of: messages.first?.id) { _, newId in
                // Scroll to bottom when new messages arrive
                if let newId {
                    withAnimation(.easeOut(duration: 0.3)) {
                        proxy.scrollTo("bottom", anchor: .bottom)
                    }
                }
            }
            .onAppear {
                // Scroll to bottom on initial load
                proxy.scrollTo("bottom", anchor: .bottom)
            }
        }
    }

    /// Messages reversed for display (oldest at top).
    private var displayMessages: [MessageResponse] {
        messages.reversed()
    }
}

import SwiftUI

/// Sidebar / list of conversations (groups + DMs).
/// Mirrors the web client's `conversation-list.tsx`.
struct ConversationsListView: View {
    let conversations: [Conversation]
    let activeId: String?
    let onSelect: (Conversation) -> Void
    let onNewMessage: () -> Void

    var body: some View {
        List {
            // Groups section
            let groups = conversations.filter { $0.type == .group }
            if !groups.isEmpty {
                Section("Groups") {
                    ForEach(groups) { conv in
                        ConversationRow(
                            conversation: conv,
                            isActive: activeId == conv.id,
                            onSelect: onSelect)
                    }
                }
            }

            // DMs section
            let dms = conversations.filter { $0.type == .direct }
            if !dms.isEmpty {
                Section("Direct Messages") {
                    ForEach(dms) { conv in
                        ConversationRow(
                            conversation: conv,
                            isActive: activeId == conv.id,
                            onSelect: onSelect)
                    }
                }
            }

            if conversations.isEmpty {
                ContentUnavailableView(
                    "No Conversations",
                    systemImage: "message",
                    description: Text("Join a group or start a direct message."))
            }
        }
        .listStyle(.sidebar)
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    onNewMessage()
                } label: {
                    Image(systemName: "square.and.pencil")
                }
                .accessibilityLabel("New direct message")
            }
        }
    }
}

// MARK: - Conversation Row

private struct ConversationRow: View {
    let conversation: Conversation
    let isActive: Bool
    let onSelect: (Conversation) -> Void

    var body: some View {
        Button {
            onSelect(conversation)
        } label: {
            HStack(spacing: 8) {
                Image(systemName: conversation.type == .group ? "number" : "person")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .frame(width: 20)

                Text(conversation.name)
                    .font(.subheadline.weight(isActive ? .semibold : .regular))
                    .lineLimit(1)
            }
        }
        .listRowBackground(isActive ? Color.accentColor.opacity(0.1) : nil)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(conversation.type == .group ? "Group" : "Direct message"): \(conversation.name)")
        .accessibilityValue(isActive ? "Selected" : "")
        .accessibilityAddTraits(isActive ? .isSelected : [])
    }
}

"use client";

import { useState, useCallback, useEffect } from "react";
import { useAuth } from "@/lib/auth-context";
import { useWebSocket } from "@/lib/websocket-context";
import { useConversations } from "@/lib/hooks/use-conversations";
import {
  useSendMessage,
  useGroupMessages,
  useDirectMessages,
} from "@/lib/hooks/use-messages";
import { ConversationList } from "@/components/chat/conversation-list";
import { MessageThread } from "@/components/chat/message-thread";
import { MessageInput } from "@/components/chat/message-input";
import { NewDmDialog } from "@/components/chat/new-dm-dialog";
import { Separator } from "@/components/ui/separator";
import type { Conversation, MessageResponse } from "@/types/api";
import { Hash, MessageSquare, User as UserIcon } from "lucide-react";

export default function MessagesPage() {
  const { user } = useAuth();
  const { subscribe } = useWebSocket();
  const {
    conversations,
    isLoading: convLoading,
    addDmConversation,
  } = useConversations();
  const { sendMessage, isLoading: sending } = useSendMessage();

  const [activeConversation, setActiveConversation] =
    useState<Conversation | null>(null);
  const [newDmOpen, setNewDmOpen] = useState(false);

  // Group messages hook
  const groupMessages = useGroupMessages(
    activeConversation?.type === "group" ? activeConversation.id : null
  );

  // Direct messages hook
  const directMessages = useDirectMessages(
    activeConversation?.type === "direct" ? activeConversation.id : null
  );

  // Select the right messages based on conversation type
  const activeMessages =
    activeConversation?.type === "group" ? groupMessages : directMessages;

  // Listen for incoming DMs via WebSocket and auto-add the sender's conversation
  useEffect(() => {
    const unsubscribe = subscribe((type, payload) => {
      if (type !== "message_new" || !user) return;
      const msg = payload as MessageResponse;

      // Only handle DMs (no group_id), from someone else
      if (msg.group_id || msg.sender_id === user.id) return;

      // Add the sender as a DM conversation if not present
      const name = msg.display_name || msg.username;
      addDmConversation(msg.sender_id, name);
    });

    return unsubscribe;
  }, [subscribe, user, addDmConversation]);

  const handleSend = useCallback(
    async (content: string, files: File[]) => {
      if (!activeConversation || !user) return;

      try {
        const result = await sendMessage({
          content: content || undefined,
          groupId:
            activeConversation.type === "group"
              ? activeConversation.id
              : undefined,
          recipientId:
            activeConversation.type === "direct"
              ? activeConversation.id
              : undefined,
          files: files.length > 0 ? files : undefined,
        });

        // Optimistic render — add the message immediately
        activeMessages.addOptimistic(result);
      } catch (err) {
        console.error("Failed to send message:", err);
      }
    },
    [activeConversation, user, sendMessage, activeMessages]
  );

  const handleSelectConversation = useCallback((conv: Conversation) => {
    setActiveConversation(conv);
  }, []);

  // When a user is picked from the New DM dialog
  const handleNewDmSelect = useCallback(
    (userId: string, displayName: string) => {
      const conv = addDmConversation(userId, displayName);
      setActiveConversation(conv);
    },
    [addDmConversation]
  );

  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      {/* Left panel: conversation list */}
      <div className="w-64 shrink-0 border-r flex flex-col">
        <div className="p-3 border-b">
          <h2 className="text-sm font-semibold flex items-center gap-1.5">
            <MessageSquare className="h-4 w-4" />
            Conversations
          </h2>
        </div>
        {convLoading ? (
          <div className="flex items-center justify-center flex-1 text-muted-foreground text-sm">
            Loading...
          </div>
        ) : (
          <ConversationList
            conversations={conversations}
            activeId={activeConversation?.id ?? null}
            onSelect={handleSelectConversation}
            onNewMessage={() => setNewDmOpen(true)}
          />
        )}
      </div>

      <Separator orientation="vertical" />

      {/* Right panel: message thread + input */}
      <div className="flex-1 flex flex-col min-w-0">
        {activeConversation ? (
          <>
            {/* Thread header */}
            <div className="p-3 border-b flex items-center gap-2">
              {activeConversation.type === "group" ? (
                <Hash className="h-4 w-4 text-muted-foreground" />
              ) : (
                <UserIcon className="h-4 w-4 text-muted-foreground" />
              )}
              <h3 className="text-sm font-semibold truncate">
                {activeConversation.name}
              </h3>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-hidden">
              <MessageThread
                messages={activeMessages.messages}
                currentUserId={user?.id ?? ""}
                isLoading={activeMessages.isLoading}
                hasMore={activeMessages.hasMore}
                onLoadMore={activeMessages.loadMore}
              />
            </div>

            {/* Input */}
            <MessageInput onSend={handleSend} disabled={sending} />
          </>
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
            Select a conversation to start messaging
          </div>
        )}
      </div>

      {/* New DM dialog */}
      <NewDmDialog
        open={newDmOpen}
        onOpenChange={setNewDmOpen}
        onSelect={handleNewDmSelect}
      />
    </div>
  );
}

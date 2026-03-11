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
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { useCreateCall } from "@/lib/hooks/use-calls";
import { ConversationList } from "@/components/chat/conversation-list";
import { MessageThread } from "@/components/chat/message-thread";
import { MessageInput } from "@/components/chat/message-input";
import { NewDmDialog } from "@/components/chat/new-dm-dialog";
import { Separator } from "@/components/ui/separator";
import { Button } from "@/components/ui/button";
import type { Conversation, MessageResponse } from "@/types/api";
import { useLocationSharing } from "@/lib/hooks/use-location-sharing";
import {
  ArrowLeft,
  Hash,
  MessageSquare,
  Phone,
  User as UserIcon,
  Video,
} from "lucide-react";

export default function MessagesPage() {
  const { user } = useAuth();
  const { subscribe } = useWebSocket();
  const { lastPosition } = useLocationSharing();
  const {
    conversations,
    isLoading: convLoading,
    addDmConversation,
  } = useConversations();
  const { sendMessage, isLoading: sending } = useSendMessage();
  const { createCall, isLoading: callLoading } = useCreateCall();

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
          lat: lastPosition?.lat,
          lng: lastPosition?.lng,
          files: files.length > 0 ? files : undefined,
        });

        // Optimistic render — add the message immediately
        activeMessages.addOptimistic(result);
      } catch (err) {
        console.error("Failed to send message:", err);
      }
    },
    [activeConversation, user, sendMessage, activeMessages, lastPosition]
  );

  const handleSelectConversation = useCallback((conv: Conversation) => {
    setActiveConversation(conv);
  }, []);

  // Mobile back: clear active conversation to show the list
  const handleBack = useCallback(() => {
    setActiveConversation(null);
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
      {/* Mobile: hidden when a conversation is selected */}
      <div
        className={`w-full md:w-64 shrink-0 border-r flex flex-col ${
          activeConversation ? "hidden md:flex" : "flex"
        }`}
      >
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

      <Separator orientation="vertical" className="hidden md:block" />

      {/* Right panel: message thread + input */}
      {/* Mobile: hidden when no conversation is selected */}
      <div
        className={`flex-1 flex flex-col min-w-0 ${
          activeConversation ? "flex" : "hidden md:flex"
        }`}
      >
        {activeConversation ? (
          <>
            {/* Thread header */}
            <div className="p-3 border-b flex items-center gap-2">
              {/* Mobile back button */}
              <Button
                variant="ghost"
                size="sm"
                className="md:hidden h-8 w-8 p-0 shrink-0"
                onClick={handleBack}
              >
                <ArrowLeft className="h-4 w-4" />
                <span className="sr-only">Back to conversations</span>
              </Button>
              {activeConversation.type === "group" ? (
                <Hash className="h-4 w-4 text-muted-foreground" />
              ) : (
                <UserIcon className="h-4 w-4 text-muted-foreground" />
              )}
              <h3 className="text-sm font-semibold truncate">
                {activeConversation.name}
              </h3>
              {activeConversation.type === "group" && (
                <div className="ml-auto flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 w-8 p-0"
                    disabled={callLoading}
                    onClick={async () => {
                      try {
                        await createCall({
                          group_id: activeConversation.id,
                          name: `${activeConversation.name} call`,
                          video_enabled: false,
                        });
                        toast.success("Call started");
                      } catch (err) {
                        toast.error(
                          err instanceof ApiError
                            ? err.message
                            : "Failed to start call"
                        );
                      }
                    }}
                    title="Start voice call"
                  >
                    <Phone className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 w-8 p-0"
                    disabled={callLoading}
                    onClick={async () => {
                      try {
                        await createCall({
                          group_id: activeConversation.id,
                          name: `${activeConversation.name} call`,
                          video_enabled: true,
                        });
                        toast.success("Video call started");
                      } catch (err) {
                        toast.error(
                          err instanceof ApiError
                            ? err.message
                            : "Failed to start call"
                        );
                      }
                    }}
                    title="Start video call"
                  >
                    <Video className="h-4 w-4" />
                  </Button>
                </div>
              )}
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

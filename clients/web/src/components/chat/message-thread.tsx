"use client";

import { useCallback, useEffect, useMemo, useRef } from "react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { MessageBubble } from "@/components/chat/message-bubble";
import type { MessageResponse } from "@/types/api";
import { Loader2 } from "lucide-react";

interface MessageThreadProps {
  messages: MessageResponse[];
  currentUserId: string;
  isLoading: boolean;
  hasMore: boolean;
  onLoadMore: () => void;
}

export function MessageThread({
  messages,
  currentUserId,
  isLoading,
  hasMore,
  onLoadMore,
}: MessageThreadProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Messages come from the API newest-first, so reverse for display
  // (oldest at top, newest at bottom)
  const displayMessages = useMemo(() => [...messages].reverse(), [messages]);

  // Scroll to bottom when a new message arrives or conversation switches
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Infinite scroll: detect when user scrolls near the top
  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el || isLoading || !hasMore) return;
    if (el.scrollTop < 100) {
      onLoadMore();
    }
  }, [isLoading, hasMore, onLoadMore]);

  if (messages.length === 0 && !isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
        No messages yet. Start the conversation!
      </div>
    );
  }

  return (
    <ScrollArea className="h-full" ref={scrollRef} onScrollCapture={handleScroll}>
      <div className="flex flex-col gap-3 p-4">
        {/* Load more indicator */}
        {hasMore ? (
          <div className="flex justify-center py-2">
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            ) : (
              <button
                type="button"
                onClick={onLoadMore}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Load older messages
              </button>
            )}
          </div>
        ) : null}

        {displayMessages.map((msg) => (
          <MessageBubble
            key={msg.id}
            message={msg}
            isOwn={msg.sender_id === currentUserId}
          />
        ))}

        <div ref={bottomRef} />
      </div>
    </ScrollArea>
  );
}

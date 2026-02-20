"use client";

import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import type { Conversation } from "@/types/api";
import { Hash, PenSquare, User } from "lucide-react";

interface ConversationListProps {
  conversations: Conversation[];
  activeId: string | null;
  onSelect: (conversation: Conversation) => void;
  onNewMessage?: () => void;
}

export function ConversationList({
  conversations,
  activeId,
  onSelect,
  onNewMessage,
}: ConversationListProps) {
  // Separate groups and DMs for display
  const groups = conversations.filter((c) => c.type === "group");
  const dms = conversations.filter((c) => c.type === "direct");

  return (
    <div className="flex flex-col h-full">
      {/* New DM button */}
      {onNewMessage && (
        <div className="px-2 pt-2">
          <Button
            variant="outline"
            size="sm"
            className="w-full justify-start gap-2 text-xs"
            onClick={onNewMessage}
          >
            <PenSquare className="h-3.5 w-3.5" />
            New Message
          </Button>
        </div>
      )}

      <ScrollArea className="flex-1">
        <div className="flex flex-col gap-0.5 p-2">
          {/* Groups section */}
          {groups.length > 0 && (
            <>
              <div className="px-3 pt-2 pb-1 text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Groups
              </div>
              {groups.map((conv) => (
                <ConversationItem
                  key={conv.id}
                  conversation={conv}
                  isActive={activeId === conv.id}
                  onSelect={onSelect}
                />
              ))}
            </>
          )}

          {/* DMs section */}
          {dms.length > 0 && (
            <>
              <div className="px-3 pt-3 pb-1 text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Direct Messages
              </div>
              {dms.map((conv) => (
                <ConversationItem
                  key={conv.id}
                  conversation={conv}
                  isActive={activeId === conv.id}
                  onSelect={onSelect}
                />
              ))}
            </>
          )}

          {conversations.length === 0 && (
            <div className="flex items-center justify-center py-8 text-muted-foreground text-sm">
              No conversations yet
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}

function ConversationItem({
  conversation,
  isActive,
  onSelect,
}: {
  conversation: Conversation;
  isActive: boolean;
  onSelect: (conversation: Conversation) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(conversation)}
      className={cn(
        "flex items-center gap-2 rounded-md px-3 py-2 text-sm text-left transition-colors",
        "hover:bg-accent hover:text-accent-foreground",
        isActive && "bg-accent text-accent-foreground"
      )}
    >
      {conversation.type === "group" ? (
        <Hash className="h-4 w-4 shrink-0 text-muted-foreground" />
      ) : (
        <User className="h-4 w-4 shrink-0 text-muted-foreground" />
      )}
      <span className="truncate font-medium">{conversation.name}</span>
    </button>
  );
}

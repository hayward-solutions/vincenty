"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { MessageSquare, Users, ArrowRight } from "lucide-react";
import { useWebSocket } from "@/lib/websocket-context";
import { useConversations } from "@/lib/hooks/use-conversations";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { MessageResponse } from "@/types/api";

/** Format a timestamp as a short relative string without external deps. */
function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60) return "just now";
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

/** Truncate text for preview. */
function truncate(text: string, max = 60): string {
  if (!text) return "";
  return text.length > max ? text.slice(0, max) + "…" : text;
}

export function RecentMessages() {
  const { conversations, isLoading } = useConversations();
  const { subscribe } = useWebSocket();

  // Track the most-recent message per conversation (keyed by conversation id)
  const [latestMessages, setLatestMessages] = useState<
    Map<string, MessageResponse>
  >(new Map());

  useEffect(() => {
    const unsubscribe = subscribe((type, payload) => {
      if (type === "message_new") {
        const msg = payload as MessageResponse;
        const convId = msg.group_id ?? msg.recipient_id ?? msg.sender_id;
        if (!convId) return;
        setLatestMessages((prev) => {
          const next = new Map(prev);
          const existing = next.get(convId);
          if (
            !existing ||
            new Date(msg.created_at) > new Date(existing.created_at)
          ) {
            next.set(convId, msg);
          }
          return next;
        });
      }
    });
    return unsubscribe;
  }, [subscribe]);

  const LIMIT = 5;
  const shown = conversations.slice(0, LIMIT);

  return (
    <Card className="flex flex-col">
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="text-base font-semibold">
          Recent Messages
        </CardTitle>
        <Button asChild variant="ghost" size="sm" className="text-xs gap-1">
          <Link href="/messages">
            View all
            <ArrowRight className="h-3 w-3" />
          </Link>
        </Button>
      </CardHeader>
      <CardContent className="flex-1 p-0">
        {isLoading ? (
          <ul className="divide-y">
            {Array.from({ length: 4 }).map((_, i) => (
              <li key={i} className="flex items-start gap-3 px-6 py-3">
                <Skeleton className="h-8 w-8 rounded-full shrink-0 mt-0.5" />
                <div className="flex-1 min-w-0 space-y-1">
                  <Skeleton className="h-3.5 w-32" />
                  <Skeleton className="h-3 w-48" />
                </div>
              </li>
            ))}
          </ul>
        ) : conversations.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-10 text-center px-6">
            <MessageSquare className="h-8 w-8 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">No conversations yet.</p>
            <Button asChild variant="link" size="sm" className="mt-1">
              <Link href="/messages">Start a conversation</Link>
            </Button>
          </div>
        ) : (
          <ul className="divide-y">
            {shown.map((conv) => {
              const lastMsg = latestMessages.get(conv.id);
              const href =
                conv.type === "group"
                  ? `/messages?group=${conv.id}`
                  : `/messages?dm=${conv.id}`;

              return (
                <li key={conv.id}>
                  <Link
                    href={href}
                    className={cn(
                      "flex items-start gap-3 px-6 py-3 hover:bg-muted/50 transition-colors"
                    )}
                  >
                    <div className="mt-0.5 shrink-0 flex h-8 w-8 items-center justify-center rounded-full bg-muted">
                      {conv.type === "group" ? (
                        <Users className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <MessageSquare className="h-4 w-4 text-muted-foreground" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-sm font-medium truncate">
                          {conv.name}
                        </span>
                        <div className="flex items-center gap-1.5 shrink-0">
                          <Badge
                            variant="outline"
                            className="text-[10px] px-1.5 py-0"
                          >
                            {conv.type === "group" ? "Group" : "DM"}
                          </Badge>
                          {lastMsg && (
                            <span className="text-[11px] text-muted-foreground whitespace-nowrap">
                              {relativeTime(lastMsg.created_at)}
                            </span>
                          )}
                        </div>
                      </div>
                      {lastMsg ? (
                        <p className="text-xs text-muted-foreground truncate mt-0.5">
                          <span className="font-medium text-foreground/70">
                            {lastMsg.display_name || lastMsg.username}:
                          </span>{" "}
                          {lastMsg.attachments.length > 0 && !lastMsg.content
                            ? `${lastMsg.attachments.length} attachment${lastMsg.attachments.length > 1 ? "s" : ""}`
                            : truncate(lastMsg.content)}
                        </p>
                      ) : (
                        <p className="text-xs text-muted-foreground mt-0.5">
                          Tap to open conversation
                        </p>
                      )}
                    </div>
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

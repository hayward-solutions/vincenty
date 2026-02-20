"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { Conversation, DMConversationPartner, Group } from "@/types/api";

/**
 * useConversations builds a conversation list from the user's groups
 * and existing + dynamically added DM conversations.
 */
export function useConversations() {
  const [groupConversations, setGroupConversations] = useState<Conversation[]>(
    []
  );
  const [dmConversations, setDmConversations] = useState<Conversation[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchConversations = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      // Fetch groups and existing DM partners in parallel
      const [groups, dmPartners] = await Promise.all([
        api.get<Group[]>("/api/v1/users/me/groups"),
        api.get<DMConversationPartner[]>("/api/v1/messages/conversations"),
      ]);

      setGroupConversations(
        groups.map((g) => ({
          id: g.id,
          type: "group" as const,
          name: g.name,
        }))
      );

      setDmConversations(
        dmPartners.map((p) => ({
          id: p.user_id,
          type: "direct" as const,
          name: p.display_name || p.username,
        }))
      );
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch conversations"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConversations();
  }, [fetchConversations]);

  // Add a DM conversation to the sidebar if it doesn't already exist.
  // Returns the conversation (existing or newly created).
  const addDmConversation = useCallback(
    (userId: string, name: string): Conversation => {
      const existing = dmConversations.find((c) => c.id === userId);
      if (existing) return existing;

      const conv: Conversation = { id: userId, type: "direct", name };
      setDmConversations((prev) => [conv, ...prev]);
      return conv;
    },
    [dmConversations]
  );

  // Groups first, then DMs
  const conversations = [...groupConversations, ...dmConversations];

  return {
    conversations,
    isLoading,
    error,
    refetch: fetchConversations,
    addDmConversation,
  };
}

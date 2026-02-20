"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useWebSocket } from "@/lib/websocket-context";
import { useAuth } from "@/lib/auth-context";
import type { MessageResponse } from "@/types/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

// ---------------------------------------------------------------------------
// Send a message (multipart/form-data via raw fetch)
// ---------------------------------------------------------------------------

interface SendMessageParams {
  content?: string;
  groupId?: string;
  recipientId?: string;
  lat?: number;
  lng?: number;
  files?: File[];
}

export function useSendMessage() {
  const [isLoading, setIsLoading] = useState(false);

  const sendMessage = async (
    params: SendMessageParams
  ): Promise<MessageResponse> => {
    setIsLoading(true);
    try {
      const formData = new FormData();
      if (params.content) formData.append("content", params.content);
      if (params.groupId) formData.append("group_id", params.groupId);
      if (params.recipientId)
        formData.append("recipient_id", params.recipientId);
      if (params.lat != null) formData.append("lat", String(params.lat));
      if (params.lng != null) formData.append("lng", String(params.lng));

      const deviceId = localStorage.getItem("device_id");
      if (deviceId) formData.append("device_id", deviceId);

      if (params.files) {
        for (const file of params.files) {
          formData.append("files", file);
        }
      }

      const token = localStorage.getItem("access_token");
      const res = await fetch(`${API_BASE}/api/v1/messages`, {
        method: "POST",
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: formData,
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({
          error: { message: res.statusText },
        }));
        throw new ApiError(res.status, body?.error?.message || res.statusText);
      }

      return await res.json();
    } finally {
      setIsLoading(false);
    }
  };

  return { sendMessage, isLoading };
}

// ---------------------------------------------------------------------------
// List group messages (cursor-based)
// ---------------------------------------------------------------------------

export function useGroupMessages(groupId: string | null) {
  const [messages, setMessages] = useState<MessageResponse[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { user } = useAuth();
  const { subscribe } = useWebSocket();

  const fetchMessages = useCallback(
    async (before?: string) => {
      if (!groupId) return;
      setIsLoading(true);
      setError(null);
      try {
        const params: Record<string, string> = { limit: "50" };
        if (before) params.before = before;

        const result = await api.get<MessageResponse[]>(
          `/api/v1/groups/${groupId}/messages`,
          { params }
        );
        if (before) {
          // Loading older messages — append
          setMessages((prev) => [...prev, ...result]);
        } else {
          // Initial load
          setMessages(result);
        }
        setHasMore(result.length === 50);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to fetch messages"
        );
      } finally {
        setIsLoading(false);
      }
    },
    [groupId]
  );

  // Initial load when groupId changes
  useEffect(() => {
    if (groupId) {
      setMessages([]);
      setHasMore(true);
      fetchMessages();
    }
  }, [groupId, fetchMessages]);

  // Subscribe to real-time messages via WS
  useEffect(() => {
    if (!groupId) return;

    const unsubscribe = subscribe((type, payload) => {
      if (type === "message_new") {
        const msg = payload as MessageResponse;
        // Only add messages for this group, from other users (sender already renders optimistically)
        if (msg.group_id === groupId && msg.sender_id !== user?.id) {
          setMessages((prev) => [msg, ...prev]);
        }
      }
    });

    return unsubscribe;
  }, [groupId, subscribe, user?.id]);

  const loadMore = useCallback(() => {
    if (messages.length > 0 && hasMore && !isLoading) {
      const oldest = messages[messages.length - 1];
      fetchMessages(oldest.id);
    }
  }, [messages, hasMore, isLoading, fetchMessages]);

  // Add an optimistic message from the sender
  const addOptimistic = useCallback((msg: MessageResponse) => {
    setMessages((prev) => [msg, ...prev]);
  }, []);

  return { messages, isLoading, hasMore, error, loadMore, addOptimistic, refetch: fetchMessages };
}

// ---------------------------------------------------------------------------
// List direct messages (cursor-based)
// ---------------------------------------------------------------------------

export function useDirectMessages(otherUserId: string | null) {
  const [messages, setMessages] = useState<MessageResponse[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { user } = useAuth();
  const { subscribe } = useWebSocket();

  const fetchMessages = useCallback(
    async (before?: string) => {
      if (!otherUserId) return;
      setIsLoading(true);
      setError(null);
      try {
        const params: Record<string, string> = { limit: "50" };
        if (before) params.before = before;

        const result = await api.get<MessageResponse[]>(
          `/api/v1/messages/direct/${otherUserId}`,
          { params }
        );
        if (before) {
          setMessages((prev) => [...prev, ...result]);
        } else {
          setMessages(result);
        }
        setHasMore(result.length === 50);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to fetch messages"
        );
      } finally {
        setIsLoading(false);
      }
    },
    [otherUserId]
  );

  useEffect(() => {
    if (otherUserId) {
      setMessages([]);
      setHasMore(true);
      fetchMessages();
    }
  }, [otherUserId, fetchMessages]);

  // Subscribe to real-time DMs via WS
  useEffect(() => {
    if (!otherUserId) return;

    const unsubscribe = subscribe((type, payload) => {
      if (type === "message_new") {
        const msg = payload as MessageResponse;
        // DM from this conversation partner
        if (
          !msg.group_id &&
          (msg.sender_id === otherUserId ||
            msg.recipient_id === otherUserId) &&
          msg.sender_id !== user?.id
        ) {
          setMessages((prev) => [msg, ...prev]);
        }
      }
    });

    return unsubscribe;
  }, [otherUserId, subscribe, user?.id]);

  const loadMore = useCallback(() => {
    if (messages.length > 0 && hasMore && !isLoading) {
      const oldest = messages[messages.length - 1];
      fetchMessages(oldest.id);
    }
  }, [messages, hasMore, isLoading, fetchMessages]);

  const addOptimistic = useCallback((msg: MessageResponse) => {
    setMessages((prev) => [msg, ...prev]);
  }, []);

  return { messages, isLoading, hasMore, error, loadMore, addOptimistic, refetch: fetchMessages };
}

// ---------------------------------------------------------------------------
// Delete message
// ---------------------------------------------------------------------------

export function useDeleteMessage() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteMessage = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/messages/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteMessage, isLoading };
}

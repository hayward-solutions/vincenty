"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useWebSocket } from "@/lib/websocket-context";
import type {
  CreatePTTChannelRequest,
  JoinPTTChannelResponse,
  PTTChannel,
  WSPTTFloorEvent,
} from "@/types/api";

export function usePTTChannels(groupId: string) {
  const [channels, setChannels] = useState<PTTChannel[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchChannels = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<PTTChannel[]>(
        `/api/v1/groups/${groupId}/ptt-channels`
      );
      setChannels(result);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch PTT channels"
      );
    } finally {
      setIsLoading(false);
    }
  }, [groupId]);

  useEffect(() => {
    fetchChannels();
  }, [fetchChannels]);

  return { channels, isLoading, error, refetch: fetchChannels };
}

export function useCreatePTTChannel() {
  const [isLoading, setIsLoading] = useState(false);

  const createChannel = async (
    groupId: string,
    req: CreatePTTChannelRequest
  ): Promise<PTTChannel> => {
    setIsLoading(true);
    try {
      return await api.post<PTTChannel>(
        `/api/v1/groups/${groupId}/ptt-channels`,
        req
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { createChannel, isLoading };
}

export function useJoinPTTChannel() {
  const [isLoading, setIsLoading] = useState(false);

  const joinChannel = async (
    groupId: string,
    channelId: string
  ): Promise<JoinPTTChannelResponse> => {
    setIsLoading(true);
    try {
      return await api.post<JoinPTTChannelResponse>(
        `/api/v1/groups/${groupId}/ptt-channels/${channelId}/join`
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { joinChannel, isLoading };
}

export function useDeletePTTChannel() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteChannel = async (
    groupId: string,
    channelId: string
  ): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(
        `/api/v1/groups/${groupId}/ptt-channels/${channelId}`
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteChannel, isLoading };
}

/** Hook that listens for PTT floor events via WebSocket. */
export function usePTTFloor(channelId: string) {
  const { subscribe, sendMessage } = useWebSocket();
  const [floorHolder, setFloorHolder] = useState<{
    id: string;
    name: string;
  } | null>(null);

  useEffect(() => {
    const unsubscribe = subscribe((type, payload) => {
      if (type === "ptt_floor_granted" || type === "ptt_floor_released") {
        const evt = payload as WSPTTFloorEvent;
        if (evt.channel_id !== channelId) return;

        if (evt.event_type === "floor_granted" && evt.holder_id) {
          setFloorHolder({
            id: evt.holder_id,
            name: evt.holder_name || "Unknown",
          });
        } else {
          setFloorHolder(null);
        }
      }
    });

    return unsubscribe;
  }, [channelId, subscribe]);

  const requestFloor = useCallback(() => {
    sendMessage("ptt_floor_request", { channel_id: channelId });
  }, [channelId, sendMessage]);

  const releaseFloor = useCallback(() => {
    sendMessage("ptt_floor_release", { channel_id: channelId });
  }, [channelId, sendMessage]);

  return { floorHolder, requestFloor, releaseFloor };
}

"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type {
  CreateStreamRequest,
  JoinRoomResponse,
  Stream,
  StreamStartResponse,
} from "@/types/api";

export function useActiveStreams() {
  const [streams, setStreams] = useState<Stream[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStreams = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<Stream[]>("/api/v1/streams");
      setStreams(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch streams");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStreams();
  }, [fetchStreams]);

  return { streams, isLoading, error, refetch: fetchStreams };
}

export function useGroupStreams(groupId: string) {
  const [streams, setStreams] = useState<Stream[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStreams = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<Stream[]>(
        `/api/v1/groups/${groupId}/streams`
      );
      setStreams(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch streams");
    } finally {
      setIsLoading(false);
    }
  }, [groupId]);

  useEffect(() => {
    fetchStreams();
  }, [fetchStreams]);

  return { streams, isLoading, error, refetch: fetchStreams };
}

export function useCreateStream() {
  const [isLoading, setIsLoading] = useState(false);

  const createStream = async (req: CreateStreamRequest): Promise<Stream> => {
    setIsLoading(true);
    try {
      return await api.post<Stream>("/api/v1/streams", req);
    } finally {
      setIsLoading(false);
    }
  };

  return { createStream, isLoading };
}

export function useStartStream() {
  const [isLoading, setIsLoading] = useState(false);

  const startStream = async (
    streamId: string
  ): Promise<StreamStartResponse> => {
    setIsLoading(true);
    try {
      return await api.post<StreamStartResponse>(
        `/api/v1/streams/${streamId}/start`
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { startStream, isLoading };
}

export function useStopStream() {
  const [isLoading, setIsLoading] = useState(false);

  const stopStream = async (streamId: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.post(`/api/v1/streams/${streamId}/stop`);
    } finally {
      setIsLoading(false);
    }
  };

  return { stopStream, isLoading };
}

export function useViewStream() {
  const [isLoading, setIsLoading] = useState(false);

  const viewStream = async (streamId: string): Promise<JoinRoomResponse> => {
    setIsLoading(true);
    try {
      return await api.get<JoinRoomResponse>(`/api/v1/streams/${streamId}/view`);
    } finally {
      setIsLoading(false);
    }
  };

  return { viewStream, isLoading };
}

export function useDeleteStream() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteStream = async (streamId: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/streams/${streamId}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteStream, isLoading };
}

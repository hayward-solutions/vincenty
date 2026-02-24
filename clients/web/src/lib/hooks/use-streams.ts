"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useWebSocket } from "@/lib/websocket-context";
import type {
  StreamResponse,
  StreamLocationResponse,
  CreateStreamRequest,
  ShareStreamRequest,
  WSStreamStarted,
  WSStreamEnded,
  WSStreamLocationBroadcast,
} from "@/types/api";

// ---------------------------------------------------------------------------
// List streams (with optional status filter + real-time WS updates)
// ---------------------------------------------------------------------------

export function useStreams(status?: string) {
  const [streams, setStreams] = useState<StreamResponse[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { subscribe } = useWebSocket();

  const fetchStreams = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const params: Record<string, string> = {};
      if (status) params.status = status;
      const result = await api.get<StreamResponse[]>("/api/v1/streams", {
        params,
      });
      setStreams(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch streams"
      );
    } finally {
      setIsLoading(false);
    }
  }, [status]);

  useEffect(() => {
    fetchStreams();
  }, [fetchStreams]);

  // Subscribe to real-time stream events
  useEffect(() => {
    const unsubscribe = subscribe((type, payload) => {
      if (type === "stream_started") {
        const stream = payload as WSStreamStarted;
        // Only add if matches the current filter (or no filter)
        if (!status || stream.status === status) {
          setStreams((prev) => {
            const exists = prev.some((s) => s.id === stream.id);
            if (exists) {
              return prev.map((s) => (s.id === stream.id ? stream : s));
            }
            return [stream, ...prev];
          });
        }
      }

      if (type === "stream_ended") {
        const { stream_id } = payload as WSStreamEnded;
        if (status === "live") {
          // Remove from live list
          setStreams((prev) => prev.filter((s) => s.id !== stream_id));
        } else {
          // Update status in list
          setStreams((prev) =>
            prev.map((s) =>
              s.id === stream_id ? { ...s, status: "ended" } : s
            )
          );
        }
      }

      if (type === "stream_location_broadcast") {
        // No list-level update needed — handled by stream markers
      }
    });

    return unsubscribe;
  }, [subscribe, status]);

  return { streams, isLoading, error, refetch: fetchStreams, setStreams };
}

// ---------------------------------------------------------------------------
// Get single stream
// ---------------------------------------------------------------------------

export function useStream(streamId: string | null) {
  const [stream, setStream] = useState<StreamResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { subscribe } = useWebSocket();

  const fetchStream = useCallback(async () => {
    if (!streamId) {
      setStream(null);
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<StreamResponse>(
        `/api/v1/streams/${streamId}`
      );
      setStream(result);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch stream"
      );
    } finally {
      setIsLoading(false);
    }
  }, [streamId]);

  useEffect(() => {
    fetchStream();
  }, [fetchStream]);

  // Subscribe to real-time updates for this specific stream
  useEffect(() => {
    if (!streamId) return;

    const unsubscribe = subscribe((type, payload) => {
      if (type === "stream_started") {
        const updated = payload as WSStreamStarted;
        if (updated.id === streamId) {
          setStream(updated);
        }
      }
      if (type === "stream_ended") {
        const { stream_id } = payload as WSStreamEnded;
        if (stream_id === streamId) {
          setStream((prev) =>
            prev ? { ...prev, status: "ended" } : prev
          );
        }
      }
    });

    return unsubscribe;
  }, [streamId, subscribe]);

  return { stream, isLoading, error, refetch: fetchStream };
}

// ---------------------------------------------------------------------------
// Get stream locations (GPS telemetry)
// ---------------------------------------------------------------------------

export function useStreamLocations(streamId: string | null) {
  const [locations, setLocations] = useState<StreamLocationResponse[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchLocations = useCallback(async () => {
    if (!streamId) {
      setLocations([]);
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<StreamLocationResponse[]>(
        `/api/v1/streams/${streamId}/locations`
      );
      setLocations(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch stream locations"
      );
    } finally {
      setIsLoading(false);
    }
  }, [streamId]);

  useEffect(() => {
    fetchLocations();
  }, [fetchLocations]);

  return { locations, isLoading, error, refetch: fetchLocations };
}

// ---------------------------------------------------------------------------
// Create stream
// ---------------------------------------------------------------------------

export function useCreateStream() {
  const [isLoading, setIsLoading] = useState(false);

  const createStream = async (
    req: CreateStreamRequest
  ): Promise<StreamResponse> => {
    setIsLoading(true);
    try {
      return await api.post<StreamResponse>("/api/v1/streams", req);
    } finally {
      setIsLoading(false);
    }
  };

  return { createStream, isLoading };
}

// ---------------------------------------------------------------------------
// End stream
// ---------------------------------------------------------------------------

export function useEndStream() {
  const [isLoading, setIsLoading] = useState(false);

  const endStream = async (streamId: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.post(`/api/v1/streams/${streamId}/end`);
    } finally {
      setIsLoading(false);
    }
  };

  return { endStream, isLoading };
}

// ---------------------------------------------------------------------------
// Share stream
// ---------------------------------------------------------------------------

export function useShareStream() {
  const [isLoading, setIsLoading] = useState(false);

  const shareStream = async (
    streamId: string,
    req: ShareStreamRequest
  ): Promise<void> => {
    setIsLoading(true);
    try {
      await api.post(`/api/v1/streams/${streamId}/share`, req);
    } finally {
      setIsLoading(false);
    }
  };

  return { shareStream, isLoading };
}

// ---------------------------------------------------------------------------
// Delete stream
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Live stream locations via WS (for map markers)
// ---------------------------------------------------------------------------

export interface LiveStreamLocation {
  stream_id: string;
  lat: number;
  lng: number;
  altitude?: number;
  heading?: number;
}

export function useLiveStreamLocations() {
  const [locations, setLocations] = useState<Map<string, LiveStreamLocation>>(
    new Map()
  );
  const { subscribe } = useWebSocket();

  useEffect(() => {
    const unsubscribe = subscribe((type, payload) => {
      if (type === "stream_location_broadcast") {
        const loc = payload as WSStreamLocationBroadcast;
        setLocations((prev) => {
          const next = new Map(prev);
          next.set(loc.stream_id, {
            stream_id: loc.stream_id,
            lat: loc.lat,
            lng: loc.lng,
            altitude: loc.altitude,
            heading: loc.heading,
          });
          return next;
        });
      }

      if (type === "stream_ended") {
        const { stream_id } = payload as WSStreamEnded;
        setLocations((prev) => {
          if (!prev.has(stream_id)) return prev;
          const next = new Map(prev);
          next.delete(stream_id);
          return next;
        });
      }
    });

    return unsubscribe;
  }, [subscribe]);

  return { locations };
}

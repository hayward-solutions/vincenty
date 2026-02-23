"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useWebSocket } from "@/lib/websocket-context";
import { useAuth } from "@/lib/auth-context";
import type {
  DrawingResponse,
  DrawingShareInfo,
  CreateDrawingRequest,
  UpdateDrawingRequest,
  ShareDrawingRequest,
  MessageResponse,
} from "@/types/api";

// ---------------------------------------------------------------------------
// List own drawings
// ---------------------------------------------------------------------------

export function useOwnDrawings() {
  const [drawings, setDrawings] = useState<DrawingResponse[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<DrawingResponse[]>("/api/v1/drawings");
      setDrawings(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch drawings"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetch();
  }, [fetch]);

  return { drawings, isLoading, error, refetch: fetch, setDrawings };
}

// ---------------------------------------------------------------------------
// List shared drawings
// ---------------------------------------------------------------------------

export function useSharedDrawings() {
  const [drawings, setDrawings] = useState<DrawingResponse[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<DrawingResponse[]>(
        "/api/v1/drawings/shared"
      );
      setDrawings(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch shared drawings"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetch();
  }, [fetch]);

  return { drawings, isLoading, error, refetch: fetch, setDrawings };
}

// ---------------------------------------------------------------------------
// Combined: all drawings (own + shared) with WS live updates
// ---------------------------------------------------------------------------

export function useDrawings() {
  const {
    drawings: ownDrawings,
    isLoading: ownLoading,
    refetch: refetchOwn,
    setDrawings: setOwnDrawings,
  } = useOwnDrawings();
  const {
    drawings: sharedDrawings,
    isLoading: sharedLoading,
    refetch: refetchShared,
    setDrawings: setSharedDrawings,
  } = useSharedDrawings();
  const { user } = useAuth();
  const { subscribe } = useWebSocket();

  // Subscribe to drawing_updated WS events
  useEffect(() => {
    const unsubscribe = subscribe((type, payload) => {
      if (type === "drawing_updated") {
        const updated = payload as DrawingResponse;
        // Update in the appropriate list
        if (updated.owner_id === user?.id) {
          setOwnDrawings((prev) =>
            prev.map((d) => (d.id === updated.id ? updated : d))
          );
        } else {
          setSharedDrawings((prev) => {
            const exists = prev.some((d) => d.id === updated.id);
            if (exists) {
              return prev.map((d) => (d.id === updated.id ? updated : d));
            }
            // New shared drawing — add it
            return [updated, ...prev];
          });
        }
      }

      // When a share message arrives via message_new, refetch shared drawings
      if (type === "message_new") {
        const msg = payload as MessageResponse;
        if (
          msg.message_type === "drawing" &&
          msg.sender_id !== user?.id
        ) {
          refetchShared();
        }
      }
    });

    return unsubscribe;
  }, [subscribe, user?.id, setOwnDrawings, setSharedDrawings, refetchShared]);

  const refetch = useCallback(() => {
    refetchOwn();
    refetchShared();
  }, [refetchOwn, refetchShared]);

  return {
    ownDrawings,
    sharedDrawings,
    isLoading: ownLoading || sharedLoading,
    refetch,
  };
}

// ---------------------------------------------------------------------------
// Create drawing
// ---------------------------------------------------------------------------

export function useCreateDrawing() {
  const [isLoading, setIsLoading] = useState(false);

  const createDrawing = async (
    req: CreateDrawingRequest
  ): Promise<DrawingResponse> => {
    setIsLoading(true);
    try {
      return await api.post<DrawingResponse>("/api/v1/drawings", req);
    } finally {
      setIsLoading(false);
    }
  };

  return { createDrawing, isLoading };
}

// ---------------------------------------------------------------------------
// Update drawing
// ---------------------------------------------------------------------------

export function useUpdateDrawing() {
  const [isLoading, setIsLoading] = useState(false);

  const updateDrawing = async (
    id: string,
    req: UpdateDrawingRequest
  ): Promise<DrawingResponse> => {
    setIsLoading(true);
    try {
      return await api.put<DrawingResponse>(`/api/v1/drawings/${id}`, req);
    } finally {
      setIsLoading(false);
    }
  };

  return { updateDrawing, isLoading };
}

// ---------------------------------------------------------------------------
// Delete drawing
// ---------------------------------------------------------------------------

export function useDeleteDrawing() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteDrawing = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/drawings/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteDrawing, isLoading };
}

// ---------------------------------------------------------------------------
// Share drawing
// ---------------------------------------------------------------------------

export function useShareDrawing() {
  const [isLoading, setIsLoading] = useState(false);

  const shareDrawing = async (
    drawingId: string,
    req: ShareDrawingRequest
  ): Promise<MessageResponse> => {
    setIsLoading(true);
    try {
      return await api.post<MessageResponse>(
        `/api/v1/drawings/${drawingId}/share`,
        req
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { shareDrawing, isLoading };
}

// ---------------------------------------------------------------------------
// List shares for a drawing
// ---------------------------------------------------------------------------

export function useDrawingShares(drawingId: string | null) {
  const [shares, setShares] = useState<DrawingShareInfo[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetch = useCallback(async () => {
    if (!drawingId) {
      setShares([]);
      return;
    }
    setIsLoading(true);
    try {
      const result = await api.get<DrawingShareInfo[]>(
        `/api/v1/drawings/${drawingId}/shares`
      );
      setShares(result ?? []);
    } catch {
      setShares([]);
    } finally {
      setIsLoading(false);
    }
  }, [drawingId]);

  useEffect(() => {
    fetch();
  }, [fetch]);

  return { shares, isLoading, refetch: fetch };
}

// ---------------------------------------------------------------------------
// Unshare drawing
// ---------------------------------------------------------------------------

export function useUnshareDrawing() {
  const [isLoading, setIsLoading] = useState(false);

  const unshareDrawing = async (
    drawingId: string,
    messageId: string
  ): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(
        `/api/v1/drawings/${drawingId}/shares/${messageId}`
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { unshareDrawing, isLoading };
}

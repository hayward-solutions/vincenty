"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type {
  StreamKeyResponse,
  CreateStreamKeyRequest,
  UpdateStreamKeyRequest,
} from "@/types/api";

// ---------------------------------------------------------------------------
// List stream keys (admin)
// ---------------------------------------------------------------------------

export function useStreamKeys() {
  const [keys, setKeys] = useState<StreamKeyResponse[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchKeys = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<StreamKeyResponse[]>(
        "/api/v1/admin/stream-keys"
      );
      setKeys(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch stream keys"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchKeys();
  }, [fetchKeys]);

  return { keys, isLoading, error, refetch: fetchKeys };
}

// ---------------------------------------------------------------------------
// Create stream key
// ---------------------------------------------------------------------------

export function useCreateStreamKey() {
  const [isLoading, setIsLoading] = useState(false);

  const createStreamKey = async (
    req: CreateStreamKeyRequest
  ): Promise<StreamKeyResponse> => {
    setIsLoading(true);
    try {
      return await api.post<StreamKeyResponse>(
        "/api/v1/admin/stream-keys",
        req
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { createStreamKey, isLoading };
}

// ---------------------------------------------------------------------------
// Update stream key
// ---------------------------------------------------------------------------

export function useUpdateStreamKey() {
  const [isLoading, setIsLoading] = useState(false);

  const updateStreamKey = async (
    id: string,
    req: UpdateStreamKeyRequest
  ): Promise<StreamKeyResponse> => {
    setIsLoading(true);
    try {
      return await api.put<StreamKeyResponse>(
        `/api/v1/admin/stream-keys/${id}`,
        req
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { updateStreamKey, isLoading };
}

// ---------------------------------------------------------------------------
// Delete stream key
// ---------------------------------------------------------------------------

export function useDeleteStreamKey() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteStreamKey = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/admin/stream-keys/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteStreamKey, isLoading };
}

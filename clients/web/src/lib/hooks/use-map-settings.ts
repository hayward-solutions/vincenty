"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type {
  MapSettings,
  MapConfigResponse,
  CreateMapConfigRequest,
  UpdateMapConfigRequest,
} from "@/types/api";

export function useMapSettings() {
  const [settings, setSettings] = useState<MapSettings | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSettings = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<MapSettings>("/api/v1/map/settings");
      setSettings(result);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch map settings"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  return { settings, isLoading, error, refetch: fetchSettings };
}

// ---------------------------------------------------------------------------
// Admin CRUD hooks
// ---------------------------------------------------------------------------

export function useMapConfigs() {
  const [configs, setConfigs] = useState<MapConfigResponse[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchConfigs = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<MapConfigResponse[]>("/api/v1/map-configs");
      setConfigs(result);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch map configs"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfigs();
  }, [fetchConfigs]);

  return { configs, isLoading, error, refetch: fetchConfigs };
}

export function useCreateMapConfig() {
  const [isLoading, setIsLoading] = useState(false);

  const createMapConfig = async (
    req: CreateMapConfigRequest
  ): Promise<MapConfigResponse> => {
    setIsLoading(true);
    try {
      return await api.post<MapConfigResponse>("/api/v1/map-configs", req);
    } finally {
      setIsLoading(false);
    }
  };

  return { createMapConfig, isLoading };
}

export function useUpdateMapConfig() {
  const [isLoading, setIsLoading] = useState(false);

  const updateMapConfig = async (
    id: string,
    req: UpdateMapConfigRequest
  ): Promise<MapConfigResponse> => {
    setIsLoading(true);
    try {
      return await api.put<MapConfigResponse>(`/api/v1/map-configs/${id}`, req);
    } finally {
      setIsLoading(false);
    }
  };

  return { updateMapConfig, isLoading };
}

export function useDeleteMapConfig() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteMapConfig = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/map-configs/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteMapConfig, isLoading };
}

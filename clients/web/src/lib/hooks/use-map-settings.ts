"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type {
  MapSettings,
  MapConfigResponse,
  MapDefaultsResponse,
  CreateMapConfigRequest,
  UpdateMapConfigRequest,
  TerrainConfigResponse,
  TerrainDefaultsResponse,
  CreateTerrainConfigRequest,
  UpdateTerrainConfigRequest,
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
// Admin hooks
// ---------------------------------------------------------------------------

export function useMapDefaults() {
  const [defaults, setDefaults] = useState<MapDefaultsResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDefaults = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<MapDefaultsResponse>(
        "/api/v1/map-configs/defaults"
      );
      setDefaults(result);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch map defaults"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDefaults();
  }, [fetchDefaults]);

  return { defaults, isLoading, error, refetch: fetchDefaults };
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

// ---------------------------------------------------------------------------
// Terrain config hooks
// ---------------------------------------------------------------------------

export function useTerrainDefaults() {
  const [defaults, setDefaults] = useState<TerrainDefaultsResponse | null>(
    null
  );
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDefaults = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<TerrainDefaultsResponse>(
        "/api/v1/terrain-configs/defaults"
      );
      setDefaults(result);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to fetch terrain defaults"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDefaults();
  }, [fetchDefaults]);

  return { defaults, isLoading, error, refetch: fetchDefaults };
}

export function useTerrainConfigs() {
  const [configs, setConfigs] = useState<TerrainConfigResponse[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchConfigs = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<TerrainConfigResponse[]>(
        "/api/v1/terrain-configs"
      );
      setConfigs(result);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to fetch terrain configs"
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

export function useCreateTerrainConfig() {
  const [isLoading, setIsLoading] = useState(false);

  const createTerrainConfig = async (
    req: CreateTerrainConfigRequest
  ): Promise<TerrainConfigResponse> => {
    setIsLoading(true);
    try {
      return await api.post<TerrainConfigResponse>(
        "/api/v1/terrain-configs",
        req
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { createTerrainConfig, isLoading };
}

export function useUpdateTerrainConfig() {
  const [isLoading, setIsLoading] = useState(false);

  const updateTerrainConfig = async (
    id: string,
    req: UpdateTerrainConfigRequest
  ): Promise<TerrainConfigResponse> => {
    setIsLoading(true);
    try {
      return await api.put<TerrainConfigResponse>(
        `/api/v1/terrain-configs/${id}`,
        req
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { updateTerrainConfig, isLoading };
}

export function useDeleteTerrainConfig() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteTerrainConfig = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/terrain-configs/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteTerrainConfig, isLoading };
}

"use client";

import { useCallback, useState } from "react";
import { api } from "@/lib/api";
import type { Device } from "@/types/api";

/**
 * Fetch the authenticated user's registered devices.
 * The endpoint returns a plain Device[] (not paginated).
 */
export function useMyDevices() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<Device[]>("/api/v1/users/me/devices");
      setDevices(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch devices"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { devices, isLoading, error, fetch };
}

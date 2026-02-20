"use client";

import { useCallback, useState } from "react";
import { api } from "@/lib/api";
import type { Device, DeviceResolveResponse } from "@/types/api";

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

/**
 * Try to resolve the current browser to an existing device via cookie / UA
 * heuristic. Returns the resolve response which indicates whether a match
 * was found and, if not, the user's existing devices.
 */
export function useResolveDevice() {
  const [isLoading, setIsLoading] = useState(false);

  const resolve = async (): Promise<DeviceResolveResponse> => {
    setIsLoading(true);
    try {
      return await api.post<DeviceResolveResponse>(
        "/api/v1/users/me/devices/resolve"
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { resolve, isLoading };
}

/** Claim an existing device (re-use after e.g. clearing browser data). */
export function useClaimDevice() {
  const [isLoading, setIsLoading] = useState(false);

  const claimDevice = async (id: string): Promise<Device> => {
    setIsLoading(true);
    try {
      return await api.post<Device>(`/api/v1/users/me/devices/${id}/claim`);
    } finally {
      setIsLoading(false);
    }
  };

  return { claimDevice, isLoading };
}

/** Create a brand-new device via POST /api/v1/users/me/devices. */
export function useCreateDevice() {
  const [isLoading, setIsLoading] = useState(false);

  const createDevice = async (): Promise<Device> => {
    setIsLoading(true);
    try {
      return await api.post<Device>("/api/v1/users/me/devices", {
        name: "Web Browser",
        device_type: "web",
      });
    } finally {
      setIsLoading(false);
    }
  };

  return { createDevice, isLoading };
}

/** Delete a device by ID via DELETE /api/v1/devices/{id}. */
export function useDeleteDevice() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteDevice = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/devices/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteDevice, isLoading };
}

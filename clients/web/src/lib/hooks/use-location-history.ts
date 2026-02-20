"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { LocationHistoryEntry, LatestLocationEntry, Group } from "@/types/api";

/**
 * useLocationHistory fetches location history for a group within a time range.
 */
export function useLocationHistory() {
  const [data, setData] = useState<LocationHistoryEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(
    async (groupId: string, from: Date, to: Date) => {
      setIsLoading(true);
      setError(null);
      try {
        const result = await api.get<LocationHistoryEntry[]>(
          `/api/v1/groups/${groupId}/locations/history`,
          {
            params: {
              from: from.toISOString(),
              to: to.toISOString(),
            },
          }
        );
        setData(result ?? []);
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Failed to fetch location history"
        );
        setData([]);
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  const clear = useCallback(() => setData([]), []);

  return { data, isLoading, error, fetchHistory, clear };
}

/**
 * useMyLocationHistory fetches the current user's own location history.
 */
export function useMyLocationHistory() {
  const [data, setData] = useState<LocationHistoryEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(async (from: Date, to: Date) => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<LocationHistoryEntry[]>(
        "/api/v1/users/me/locations/history",
        {
          params: {
            from: from.toISOString(),
            to: to.toISOString(),
          },
        }
      );
      setData(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to fetch my location history"
      );
      setData([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const clear = useCallback(() => setData([]), []);

  return { data, isLoading, error, fetchHistory, clear };
}

/**
 * useVisibleHistory fetches location history for all users visible to the caller.
 * Admins see all users; non-admins see users who share a group with them.
 */
export function useVisibleHistory() {
  const [data, setData] = useState<LocationHistoryEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(async (from: Date, to: Date) => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<LocationHistoryEntry[]>(
        "/api/v1/locations/history",
        {
          params: {
            from: from.toISOString(),
            to: to.toISOString(),
          },
        }
      );
      setData(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to fetch location history"
      );
      setData([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const clear = useCallback(() => setData([]), []);

  return { data, isLoading, error, fetchHistory, clear };
}

/**
 * useUserLocationHistory fetches a specific user's location history.
 * Admins can query any user; non-admins can query users in shared groups.
 */
export function useUserLocationHistory() {
  const [data, setData] = useState<LocationHistoryEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(
    async (userId: string, from: Date, to: Date) => {
      setIsLoading(true);
      setError(null);
      try {
        const result = await api.get<LocationHistoryEntry[]>(
          `/api/v1/users/${userId}/locations/history`,
          {
            params: {
              from: from.toISOString(),
              to: to.toISOString(),
            },
          }
        );
        setData(result ?? []);
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Failed to fetch user location history"
        );
        setData([]);
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  const clear = useCallback(() => setData([]), []);

  return { data, isLoading, error, fetchHistory, clear };
}

/**
 * useMyGroups fetches the groups the current user belongs to.
 */
export function useMyGroups() {
  const [groups, setGroups] = useState<Group[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetchGroups = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await api.get<Group[]>("/api/v1/users/me/groups");
      setGroups(result ?? []);
    } catch {
      setGroups([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  return { groups, isLoading };
}

/**
 * useAllLocations fetches the latest location for every user (admin only).
 */
export function useAllLocations() {
  const [data, setData] = useState<LatestLocationEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAll = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<LatestLocationEntry[]>("/api/v1/locations");
      setData(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch all locations"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { data, isLoading, error, fetchAll };
}

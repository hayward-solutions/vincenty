"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useWebSocket } from "@/lib/websocket-context";
import type { WSLocationBroadcast, WSLocationSnapshot } from "@/types/api";

export interface UserLocation {
  user_id: string;
  username: string;
  display_name: string;
  device_id: string;
  device_name: string;
  is_primary: boolean;
  group_id: string;
  lat: number;
  lng: number;
  altitude?: number;
  heading?: number;
  speed?: number;
  timestamp: string;
}

/**
 * useLocations tracks the latest known position of other devices,
 * populated from WebSocket location_broadcast and location_snapshot messages.
 *
 * The map is keyed by device_id (not user_id), so each device gets its own entry.
 */
export function useLocations() {
  const { subscribe } = useWebSocket();

  // Map of device_id → latest UserLocation.
  // Using a ref + state combo: ref for fast updates, state for re-renders.
  const locationsRef = useRef<Map<string, UserLocation>>(new Map());
  const [locations, setLocations] = useState<Map<string, UserLocation>>(
    new Map()
  );

  // Batch state updates to avoid excessive re-renders
  const pendingRef = useRef(false);
  const flush = useCallback(() => {
    if (!pendingRef.current) {
      pendingRef.current = true;
      requestAnimationFrame(() => {
        setLocations(new Map(locationsRef.current));
        pendingRef.current = false;
      });
    }
  }, []);

  useEffect(() => {
    const unsubscribe = subscribe((type, payload) => {
      if (type === "location_broadcast") {
        const loc = payload as WSLocationBroadcast;
        locationsRef.current.set(loc.device_id, {
          user_id: loc.user_id,
          username: loc.username,
          display_name: loc.display_name,
          device_id: loc.device_id,
          device_name: loc.device_name,
          is_primary: loc.is_primary,
          group_id: loc.group_id,
          lat: loc.lat,
          lng: loc.lng,
          altitude: loc.altitude,
          heading: loc.heading,
          speed: loc.speed,
          timestamp: loc.timestamp,
        });
        flush();
      }

      if (type === "location_snapshot") {
        const snapshot = payload as WSLocationSnapshot;
        for (const loc of snapshot.locations) {
          locationsRef.current.set(loc.device_id, {
            user_id: loc.user_id,
            username: loc.username,
            display_name: loc.display_name,
            device_id: loc.device_id,
            device_name: loc.device_name,
            is_primary: loc.is_primary,
            group_id: snapshot.group_id,
            lat: loc.lat,
            lng: loc.lng,
            altitude: loc.altitude,
            heading: loc.heading,
            speed: loc.speed,
            timestamp: loc.timestamp,
          });
        }
        flush();
      }
    });

    return unsubscribe;
  }, [subscribe, flush]);

  /** Get locations filtered by group. */
  const getGroupLocations = useCallback(
    (groupId: string): UserLocation[] => {
      return Array.from(locations.values()).filter(
        (loc) => loc.group_id === groupId
      );
    },
    [locations]
  );

  return { locations, getGroupLocations };
}

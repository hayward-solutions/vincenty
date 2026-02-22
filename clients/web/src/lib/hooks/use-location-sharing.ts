"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useAuth } from "@/lib/auth-context";
import { useWebSocket } from "@/lib/websocket-context";
import type { ReactNode } from "react";
import React from "react";

// Minimum interval between location updates sent to server (ms).
const MIN_SEND_INTERVAL = 5_000;

// Number of consecutive POSITION_UNAVAILABLE errors before falling back to low accuracy.
const HIGH_ACCURACY_FAIL_THRESHOLD = 2;

export interface LocationState {
  lat: number;
  lng: number;
  altitude: number | null;
  heading: number | null;
  speed: number | null;
  accuracy: number | null;
}

interface LocationContextType {
  /** The most recent position from the browser. */
  lastPosition: LocationState | null;
  /** Error message if geolocation is unavailable or denied. */
  error: string | null;
}

const LocationContext = createContext<LocationContextType | undefined>(undefined);

export function LocationProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  const { connectionState, deviceId, sendMessage } = useWebSocket();

  const [lastPosition, setLastPosition] = useState<LocationState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [highAccuracy, setHighAccuracy] = useState(true);

  const watchIdRef = useRef<number | null>(null);
  const lastSentRef = useRef<number>(0);
  const failCountRef = useRef(0);

  // Keep mutable refs to WS state so the watchPosition callbacks can read
  // the latest values without restarting the watch on every WS state change.
  const wsStateRef = useRef({ connectionState, deviceId, sendMessage });
  useEffect(() => {
    wsStateRef.current = { connectionState, deviceId, sendMessage };
  }, [connectionState, deviceId, sendMessage]);

  // Start/stop geolocation watch based on authentication only.
  // Sending to the server is gated on WS state inside the callback.
  useEffect(() => {
    if (
      !isAuthenticated ||
      typeof navigator === "undefined" ||
      !navigator.geolocation
    ) {
      // Stop any active watch
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current);
        watchIdRef.current = null;
      }
      if (typeof navigator !== "undefined" && !navigator?.geolocation) {
        setError("Geolocation is not supported by this browser");
      }
      return;
    }

    setError(null);

    watchIdRef.current = navigator.geolocation.watchPosition(
      (position) => {
        failCountRef.current = 0;
        setError(null);
        const pos: LocationState = {
          lat: position.coords.latitude,
          lng: position.coords.longitude,
          altitude: position.coords.altitude,
          heading: position.coords.heading,
          speed: position.coords.speed,
          accuracy: position.coords.accuracy,
        };

        setLastPosition(pos);

        // Only send to server when WS is connected
        const ws = wsStateRef.current;
        if (ws.connectionState !== "connected" || !ws.deviceId) return;

        // Throttle outgoing messages client-side
        const now = Date.now();
        if (now - lastSentRef.current < MIN_SEND_INTERVAL) return;
        lastSentRef.current = now;

        ws.sendMessage("location_update", {
          device_id: ws.deviceId,
          lat: pos.lat,
          lng: pos.lng,
          altitude: pos.altitude ?? undefined,
          heading: pos.heading ?? undefined,
          speed: pos.speed ?? undefined,
          accuracy: pos.accuracy ?? undefined,
        });
      },
      (err) => {
        if (err.code === err.PERMISSION_DENIED) {
          setError("Location permission denied");
        } else if (err.code === err.POSITION_UNAVAILABLE) {
          failCountRef.current++;
          if (highAccuracy && failCountRef.current >= HIGH_ACCURACY_FAIL_THRESHOLD) {
            failCountRef.current = 0;
            setHighAccuracy(false);
            return;
          }
          setError("Location unavailable — check system location settings");
        }
        // TIMEOUT (code 3) stays silent — watchPosition retries automatically
      },
      {
        enableHighAccuracy: highAccuracy,
        timeout: 15_000,
        maximumAge: 5_000,
      }
    );

    return () => {
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current);
        watchIdRef.current = null;
      }
    };
  }, [isAuthenticated, highAccuracy]);

  const value = React.useMemo(
    () => ({ lastPosition, error }),
    [lastPosition, error]
  );

  return React.createElement(LocationContext.Provider, { value }, children);
}

export function useLocationSharing(): LocationContextType {
  const context = useContext(LocationContext);
  if (context === undefined) {
    throw new Error(
      "useLocationSharing must be used within a LocationProvider"
    );
  }
  return context;
}

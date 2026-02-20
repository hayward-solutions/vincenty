"use client";

import { useEffect, useRef, useState } from "react";
import { useWebSocket } from "@/lib/websocket-context";

// Minimum interval between location updates sent to server (ms).
const MIN_SEND_INTERVAL = 5_000;

// Number of consecutive POSITION_UNAVAILABLE errors before falling back to low accuracy.
const HIGH_ACCURACY_FAIL_THRESHOLD = 2;

interface LocationState {
  lat: number;
  lng: number;
  altitude: number | null;
  heading: number | null;
  speed: number | null;
  accuracy: number | null;
}

interface UseLocationSharingResult {
  /** The most recent position from the browser. */
  lastPosition: LocationState | null;
  /** Error message if geolocation is unavailable or denied. */
  error: string | null;
}

export function useLocationSharing(): UseLocationSharingResult {
  const { connectionState, deviceId, sendMessage } = useWebSocket();

  const [lastPosition, setLastPosition] = useState<LocationState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [highAccuracy, setHighAccuracy] = useState(true);

  const watchIdRef = useRef<number | null>(null);
  const lastSentRef = useRef<number>(0);
  const failCountRef = useRef(0);

  // Start/stop geolocation watch based on WS connection
  useEffect(() => {
    console.log("[Location] Effect check:", { connectionState, deviceId, highAccuracy, hasGeo: typeof navigator !== "undefined" && !!navigator.geolocation });
    if (
      connectionState !== "connected" ||
      !deviceId ||
      typeof navigator === "undefined" ||
      !navigator.geolocation
    ) {
      // Stop any active watch
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current);
        watchIdRef.current = null;
      }
      if (!navigator?.geolocation) {
        setError("Geolocation is not supported by this browser");
      }
      return;
    }

    setError(null);
    console.log("[Location] Starting watchPosition...", { enableHighAccuracy: highAccuracy });

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
        console.log("[Location] Browser position:", pos.lat, pos.lng, "accuracy:", pos.accuracy);

        // Throttle outgoing messages client-side
        const now = Date.now();
        if (now - lastSentRef.current < MIN_SEND_INTERVAL) return;
        lastSentRef.current = now;

        sendMessage("location_update", {
          device_id: deviceId,
          lat: pos.lat,
          lng: pos.lng,
          altitude: pos.altitude ?? undefined,
          heading: pos.heading ?? undefined,
          speed: pos.speed ?? undefined,
          accuracy: pos.accuracy ?? undefined,
        });
        console.log("[Location] Sent update to API via WS", { lat: pos.lat, lng: pos.lng, device_id: deviceId });
      },
      (err) => {
        console.log("[Location] Geolocation error:", err.code, err.message);
        if (err.code === err.PERMISSION_DENIED) {
          setError("Location permission denied");
        } else if (err.code === err.POSITION_UNAVAILABLE) {
          failCountRef.current++;
          if (highAccuracy && failCountRef.current >= HIGH_ACCURACY_FAIL_THRESHOLD) {
            console.log("[Location] High accuracy failed, falling back to low accuracy");
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
      console.log("[Location] Cleanup: clearing watch", watchIdRef.current);
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current);
        watchIdRef.current = null;
      }
    };
  }, [connectionState, deviceId, sendMessage, highAccuracy]);

  return { lastPosition, error };
}

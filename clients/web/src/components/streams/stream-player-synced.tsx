"use client";

import { useCallback, useEffect, useMemo, useRef } from "react";
import { useStreamLocations } from "@/lib/hooks/use-streams";
import { StreamViewer } from "@/components/streams/stream-viewer";
import type { StreamResponse, StreamLocationResponse } from "@/types/api";

interface StreamPlayerSyncedProps {
  stream: StreamResponse;
  className?: string;
  /** Called when the synced location changes during playback. */
  onLocationUpdate?: (location: {
    lat: number;
    lng: number;
    altitude?: number;
    heading?: number;
  }) => void;
}

/**
 * StreamPlayerSynced extends StreamViewer with location sync.
 * On video timeupdate, it maps the current playback time to the closest
 * GPS location and calls onLocationUpdate.
 */
export function StreamPlayerSynced({
  stream,
  className,
  onLocationUpdate,
}: StreamPlayerSyncedProps) {
  const { locations } = useStreamLocations(stream.id);
  const locationsRef = useRef<StreamLocationResponse[]>([]);

  // Keep locations in a ref for fast access in timeupdate handler
  useEffect(() => {
    locationsRef.current = locations;
  }, [locations]);

  // Stream start time in seconds since epoch
  const streamStartTime = useMemo(() => {
    return new Date(stream.started_at).getTime() / 1000;
  }, [stream.started_at]);

  const handleTimeUpdate = useCallback(
    (currentTime: number) => {
      if (!onLocationUpdate) return;
      const locs = locationsRef.current;
      if (locs.length === 0) return;

      // Map video currentTime (seconds from start) to absolute time
      const absoluteTime = streamStartTime + currentTime;

      // Binary search for the closest location by recorded_at
      const closest = findClosestLocation(locs, absoluteTime);
      if (closest) {
        onLocationUpdate({
          lat: closest.lat,
          lng: closest.lng,
          altitude: closest.altitude,
          heading: closest.heading,
        });
      }
    },
    [streamStartTime, onLocationUpdate]
  );

  return (
    <StreamViewer
      stream={stream}
      className={className}
      onTimeUpdate={handleTimeUpdate}
    />
  );
}

/**
 * Binary search to find the location entry closest to the given absolute
 * time (seconds since epoch).
 */
function findClosestLocation(
  locations: StreamLocationResponse[],
  targetTime: number
): StreamLocationResponse | null {
  if (locations.length === 0) return null;
  if (locations.length === 1) return locations[0];

  let low = 0;
  let high = locations.length - 1;

  // If target is before first or after last, clamp
  const firstTime = new Date(locations[0].recorded_at).getTime() / 1000;
  const lastTime =
    new Date(locations[locations.length - 1].recorded_at).getTime() / 1000;

  if (targetTime <= firstTime) return locations[0];
  if (targetTime >= lastTime) return locations[locations.length - 1];

  while (low <= high) {
    const mid = Math.floor((low + high) / 2);
    const midTime = new Date(locations[mid].recorded_at).getTime() / 1000;

    if (midTime === targetTime) {
      return locations[mid];
    } else if (midTime < targetTime) {
      low = mid + 1;
    } else {
      high = mid - 1;
    }
  }

  // low and high have crossed — pick the closer of the two
  const lowTime =
    low < locations.length
      ? new Date(locations[low].recorded_at).getTime() / 1000
      : Infinity;
  const highTime =
    high >= 0
      ? new Date(locations[high].recorded_at).getTime() / 1000
      : -Infinity;

  if (Math.abs(lowTime - targetTime) < Math.abs(highTime - targetTime)) {
    return locations[low];
  }
  return locations[high];
}

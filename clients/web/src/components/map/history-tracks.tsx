"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import type { LocationHistoryEntry } from "@/types/api";

interface HistoryTracksProps {
  map: maplibregl.Map;
  history: LocationHistoryEntry[];
  /** If set, only show history up to this time */
  playbackTime?: Date;
}

// Track colors per user
const TRACK_COLORS = [
  "#60a5fa", // blue-400
  "#f87171", // red-400
  "#4ade80", // green-400
  "#fbbf24", // amber-400
  "#a78bfa", // violet-400
  "#f472b6", // pink-400
  "#22d3ee", // cyan-400
  "#fb923c", // orange-400
];

/**
 * HistoryTracks renders polyline tracks on the map from location history data.
 * Groups points by user and draws a line per user.
 */
export function HistoryTracks({
  map,
  history,
  playbackTime,
}: HistoryTracksProps) {
  const sourceIdsRef = useRef<string[]>([]);
  const layerIdsRef = useRef<string[]>([]);

  useEffect(() => {
    // Clean up previous layers and sources
    layerIdsRef.current.forEach((id) => {
      if (map.getLayer(id)) map.removeLayer(id);
    });
    sourceIdsRef.current.forEach((id) => {
      if (map.getSource(id)) map.removeSource(id);
    });
    layerIdsRef.current = [];
    sourceIdsRef.current = [];

    if (history.length === 0) return;

    // Group by user
    const byUser = new Map<string, LocationHistoryEntry[]>();
    for (const entry of history) {
      // Filter by playback time if set
      if (playbackTime && new Date(entry.recorded_at) > playbackTime) {
        continue;
      }

      if (!byUser.has(entry.user_id)) {
        byUser.set(entry.user_id, []);
      }
      byUser.get(entry.user_id)!.push(entry);
    }

    let colorIdx = 0;
    byUser.forEach((points, userId) => {
      if (points.length < 2) return;

      const sourceId = `track-source-${userId}`;
      const layerId = `track-layer-${userId}`;
      const color = TRACK_COLORS[colorIdx % TRACK_COLORS.length];
      colorIdx++;

      const coordinates = points.map((p) => [p.lng, p.lat]);

      map.addSource(sourceId, {
        type: "geojson",
        data: {
          type: "Feature",
          properties: { user_id: userId, username: points[0].username },
          geometry: {
            type: "LineString",
            coordinates,
          },
        },
      });

      map.addLayer({
        id: layerId,
        type: "line",
        source: sourceId,
        layout: {
          "line-join": "round",
          "line-cap": "round",
        },
        paint: {
          "line-color": color,
          "line-width": 3,
          "line-opacity": 0.8,
        },
      });

      sourceIdsRef.current.push(sourceId);
      layerIdsRef.current.push(layerId);
    });

    // Cleanup
    return () => {
      layerIdsRef.current.forEach((id) => {
        if (map.getLayer(id)) map.removeLayer(id);
      });
      sourceIdsRef.current.forEach((id) => {
        if (map.getSource(id)) map.removeSource(id);
      });
      layerIdsRef.current = [];
      sourceIdsRef.current = [];
    };
  }, [map, history, playbackTime]);

  return null;
}

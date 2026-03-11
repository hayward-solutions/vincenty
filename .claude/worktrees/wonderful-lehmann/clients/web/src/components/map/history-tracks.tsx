"use client";

import { useEffect, useMemo, useRef } from "react";
import maplibregl from "maplibre-gl";
import type { LocationHistoryEntry } from "@/types/api";

interface HistoryTracksProps {
  map: maplibregl.Map;
  history: LocationHistoryEntry[];
  /** If set, only show history up to this time */
  playbackTime?: Date;
}

// Track colors per user-device pair
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

/** Composite key for grouping history points by user+device. */
function trackKey(entry: LocationHistoryEntry): string {
  return `${entry.user_id}:${entry.device_id}`;
}

interface TrackGroup {
  entries: LocationHistoryEntry[];
  color: string;
  sourceId: string;
  layerId: string;
}

/** Empty GeoJSON LineString used as initial source data. */
function emptyLineString(): GeoJSON.Feature {
  return {
    type: "Feature",
    properties: {},
    geometry: { type: "LineString", coordinates: [] },
  };
}

/** Empty GeoJSON FeatureCollection for head markers. */
function emptyFeatureCollection(): GeoJSON.FeatureCollection {
  return { type: "FeatureCollection", features: [] };
}

const HEAD_SOURCE_ID = "track-heads-source";
const HEAD_LAYER_ID = "track-heads-layer";

/**
 * HistoryTracks renders polyline tracks on the map from location history data.
 * Groups points by user+device so each device gets its own track line.
 *
 * Layer lifecycle (create/destroy) is separated from data updates so that
 * playback-time changes only call source.setData() instead of tearing down
 * and re-adding MapLibre layers every frame.
 */
export function HistoryTracks({
  map,
  history,
  playbackTime,
}: HistoryTracksProps) {
  const layerIdsRef = useRef<string[]>([]);
  const sourceIdsRef = useRef<string[]>([]);

  // -----------------------------------------------------------------------
  // 1. Compute stable track groups from history (independent of playbackTime)
  // -----------------------------------------------------------------------
  const trackGroups = useMemo((): Map<string, TrackGroup> => {
    const byTrack = new Map<string, LocationHistoryEntry[]>();
    for (const entry of history) {
      const key = trackKey(entry);
      if (!byTrack.has(key)) {
        byTrack.set(key, []);
      }
      byTrack.get(key)!.push(entry);
    }

    const groups = new Map<string, TrackGroup>();
    let colorIdx = 0;
    byTrack.forEach((entries, key) => {
      // Require at least 1 point. Tracks with a single point render no line
      // but still show a head-marker dot so users with sparse history are visible.
      if (entries.length < 1) return;

      const safeId = key.replace(/:/g, "-");
      groups.set(key, {
        entries,
        color: TRACK_COLORS[colorIdx % TRACK_COLORS.length],
        sourceId: `track-source-${safeId}`,
        layerId: `track-layer-${safeId}`,
      });
      colorIdx++;
    });

    return groups;
  }, [history]);

  // -----------------------------------------------------------------------
  // 2. Layer lifecycle — create / destroy sources + layers when tracks change
  // -----------------------------------------------------------------------
  useEffect(() => {
    // Clean up any previous layers/sources
    try {
      layerIdsRef.current.forEach((id) => {
        if (map.getLayer(id)) map.removeLayer(id);
      });
      sourceIdsRef.current.forEach((id) => {
        if (map.getSource(id)) map.removeSource(id);
      });
    } catch {
      // Map may already be destroyed during navigation
    }
    layerIdsRef.current = [];
    sourceIdsRef.current = [];

    if (trackGroups.size === 0) return;

    // Create one source + layer per track (seeded with empty data)
    trackGroups.forEach((group) => {
      map.addSource(group.sourceId, {
        type: "geojson",
        data: emptyLineString(),
      });

      map.addLayer({
        id: group.layerId,
        type: "line",
        source: group.sourceId,
        layout: {
          "line-join": "round",
          "line-cap": "round",
        },
        paint: {
          "line-color": group.color,
          "line-width": 3,
          "line-opacity": 0.8,
        },
      });

      sourceIdsRef.current.push(group.sourceId);
      layerIdsRef.current.push(group.layerId);
    });

    // Head markers — single source with data-driven color, one dot per track
    map.addSource(HEAD_SOURCE_ID, {
      type: "geojson",
      data: emptyFeatureCollection(),
    });

    map.addLayer({
      id: HEAD_LAYER_ID,
      type: "circle",
      source: HEAD_SOURCE_ID,
      paint: {
        "circle-radius": 6,
        "circle-color": ["get", "color"],
        "circle-stroke-color": "#ffffff",
        "circle-stroke-width": 2,
        "circle-opacity": 0.9,
      },
    });

    sourceIdsRef.current.push(HEAD_SOURCE_ID);
    layerIdsRef.current.push(HEAD_LAYER_ID);

    return () => {
      try {
        layerIdsRef.current.forEach((id) => {
          if (map.getLayer(id)) map.removeLayer(id);
        });
        sourceIdsRef.current.forEach((id) => {
          if (map.getSource(id)) map.removeSource(id);
        });
      } catch {
        // Map may already be destroyed during navigation
      }
      layerIdsRef.current = [];
      sourceIdsRef.current = [];
    };
  }, [map, trackGroups]);

  // -----------------------------------------------------------------------
  // 3. Data updates — efficiently update source data when playbackTime changes
  //    Uses source.setData() so MapLibre layers are never torn down mid-playback.
  // -----------------------------------------------------------------------
  useEffect(() => {
    if (trackGroups.size === 0) return;

    const headFeatures: GeoJSON.Feature[] = [];

    trackGroups.forEach((group) => {
      // Filter entries to those at or before the current playback cursor
      const filtered = playbackTime
        ? group.entries.filter(
            (e) => new Date(e.recorded_at) <= playbackTime
          )
        : group.entries;

      // Push updated coordinates to the existing line source
      const source = map.getSource(
        group.sourceId
      ) as maplibregl.GeoJSONSource | undefined;
      if (source) {
        source.setData({
          type: "Feature",
          properties: {
            user_id: group.entries[0].user_id,
            device_id: group.entries[0].device_id,
            username: group.entries[0].username,
            device_name: group.entries[0].device_name,
          },
          geometry: {
            type: "LineString",
            coordinates: filtered.map((p) => [p.lng, p.lat]),
          },
        });
      }

      // Build a head marker at the last visible point for this track
      if (filtered.length > 0) {
        const last = filtered[filtered.length - 1];
        headFeatures.push({
          type: "Feature",
          properties: {
            color: group.color,
            username: last.display_name || last.username,
            device_name: last.device_name,
          },
          geometry: {
            type: "Point",
            coordinates: [last.lng, last.lat],
          },
        });
      }
    });

    // Push all head markers in one call
    const headSource = map.getSource(
      HEAD_SOURCE_ID
    ) as maplibregl.GeoJSONSource | undefined;
    if (headSource) {
      headSource.setData({
        type: "FeatureCollection",
        features: headFeatures,
      });
    }
  }, [map, trackGroups, playbackTime]);

  return null;
}

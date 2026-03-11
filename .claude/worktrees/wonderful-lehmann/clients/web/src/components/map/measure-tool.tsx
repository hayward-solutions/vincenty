"use client";

import { useEffect, useRef, useCallback } from "react";
import maplibregl from "maplibre-gl";
import type { GeoJSON } from "geojson";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface MeasureResult {
  /** Distance of each committed segment in meters. */
  segments: number[];
  /** Sum of all segment distances in meters. */
  total: number;
  /** Circle radius in meters (circle mode only). */
  radius?: number;
  /** Circle area in m² (circle mode only). */
  area?: number;
}

interface MeasureToolProps {
  map: maplibregl.Map;
  active: boolean;
  mode: "line" | "circle";
  /** Increment to force a reset (clear points) without toggling active. */
  resetKey: number;
  onMeasurementsChange: (result: MeasureResult) => void;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type Coord = [number, number]; // [lng, lat]

/** Haversine distance between two [lng,lat] coordinates, returns meters. */
function haversineDistance([lng1, lat1]: Coord, [lng2, lat2]: Coord): number {
  const R = 6_371_000; // earth radius in metres
  const toRad = (d: number) => (d * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

/** Format a distance in meters to a human-readable string. */
export function formatDistance(meters: number): string {
  if (meters < 1000) return `${Math.round(meters)}m`;
  return `${(meters / 1000).toFixed(2)}km`;
}

/** Format an area in m² to a human-readable string. */
export function formatArea(sqMeters: number): string {
  if (sqMeters < 1_000_000)
    return `${Math.round(sqMeters).toLocaleString()}m²`;
  return `${(sqMeters / 1_000_000).toFixed(2)}km²`;
}

/** Generate a circle polygon approximation (64 steps). */
function generateCircleCoords(
  center: Coord,
  radiusMeters: number,
  steps = 64
): Coord[] {
  const coords: Coord[] = [];
  const dLat = (radiusMeters / 6_371_000) * (180 / Math.PI);
  const dLng = dLat / Math.cos((center[1] * Math.PI) / 180);
  for (let i = 0; i <= steps; i++) {
    const angle = (i / steps) * 2 * Math.PI;
    coords.push([
      center[0] + dLng * Math.sin(angle),
      center[1] + dLat * Math.cos(angle),
    ]);
  }
  return coords;
}

/** Midpoint of two coordinates. */
function midpoint(a: Coord, b: Coord): Coord {
  return [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2];
}

// ---------------------------------------------------------------------------
// Layer / source IDs
// ---------------------------------------------------------------------------

const SOURCE_ID = "measure-geojson";

const LINE_LAYERS = [
  "measure-lines",
  "measure-pending",
  "measure-points",
  "measure-labels",
] as const;

const CIRCLE_LAYERS = [
  "measure-fill",
  "measure-outline",
  "measure-radius",
  "measure-center",
  "measure-labels",
] as const;

const ALL_LAYERS = Array.from(new Set([...LINE_LAYERS, ...CIRCLE_LAYERS]));

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

function cleanup(map: maplibregl.Map) {
  try {
    for (const id of ALL_LAYERS) {
      if (map.getLayer(id)) map.removeLayer(id);
    }
    if (map.getSource(SOURCE_ID)) map.removeSource(SOURCE_ID);
  } catch {
    /* map may already be destroyed */
  }
}

// ---------------------------------------------------------------------------
// Layer setup helpers
// ---------------------------------------------------------------------------

function addLineLayers(map: maplibregl.Map) {
  // Committed segments (solid)
  map.addLayer({
    id: "measure-lines",
    type: "line",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "line"],
    paint: {
      "line-color": "#f43f5e",
      "line-width": 2.5,
      "line-opacity": 0.9,
    },
  });

  // Pending segment (dashed, from last point to cursor)
  map.addLayer({
    id: "measure-pending",
    type: "line",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "pending"],
    paint: {
      "line-color": "#f43f5e",
      "line-width": 2,
      "line-dasharray": [3, 3],
      "line-opacity": 0.6,
    },
  });

  // Placed point markers
  map.addLayer({
    id: "measure-points",
    type: "circle",
    source: SOURCE_ID,
    filter: [
      "all",
      ["==", "$type", "Point"],
      ["==", ["get", "kind"], "point"],
    ],
    paint: {
      "circle-radius": 5,
      "circle-color": "#f43f5e",
      "circle-stroke-color": "#ffffff",
      "circle-stroke-width": 2,
    },
  });

  // Distance labels at segment midpoints
  map.addLayer({
    id: "measure-labels",
    type: "symbol",
    source: SOURCE_ID,
    filter: [
      "all",
      ["==", "$type", "Point"],
      ["has", "label"],
    ],
    layout: {
      "text-field": ["get", "label"],
      "text-size": 12,
      "text-font": ["Open Sans Bold", "Arial Unicode MS Bold"],
      "text-offset": [0, -1.5],
      "text-allow-overlap": true,
    },
    paint: {
      "text-color": "#f43f5e",
      "text-halo-color": "#ffffff",
      "text-halo-width": 1.5,
    },
  });
}

function addCircleLayers(map: maplibregl.Map) {
  // Semi-transparent fill
  map.addLayer({
    id: "measure-fill",
    type: "fill",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "circle-fill"],
    paint: {
      "fill-color": "#f43f5e",
      "fill-opacity": 0.1,
    },
  });

  // Circle outline
  map.addLayer({
    id: "measure-outline",
    type: "line",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "circle-fill"],
    paint: {
      "line-color": "#f43f5e",
      "line-width": 2,
      "line-opacity": 0.8,
    },
  });

  // Radius line (dashed)
  map.addLayer({
    id: "measure-radius",
    type: "line",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "radius"],
    paint: {
      "line-color": "#f43f5e",
      "line-width": 2,
      "line-dasharray": [3, 3],
      "line-opacity": 0.8,
    },
  });

  // Center point
  map.addLayer({
    id: "measure-center",
    type: "circle",
    source: SOURCE_ID,
    filter: [
      "all",
      ["==", "$type", "Point"],
      ["==", ["get", "kind"], "point"],
    ],
    paint: {
      "circle-radius": 5,
      "circle-color": "#f43f5e",
      "circle-stroke-color": "#ffffff",
      "circle-stroke-width": 2,
    },
  });

  // Labels
  map.addLayer({
    id: "measure-labels",
    type: "symbol",
    source: SOURCE_ID,
    filter: [
      "all",
      ["==", "$type", "Point"],
      ["has", "label"],
    ],
    layout: {
      "text-field": ["get", "label"],
      "text-size": 12,
      "text-font": ["Open Sans Bold", "Arial Unicode MS Bold"],
      "text-offset": [0, -1.5],
      "text-allow-overlap": true,
    },
    paint: {
      "text-color": "#f43f5e",
      "text-halo-color": "#ffffff",
      "text-halo-width": 1.5,
    },
  });
}

// ---------------------------------------------------------------------------
// GeoJSON builders
// ---------------------------------------------------------------------------

function buildLineGeoJSON(
  points: Coord[],
  cursor: Coord | null
): { geojson: GeoJSON.FeatureCollection; segments: number[]; total: number } {
  const features: GeoJSON.Feature[] = [];
  const segments: number[] = [];
  let total = 0;

  // Committed segments + midpoint labels
  for (let i = 1; i < points.length; i++) {
    const dist = haversineDistance(points[i - 1], points[i]);
    segments.push(dist);
    total += dist;

    features.push({
      type: "Feature",
      properties: { kind: "line" },
      geometry: { type: "LineString", coordinates: [points[i - 1], points[i]] },
    });

    features.push({
      type: "Feature",
      properties: { kind: "label", label: formatDistance(dist) },
      geometry: { type: "Point", coordinates: midpoint(points[i - 1], points[i]) },
    });
  }

  // Pending line to cursor
  if (cursor && points.length > 0) {
    const last = points[points.length - 1];
    const pendingDist = haversineDistance(last, cursor);

    features.push({
      type: "Feature",
      properties: { kind: "pending" },
      geometry: { type: "LineString", coordinates: [last, cursor] },
    });

    features.push({
      type: "Feature",
      properties: { kind: "label", label: formatDistance(pendingDist) },
      geometry: { type: "Point", coordinates: midpoint(last, cursor) },
    });
  }

  // Point markers
  for (const pt of points) {
    features.push({
      type: "Feature",
      properties: { kind: "point" },
      geometry: { type: "Point", coordinates: pt },
    });
  }

  return {
    geojson: { type: "FeatureCollection", features },
    segments,
    total,
  };
}

function buildCircleGeoJSON(
  center: Coord | null,
  edge: Coord | null
): {
  geojson: GeoJSON.FeatureCollection;
  radius: number;
  area: number;
} {
  const features: GeoJSON.Feature[] = [];
  let radius = 0;
  let area = 0;

  if (center) {
    features.push({
      type: "Feature",
      properties: { kind: "point" },
      geometry: { type: "Point", coordinates: center },
    });

    if (edge) {
      radius = haversineDistance(center, edge);
      area = Math.PI * radius * radius;

      // Circle polygon
      const circleCoords = generateCircleCoords(center, radius);
      features.push({
        type: "Feature",
        properties: { kind: "circle-fill" },
        geometry: { type: "Polygon", coordinates: [circleCoords] },
      });

      // Radius line
      features.push({
        type: "Feature",
        properties: { kind: "radius" },
        geometry: { type: "LineString", coordinates: [center, edge] },
      });

      // Label at midpoint of radius
      features.push({
        type: "Feature",
        properties: { kind: "label", label: formatDistance(radius) },
        geometry: { type: "Point", coordinates: midpoint(center, edge) },
      });
    }
  }

  return { geojson: { type: "FeatureCollection", features }, radius, area };
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * MeasureTool is a renderless component that manages map interactions
 * for distance measurement (point-to-point or circle/radius).
 */
export function MeasureTool({
  map,
  active,
  mode,
  resetKey,
  onMeasurementsChange,
}: MeasureToolProps) {
  const pointsRef = useRef<Coord[]>([]);
  const cursorRef = useRef<Coord | null>(null);
  const finalizedRef = useRef(false);
  const onMeasurementsChangeRef = useRef(onMeasurementsChange);
  onMeasurementsChangeRef.current = onMeasurementsChange;

  const updateSource = useCallback(
    (geojson: GeoJSON.FeatureCollection) => {
      try {
        const src = map.getSource(SOURCE_ID);
        if (src && "setData" in src) {
          (src as maplibregl.GeoJSONSource).setData(geojson);
        }
      } catch {
        /* source may not exist yet */
      }
    },
    [map]
  );

  // Main effect — manages the full lifecycle of the measure tool
  useEffect(() => {
    if (!active) return;

    // Reset state
    pointsRef.current = [];
    cursorRef.current = null;
    finalizedRef.current = false;
    onMeasurementsChangeRef.current({
      segments: [],
      total: 0,
      radius: undefined,
      area: undefined,
    });

    // Setup
    cleanup(map);
    map.getCanvas().style.cursor = "crosshair";

    const emptyFC: GeoJSON.FeatureCollection = {
      type: "FeatureCollection",
      features: [],
    };

    map.addSource(SOURCE_ID, { type: "geojson", data: emptyFC });

    if (mode === "line") {
      addLineLayers(map);
    } else {
      addCircleLayers(map);
    }

    // -----------------------------------------------------------------------
    // Handlers
    // -----------------------------------------------------------------------

    function rebuild() {
      if (mode === "line") {
        const { geojson, segments, total } = buildLineGeoJSON(
          pointsRef.current,
          finalizedRef.current ? null : cursorRef.current
        );
        updateSource(geojson);
        onMeasurementsChangeRef.current({ segments, total });
      } else {
        const center = pointsRef.current[0] ?? null;
        const edge =
          pointsRef.current[1] ?? (center ? cursorRef.current : null);
        const { geojson, radius, area } = buildCircleGeoJSON(center, edge);
        updateSource(geojson);
        onMeasurementsChangeRef.current({
          segments: [],
          total: 0,
          radius: radius || undefined,
          area: area || undefined,
        });
      }
    }

    function handleClick(e: maplibregl.MapMouseEvent) {
      const coord: Coord = [e.lngLat.lng, e.lngLat.lat];

      if (mode === "line") {
        if (finalizedRef.current) return;
        pointsRef.current = [...pointsRef.current, coord];
      } else {
        // Circle: first click = center, second = edge, third+ = reset
        if (pointsRef.current.length >= 2) {
          // Start new measurement
          pointsRef.current = [coord];
        } else {
          pointsRef.current = [...pointsRef.current, coord];
        }
      }
      rebuild();
    }

    function handleMouseMove(e: maplibregl.MapMouseEvent) {
      cursorRef.current = [e.lngLat.lng, e.lngLat.lat];
      if (pointsRef.current.length > 0 && !finalizedRef.current) {
        rebuild();
      }
    }

    function handleDblClick(e: maplibregl.MapMouseEvent) {
      if (mode !== "line") return;
      e.preventDefault();
      finalizedRef.current = true;
      // Remove the duplicate point added by the second click of the dblclick
      if (pointsRef.current.length > 1) {
        pointsRef.current = pointsRef.current.slice(0, -1);
      }
      rebuild();
    }

    map.on("click", handleClick);
    map.on("mousemove", handleMouseMove);
    map.on("dblclick", handleDblClick);

    return () => {
      map.off("click", handleClick);
      map.off("mousemove", handleMouseMove);
      map.off("dblclick", handleDblClick);
      cleanup(map);
      try {
        map.getCanvas().style.cursor = "";
      } catch {
        /* map destroyed */
      }
    };
  }, [map, active, mode, resetKey, updateSource]);

  return null;
}

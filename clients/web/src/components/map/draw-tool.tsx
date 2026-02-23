"use client";

import { useEffect, useRef, useCallback } from "react";
import maplibregl from "maplibre-gl";
import type { GeoJSON } from "geojson";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type DrawMode = "line" | "circle" | "rectangle";

export interface DrawStyle {
  stroke: string;
  fill: string;
  strokeWidth: number;
}

/** A completed shape ready to be added to a drawing's FeatureCollection. */
export interface CompletedShape {
  feature: GeoJSON.Feature;
}

interface DrawToolProps {
  map: maplibregl.Map;
  active: boolean;
  mode: DrawMode;
  style: DrawStyle;
  /** Increment to cancel the current in-progress shape. */
  resetKey: number;
  /** Previously completed shapes from this session — rendered alongside the in-progress preview. */
  completedFeatures: GeoJSON.Feature[];
  /** Called when a shape is completed (double-click for line, second click for circle/rect). */
  onShapeComplete: (shape: CompletedShape) => void;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type Coord = [number, number]; // [lng, lat]

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

/** Haversine distance between two [lng,lat] coordinates, returns meters. */
function haversineDistance([lng1, lat1]: Coord, [lng2, lat2]: Coord): number {
  const R = 6_371_000;
  const toRad = (d: number) => (d * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

/**
 * Build an oriented rectangle from 3 points.
 * A→B defines one edge (direction + width).
 * C determines the depth perpendicular to edge AB.
 */
function orientedRectCoords(a: Coord, b: Coord, c: Coord): Coord[] {
  // cos(lat) correction so the 90° rotation is true in physical space
  const midLat = ((a[1] + b[1]) / 2) * (Math.PI / 180);
  const cosLat = Math.cos(midLat);

  // Edge AB in equalized (approximate metric) space
  const abX = (b[0] - a[0]) * cosLat;
  const abY = b[1] - a[1];

  // Perpendicular (90° rotation in equalized space)
  const perpX = -abY;
  const perpY = abX;
  const perpLen = Math.sqrt(perpX ** 2 + perpY ** 2);
  if (perpLen === 0) return [a, b, a]; // degenerate edge
  const perpUnitX = perpX / perpLen;
  const perpUnitY = perpY / perpLen;

  // Project AC onto the perpendicular (in equalized space)
  const acX = (c[0] - a[0]) * cosLat;
  const acY = c[1] - a[1];
  const depth = acX * perpUnitX + acY * perpUnitY;

  // Offset back in lng/lat
  const offsetLng = (depth * perpUnitX) / cosLat;
  const offsetLat = depth * perpUnitY;

  return [
    a,
    b,
    [b[0] + offsetLng, b[1] + offsetLat],
    [a[0] + offsetLng, a[1] + offsetLat],
    a, // close the ring
  ];
}

// ---------------------------------------------------------------------------
// Source / Layer IDs
// ---------------------------------------------------------------------------

const SOURCE_ID = "draw-tool-geojson";

const ALL_LAYERS = [
  "draw-tool-fill",
  "draw-tool-outline",
  "draw-tool-line",
  "draw-tool-pending",
  "draw-tool-points",
] as const;

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
// Layer setup
// ---------------------------------------------------------------------------

function addLayers(map: maplibregl.Map) {
  // Fill for polygons (circles and rectangles) — data-driven per feature
  map.addLayer({
    id: "draw-tool-fill",
    type: "fill",
    source: SOURCE_ID,
    filter: ["==", "$type", "Polygon"],
    paint: {
      "fill-color": ["get", "fill"],
      "fill-opacity": ["case", ["==", ["get", "fill"], "transparent"], 0, 0.25],
    },
  });

  // Outline for polygons (previews + completed shapes)
  map.addLayer({
    id: "draw-tool-outline",
    type: "line",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "shape"],
    paint: {
      "line-color": ["get", "stroke"],
      "line-width": ["get", "strokeWidth"],
      "line-opacity": 0.9,
    },
  });

  // Committed line segments + completed line features
  map.addLayer({
    id: "draw-tool-line",
    type: "line",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "line"],
    paint: {
      "line-color": ["get", "stroke"],
      "line-width": ["get", "strokeWidth"],
      "line-opacity": 0.9,
    },
  });

  // Pending line (dashed, from last point to cursor)
  map.addLayer({
    id: "draw-tool-pending",
    type: "line",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "pending"],
    paint: {
      "line-color": ["get", "stroke"],
      "line-width": ["get", "strokeWidth"],
      "line-dasharray": [3, 3],
      "line-opacity": 0.5,
    },
  });

  // Vertex points
  map.addLayer({
    id: "draw-tool-points",
    type: "circle",
    source: SOURCE_ID,
    filter: ["==", ["get", "kind"], "point"],
    paint: {
      "circle-radius": 5,
      "circle-color": ["get", "stroke"],
      "circle-stroke-color": "#ffffff",
      "circle-stroke-width": 2,
    },
  });
}

// ---------------------------------------------------------------------------
// GeoJSON builders for in-progress preview
// ---------------------------------------------------------------------------

function buildLinePreview(
  points: Coord[],
  cursor: Coord | null,
  finalized: boolean,
  style: DrawStyle
): GeoJSON.FeatureCollection {
  const features: GeoJSON.Feature[] = [];

  // Committed line
  if (points.length >= 2) {
    features.push({
      type: "Feature",
      properties: { kind: "line", stroke: style.stroke, fill: style.fill, strokeWidth: style.strokeWidth },
      geometry: { type: "LineString", coordinates: points },
    });
  }

  // Pending segment
  if (cursor && points.length > 0 && !finalized) {
    const last = points[points.length - 1];
    features.push({
      type: "Feature",
      properties: { kind: "pending", stroke: style.stroke, strokeWidth: style.strokeWidth },
      geometry: { type: "LineString", coordinates: [last, cursor] },
    });
  }

  // Vertex markers
  for (const pt of points) {
    features.push({
      type: "Feature",
      properties: { kind: "point", stroke: style.stroke },
      geometry: { type: "Point", coordinates: pt },
    });
  }

  return { type: "FeatureCollection", features };
}

function buildCirclePreview(
  center: Coord | null,
  edge: Coord | null,
  style: DrawStyle
): GeoJSON.FeatureCollection {
  const features: GeoJSON.Feature[] = [];

  if (center) {
    features.push({
      type: "Feature",
      properties: { kind: "point", stroke: style.stroke },
      geometry: { type: "Point", coordinates: center },
    });

    if (edge) {
      const radius = haversineDistance(center, edge);
      const circleCoords = generateCircleCoords(center, radius);
      features.push({
        type: "Feature",
        properties: {
          kind: "shape",
          shapeType: "circle",
          center,
          radiusMeters: radius,
          stroke: style.stroke,
          fill: style.fill,
          strokeWidth: style.strokeWidth,
        },
        geometry: { type: "Polygon", coordinates: [circleCoords] },
      });
    }
  }

  return { type: "FeatureCollection", features };
}

function buildRectPreview(
  points: Coord[],
  cursor: Coord | null,
  style: DrawStyle
): GeoJSON.FeatureCollection {
  const features: GeoJSON.Feature[] = [];
  const a = points[0] ?? null;
  const b = points[1] ?? null;

  // Phase 1: One point placed — show vertex + pending line to cursor
  if (a && !b) {
    features.push({
      type: "Feature",
      properties: { kind: "point", stroke: style.stroke },
      geometry: { type: "Point", coordinates: a },
    });
    if (cursor) {
      features.push({
        type: "Feature",
        properties: { kind: "pending", stroke: style.stroke, strokeWidth: style.strokeWidth },
        geometry: { type: "LineString", coordinates: [a, cursor] },
      });
    }
  }

  // Phase 2: Two points placed — show edge A→B + rectangle preview using cursor as depth
  if (a && b) {
    features.push({
      type: "Feature",
      properties: { kind: "point", stroke: style.stroke },
      geometry: { type: "Point", coordinates: a },
    });
    features.push({
      type: "Feature",
      properties: { kind: "point", stroke: style.stroke },
      geometry: { type: "Point", coordinates: b },
    });
    // Committed edge line A→B
    features.push({
      type: "Feature",
      properties: { kind: "line", stroke: style.stroke, strokeWidth: style.strokeWidth },
      geometry: { type: "LineString", coordinates: [a, b] },
    });
    // Rectangle preview (depth from cursor or third point)
    const c = points[2] ?? cursor;
    if (c) {
      const coords = orientedRectCoords(a, b, c);
      features.push({
        type: "Feature",
        properties: {
          kind: "shape",
          shapeType: "rectangle",
          stroke: style.stroke,
          fill: style.fill,
          strokeWidth: style.strokeWidth,
        },
        geometry: { type: "Polygon", coordinates: [coords] },
      });
    }
  }

  return { type: "FeatureCollection", features };
}

// ---------------------------------------------------------------------------
// Build final GeoJSON Feature from completed shape
// ---------------------------------------------------------------------------

function buildLineFeature(points: Coord[], style: DrawStyle): GeoJSON.Feature {
  return {
    type: "Feature",
    properties: {
      kind: "line",
      shapeType: "line",
      stroke: style.stroke,
      fill: style.fill,
      strokeWidth: style.strokeWidth,
    },
    geometry: { type: "LineString", coordinates: [...points] },
  };
}

function buildCircleFeature(
  center: Coord,
  edge: Coord,
  style: DrawStyle
): GeoJSON.Feature {
  const radius = haversineDistance(center, edge);
  const circleCoords = generateCircleCoords(center, radius);
  return {
    type: "Feature",
    properties: {
      kind: "shape",
      shapeType: "circle",
      center,
      radiusMeters: radius,
      stroke: style.stroke,
      fill: style.fill,
      strokeWidth: style.strokeWidth,
    },
    geometry: { type: "Polygon", coordinates: [circleCoords] },
  };
}

function buildRectFeature(
  a: Coord,
  b: Coord,
  c: Coord,
  style: DrawStyle
): GeoJSON.Feature {
  const coords = orientedRectCoords(a, b, c);
  return {
    type: "Feature",
    properties: {
      kind: "shape",
      shapeType: "rectangle",
      stroke: style.stroke,
      fill: style.fill,
      strokeWidth: style.strokeWidth,
    },
    geometry: { type: "Polygon", coordinates: [coords] },
  };
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * DrawTool is a renderless component that manages map interactions for
 * drawing shapes (lines, circles, rectangles) on the map.
 */
export function DrawTool({
  map,
  active,
  mode,
  style,
  resetKey,
  completedFeatures,
  onShapeComplete,
}: DrawToolProps) {
  const pointsRef = useRef<Coord[]>([]);
  const cursorRef = useRef<Coord | null>(null);
  const finalizedRef = useRef(false);
  const onShapeCompleteRef = useRef(onShapeComplete);
  onShapeCompleteRef.current = onShapeComplete;
  const styleRef = useRef(style);
  styleRef.current = style;
  const completedFeaturesRef = useRef(completedFeatures);
  completedFeaturesRef.current = completedFeatures;
  const rebuildRef = useRef<(() => void) | null>(null);

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

  // Main lifecycle effect
  useEffect(() => {
    if (!active) return;

    // Reset state
    pointsRef.current = [];
    cursorRef.current = null;
    finalizedRef.current = false;

    // Setup
    cleanup(map);
    map.getCanvas().style.cursor = "crosshair";

    const emptyFC: GeoJSON.FeatureCollection = {
      type: "FeatureCollection",
      features: [],
    };
    map.addSource(SOURCE_ID, { type: "geojson", data: emptyFC });
    addLayers(map);

    // -------------------------------------------------------------------
    // Handlers
    // -------------------------------------------------------------------

    function rebuild() {
      const s = styleRef.current;
      let previewFC: GeoJSON.FeatureCollection;

      if (mode === "line") {
        previewFC = buildLinePreview(
          pointsRef.current,
          finalizedRef.current ? null : cursorRef.current,
          finalizedRef.current,
          s
        );
      } else if (mode === "circle") {
        const center = pointsRef.current[0] ?? null;
        const edge =
          pointsRef.current[1] ?? (center ? cursorRef.current : null);
        previewFC = buildCirclePreview(center, edge, s);
      } else {
        // rectangle — 3-click: A (start), B (orientation), C (depth)
        previewFC = buildRectPreview(pointsRef.current, cursorRef.current, s);
      }

      // Prepend previously completed shapes so they stay visible
      const allFeatures = [
        ...completedFeaturesRef.current,
        ...previewFC.features,
      ];
      updateSource({ type: "FeatureCollection", features: allFeatures });
    }

    rebuildRef.current = rebuild;
    rebuild(); // Render completed shapes immediately (e.g., after mode switch)

    function completeAndReset() {
      const s = styleRef.current;
      const pts = pointsRef.current;
      let feature: GeoJSON.Feature | null = null;

      if (mode === "line" && pts.length >= 2) {
        feature = buildLineFeature(pts, s);
      } else if (mode === "circle" && pts.length >= 2) {
        feature = buildCircleFeature(pts[0], pts[1], s);
      } else if (mode === "rectangle" && pts.length >= 3) {
        feature = buildRectFeature(pts[0], pts[1], pts[2], s);
      }

      if (feature) {
        onShapeCompleteRef.current({ feature });
      }

      // Reset in-progress state for next shape — completed shapes
      // stay visible via completedFeaturesRef (updated by parent).
      pointsRef.current = [];
      cursorRef.current = null;
      finalizedRef.current = false;
      rebuild();
    }

    function handleClick(e: maplibregl.MapMouseEvent) {
      const coord: Coord = [e.lngLat.lng, e.lngLat.lat];

      if (mode === "line") {
        if (finalizedRef.current) {
          // After finalization, start a new line
          finalizedRef.current = false;
          pointsRef.current = [coord];
        } else {
          pointsRef.current = [...pointsRef.current, coord];
        }
        rebuild();
      } else if (mode === "circle") {
        if (pointsRef.current.length === 0) {
          // Set center
          pointsRef.current = [coord];
          rebuild();
        } else if (pointsRef.current.length === 1) {
          // Set edge — complete the circle
          pointsRef.current = [...pointsRef.current, coord];
          rebuild();
          completeAndReset();
        }
      } else {
        // rectangle — 3-click: A (start), B (orientation), C (depth)
        if (pointsRef.current.length === 0) {
          pointsRef.current = [coord];
          rebuild();
        } else if (pointsRef.current.length === 1) {
          // Second click — defines edge direction + width
          pointsRef.current = [...pointsRef.current, coord];
          rebuild();
        } else if (pointsRef.current.length === 2) {
          // Third click — defines depth, completes the rectangle
          pointsRef.current = [...pointsRef.current, coord];
          rebuild();
          completeAndReset();
        }
      }
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
      // Remove the duplicate point from the second click of dblclick
      if (pointsRef.current.length > 1) {
        pointsRef.current = pointsRef.current.slice(0, -1);
      }
      rebuild();
      completeAndReset();
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

  // Re-render preview features when style changes so the in-progress shape
  // picks up the new colours. Completed shapes keep their original per-feature style.
  useEffect(() => {
    if (!active) return;
    rebuildRef.current?.();
  }, [active, style.stroke, style.fill, style.strokeWidth]);

  // Re-render completed features when the list changes (e.g., shape removed)
  // Uses rebuild() via ref so in-progress preview data is preserved.
  useEffect(() => {
    if (!active) return;
    rebuildRef.current?.();
  }, [active, completedFeatures]);

  return null;
}

"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import type { DrawingResponse } from "@/types/api";

// ---------------------------------------------------------------------------
// Per-drawing source/layer naming
// ---------------------------------------------------------------------------

function sourceId(drawingId: string) {
  return `drawing-${drawingId}`;
}
function fillLayerId(drawingId: string) {
  return `drawing-${drawingId}-fill`;
}
function outlineLayerId(drawingId: string) {
  return `drawing-${drawingId}-outline`;
}
function lineLayerId(drawingId: string) {
  return `drawing-${drawingId}-line`;
}
function pointLayerId(drawingId: string) {
  return `drawing-${drawingId}-point`;
}

function allLayerIds(drawingId: string) {
  return [
    fillLayerId(drawingId),
    outlineLayerId(drawingId),
    lineLayerId(drawingId),
    pointLayerId(drawingId),
  ];
}

// ---------------------------------------------------------------------------
// Add / remove a single drawing
// ---------------------------------------------------------------------------

function addDrawing(map: maplibregl.Map, drawing: DrawingResponse) {
  const sid = sourceId(drawing.id);

  // Skip if already added
  if (map.getSource(sid)) return;

  map.addSource(sid, {
    type: "geojson",
    data: drawing.geojson,
  });

  // Fill layer for polygons (circles, rectangles)
  map.addLayer({
    id: fillLayerId(drawing.id),
    type: "fill",
    source: sid,
    filter: ["==", "$type", "Polygon"],
    paint: {
      // Use data-driven styling from feature properties
      "fill-color": ["coalesce", ["get", "fill"], "#3b82f6"],
      "fill-opacity": [
        "case",
        ["==", ["get", "fill"], "transparent"],
        0,
        0.2,
      ],
    },
  });

  // Outline for polygons
  map.addLayer({
    id: outlineLayerId(drawing.id),
    type: "line",
    source: sid,
    filter: ["==", "$type", "Polygon"],
    paint: {
      "line-color": ["coalesce", ["get", "stroke"], "#3b82f6"],
      "line-width": ["coalesce", ["get", "strokeWidth"], 2],
      "line-opacity": 0.9,
    },
  });

  // Line layer for line features
  map.addLayer({
    id: lineLayerId(drawing.id),
    type: "line",
    source: sid,
    filter: ["==", "$type", "LineString"],
    paint: {
      "line-color": ["coalesce", ["get", "stroke"], "#3b82f6"],
      "line-width": ["coalesce", ["get", "strokeWidth"], 2],
      "line-opacity": 0.9,
    },
  });

  // Point markers (for vertex points if stored)
  map.addLayer({
    id: pointLayerId(drawing.id),
    type: "circle",
    source: sid,
    filter: ["==", "$type", "Point"],
    paint: {
      "circle-radius": 4,
      "circle-color": ["coalesce", ["get", "stroke"], "#3b82f6"],
      "circle-stroke-color": "#ffffff",
      "circle-stroke-width": 1.5,
    },
  });
}

function removeDrawing(map: maplibregl.Map, drawingId: string) {
  try {
    for (const lid of allLayerIds(drawingId)) {
      if (map.getLayer(lid)) map.removeLayer(lid);
    }
    const sid = sourceId(drawingId);
    if (map.getSource(sid)) map.removeSource(sid);
  } catch {
    /* map may be destroyed */
  }
}

function updateDrawingData(map: maplibregl.Map, drawing: DrawingResponse) {
  try {
    const sid = sourceId(drawing.id);
    const src = map.getSource(sid);
    if (src && "setData" in src) {
      (src as maplibregl.GeoJSONSource).setData(drawing.geojson);
    }
  } catch {
    /* source may not exist */
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface DrawingOverlayProps {
  map: maplibregl.Map;
  /** Drawings to render on the map. Only visible (toggled-on) drawings. */
  drawings: DrawingResponse[];
}

/**
 * DrawingOverlay is a renderless component that manages multiple drawing
 * overlays on the map. Each drawing gets its own GeoJSON source and layer
 * set, using per-feature data-driven styling from properties.
 */
export function DrawingOverlay({ map, drawings }: DrawingOverlayProps) {
  // Track which drawing IDs are currently rendered
  const renderedRef = useRef<Map<string, string>>(new Map()); // id → updated_at

  useEffect(() => {
    const currentIds = new Set(drawings.map((d) => d.id));
    const rendered = renderedRef.current;

    // Remove drawings that are no longer in the list
    for (const [id] of rendered) {
      if (!currentIds.has(id)) {
        removeDrawing(map, id);
        rendered.delete(id);
      }
    }

    // Add or update drawings
    for (const drawing of drawings) {
      const prevUpdatedAt = rendered.get(drawing.id);
      if (prevUpdatedAt == null) {
        // New drawing — add it
        addDrawing(map, drawing);
        rendered.set(drawing.id, drawing.updated_at);
      } else if (prevUpdatedAt !== drawing.updated_at) {
        // Existing drawing but updated — refresh data
        updateDrawingData(map, drawing);
        rendered.set(drawing.id, drawing.updated_at);
      }
      // else: unchanged, do nothing
    }
  }, [map, drawings]);

  // Cleanup all drawings on unmount
  useEffect(() => {
    return () => {
      const rendered = renderedRef.current;
      for (const [id] of rendered) {
        removeDrawing(map, id);
      }
      rendered.clear();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return null;
}

"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import type { MessageResponse } from "@/types/api";

interface GpxOverlayProps {
  map: maplibregl.Map;
  message: MessageResponse | null;
}

const SOURCE_ID = "gpx-geojson";
const LINE_LAYER_ID = "gpx-lines";
const POINT_LAYER_ID = "gpx-points";

/**
 * GpxOverlay renders GeoJSON from a message's metadata as a map overlay.
 * Renders LineStrings (tracks/routes) as colored lines and Points (waypoints)
 * as circle markers.
 */
export function GpxOverlay({ map, message }: GpxOverlayProps) {
  const addedRef = useRef(false);

  useEffect(() => {
    if (!map || !message?.metadata) return;

    const geojson = message.metadata as GeoJSON.FeatureCollection;
    if (!geojson || geojson.type !== "FeatureCollection") return;

    // Clean up any previous GPX overlay
    cleanup(map);

    // Add the GeoJSON source
    map.addSource(SOURCE_ID, {
      type: "geojson",
      data: geojson,
    });

    // Add line layer for tracks and routes
    map.addLayer({
      id: LINE_LAYER_ID,
      type: "line",
      source: SOURCE_ID,
      filter: ["==", "$type", "LineString"],
      paint: {
        "line-color": "#f97316", // orange
        "line-width": 3,
        "line-opacity": 0.9,
      },
    });

    // Add circle layer for waypoints
    map.addLayer({
      id: POINT_LAYER_ID,
      type: "circle",
      source: SOURCE_ID,
      filter: ["==", "$type", "Point"],
      paint: {
        "circle-radius": 6,
        "circle-color": "#f97316",
        "circle-stroke-color": "#ffffff",
        "circle-stroke-width": 2,
      },
    });

    addedRef.current = true;

    // Fit the map to the GPX bounds
    const bounds = new maplibregl.LngLatBounds();
    let hasCoords = false;

    for (const feature of geojson.features) {
      if (feature.geometry.type === "Point") {
        const [lng, lat] = feature.geometry.coordinates as [number, number];
        bounds.extend([lng, lat]);
        hasCoords = true;
      } else if (feature.geometry.type === "LineString") {
        for (const coord of feature.geometry.coordinates as [
          number,
          number,
        ][]) {
          bounds.extend([coord[0], coord[1]]);
          hasCoords = true;
        }
      }
    }

    if (hasCoords) {
      map.fitBounds(bounds, { padding: 60, maxZoom: 16 });
    }

    return () => {
      cleanup(map);
      addedRef.current = false;
    };
  }, [map, message]);

  return null;
}

function cleanup(map: maplibregl.Map) {
  try {
    if (map.getLayer(LINE_LAYER_ID)) map.removeLayer(LINE_LAYER_ID);
    if (map.getLayer(POINT_LAYER_ID)) map.removeLayer(POINT_LAYER_ID);
    if (map.getSource(SOURCE_ID)) map.removeSource(SOURCE_ID);
  } catch {
    // Ignore errors during cleanup
  }
}

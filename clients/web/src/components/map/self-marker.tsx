"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";

interface SelfMarkerProps {
  map: maplibregl.Map;
  position: { lat: number; lng: number; heading: number | null } | null;
  /** If true, fly to the user's position on first fix (once). */
  autoCenter?: boolean;
}

const SOURCE_ID = "self-location";
const LAYER_DOT = "self-dot";
const LAYER_PULSE = "self-pulse";

const PULSE_DURATION = 2000;
const PULSE_RADIUS_MIN = 10;
const PULSE_RADIUS_MAX = 25;
const PULSE_OPACITY_MAX = 0.25;

/**
 * SelfMarker renders a blue pulsing dot for the current user's own position.
 * It uses a GeoJSON source + circle layers so the marker is rendered on the
 * WebGL canvas — keeping it perfectly in sync with the map during zoom/pan.
 */
export function SelfMarker({ map, position, autoCenter = true }: SelfMarkerProps) {
  const addedRef = useRef(false);
  const hasCenteredRef = useRef(false);
  const animFrameRef = useRef(0);
  const popupRef = useRef<maplibregl.Popup | null>(null);

  // Add source, layers, event handlers, and start pulse animation
  useEffect(() => {
    if (!position || addedRef.current) return;

    const data = pointFeature(position.lng, position.lat);

    map.addSource(SOURCE_ID, { type: "geojson", data });

    // Pulse ring (rendered below the dot)
    map.addLayer({
      id: LAYER_PULSE,
      type: "circle",
      source: SOURCE_ID,
      paint: {
        "circle-radius": PULSE_RADIUS_MIN,
        "circle-color": "#3b82f6",
        "circle-opacity": PULSE_OPACITY_MAX,
      },
    });

    // Solid dot
    map.addLayer({
      id: LAYER_DOT,
      type: "circle",
      source: SOURCE_ID,
      paint: {
        "circle-radius": 6,
        "circle-color": "#3b82f6",
        "circle-stroke-color": "#ffffff",
        "circle-stroke-width": 2,
      },
    });

    // Click → popup
    map.on("click", LAYER_DOT, (e) => {
      const coords = e.lngLat;
      // Close any existing popup
      popupRef.current?.remove();
      popupRef.current = new maplibregl.Popup({
        offset: 12,
        closeButton: false,
        maxWidth: "180px",
      })
        .setLngLat(coords)
        .setHTML(
          `<div style="font-weight:600;color:#3b82f6;margin-bottom:2px">You</div>` +
            `<div style="font-size:12px">${coords.lat.toFixed(5)}, ${coords.lng.toFixed(5)}</div>`,
        )
        .addTo(map);
    });

    // Cursor affordance
    map.on("mouseenter", LAYER_DOT, () => {
      map.getCanvas().style.cursor = "pointer";
    });
    map.on("mouseleave", LAYER_DOT, () => {
      map.getCanvas().style.cursor = "";
    });

    // Pulse animation loop
    const start = performance.now();
    const animate = () => {
      // Guard: stop if the map has been destroyed (style removed by map.remove())
      if (!(map as any).style) return;
      const t = ((performance.now() - start) % PULSE_DURATION) / PULSE_DURATION;
      const radius = PULSE_RADIUS_MIN + t * (PULSE_RADIUS_MAX - PULSE_RADIUS_MIN);
      const opacity = PULSE_OPACITY_MAX * (1 - t);
      map.setPaintProperty(LAYER_PULSE, "circle-radius", radius);
      map.setPaintProperty(LAYER_PULSE, "circle-opacity", opacity);
      animFrameRef.current = requestAnimationFrame(animate);
    };
    animFrameRef.current = requestAnimationFrame(animate);

    addedRef.current = true;
  }, [map, position]);

  // Update position whenever it changes
  useEffect(() => {
    if (!position || !addedRef.current) return;

    const source = map.getSource(SOURCE_ID) as maplibregl.GeoJSONSource | undefined;
    source?.setData(pointFeature(position.lng, position.lat));

    // Auto-center on first fix
    if (autoCenter && !hasCenteredRef.current) {
      hasCenteredRef.current = true;
      map.flyTo({
        center: [position.lng, position.lat],
        zoom: Math.max(map.getZoom(), 14),
        duration: 1500,
      });
    }
  }, [map, position, autoCenter]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      cancelAnimationFrame(animFrameRef.current);
      popupRef.current?.remove();
      try {
        if (map.getLayer(LAYER_DOT)) map.removeLayer(LAYER_DOT);
        if (map.getLayer(LAYER_PULSE)) map.removeLayer(LAYER_PULSE);
        if (map.getSource(SOURCE_ID)) map.removeSource(SOURCE_ID);
      } catch {
        // Map already destroyed during navigation
      }
    };
  }, [map]);

  return null;
}

function pointFeature(lng: number, lat: number): GeoJSON.Feature<GeoJSON.Point> {
  return {
    type: "Feature",
    geometry: { type: "Point", coordinates: [lng, lat] },
    properties: {},
  };
}

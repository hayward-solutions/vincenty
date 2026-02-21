"use client";

import { useEffect, useRef, useCallback, useState } from "react";
import maplibregl from "maplibre-gl";
import type { MapSettings } from "@/types/api";

interface MapViewProps {
  settings: MapSettings;
  onMapReady?: (map: maplibregl.Map) => void;
  children?: React.ReactNode;
}

/**
 * MapView is the core full-viewport MapLibre GL JS map component.
 * It initializes the map from server-provided settings and exposes the
 * map instance via onMapReady callback.
 */
export function MapView({ settings, onMapReady, children }: MapViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    if (!containerRef.current || mapRef.current) return;

    // Build style: if a full style JSON is provided, use it directly.
    // Otherwise, build a raster tile style from the tile URL.
    let style: maplibregl.StyleSpecification;

    if (settings.style_json) {
      style = settings.style_json as unknown as maplibregl.StyleSpecification;
    } else {
      style = {
        version: 8,
        projection: { type: "globe" },
        sky: {
          "atmosphere-blend": [
            "interpolate",
            ["linear"],
            ["zoom"],
            0, 1,
            5, 1,
            7, 0,
          ],
        },
        sources: {
          "raster-tiles": {
            type: "raster",
            tiles: [settings.tile_url],
            tileSize: 256,
            maxzoom: settings.max_zoom,
            attribution:
              '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
          },
        },
        layers: [
          {
            id: "raster-layer",
            type: "raster",
            source: "raster-tiles",
            minzoom: settings.min_zoom,
          },
        ],
      };
    }

    const map = new maplibregl.Map({
      container: containerRef.current,
      style,
      center: [settings.center_lng, settings.center_lat],
      zoom: settings.zoom,
      minZoom: settings.min_zoom,
      maxZoom: settings.max_zoom,
    });

    map.addControl(new maplibregl.NavigationControl({ visualizePitch: true }), "top-right");
    map.addControl(new maplibregl.GlobeControl(), "top-right");
    map.addControl(
      new maplibregl.GeolocateControl({
        positionOptions: { enableHighAccuracy: true },
        trackUserLocation: true,
      }),
      "top-right"
    );

    map.on("load", () => {
      mapRef.current = map;
      setIsReady(true);
      onMapReady?.(map);
    });

    return () => {
      map.remove();
      mapRef.current = null;
      setIsReady(false);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings.tile_url, settings.center_lat, settings.center_lng, settings.zoom, settings.min_zoom, settings.max_zoom, settings.style_json, onMapReady]);

  return (
    <div className="relative h-full w-full">
      <div ref={containerRef} className="h-full w-full" style={{ background: "#000" }} />
      {isReady && children}
    </div>
  );
}

/**
 * Hook to access the map instance from child components.
 * Use the onMapReady callback pattern instead for most cases.
 */
export function useMapInstance() {
  const mapRef = useRef<maplibregl.Map | null>(null);

  const setMap = useCallback((map: maplibregl.Map) => {
    mapRef.current = map;
  }, []);

  return { map: mapRef, setMap };
}

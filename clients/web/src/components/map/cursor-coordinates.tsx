"use client";

import { useEffect, useState } from "react";
import maplibregl from "maplibre-gl";

interface CursorCoordinatesProps {
  map: maplibregl.Map;
}

/**
 * Displays the cursor's geographic coordinates in the bottom-left corner
 * of the map. Coordinates are shown in decimal degrees and update on
 * mousemove. The overlay hides when the cursor leaves the map canvas.
 */
export function CursorCoordinates({ map }: CursorCoordinatesProps) {
  const [coords, setCoords] = useState<{ lng: number; lat: number } | null>(
    null
  );

  useEffect(() => {
    const handleMouseMove = (e: maplibregl.MapMouseEvent) => {
      setCoords({ lng: e.lngLat.lng, lat: e.lngLat.lat });
    };

    const handleMouseOut = () => {
      setCoords(null);
    };

    map.on("mousemove", handleMouseMove);
    map.getCanvas().addEventListener("mouseout", handleMouseOut);

    return () => {
      map.off("mousemove", handleMouseMove);
      map.getCanvas().removeEventListener("mouseout", handleMouseOut);
    };
  }, [map]);

  if (!coords) return null;

  return (
    <div className="absolute bottom-3 left-3 z-10 select-none bg-card/90 backdrop-blur-sm border rounded-lg px-2.5 py-1.5 shadow-lg">
      <span className="text-xs font-mono text-muted-foreground">
        {coords.lat.toFixed(6)}, {coords.lng.toFixed(6)}
      </span>
    </div>
  );
}

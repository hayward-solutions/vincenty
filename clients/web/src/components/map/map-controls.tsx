"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import type maplibregl from "maplibre-gl";
import { Plus, Minus, Compass, Globe, Mountain } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Separator } from "@/components/ui/separator";

interface MapControlsProps {
  map: maplibregl.Map;
  terrainAvailable?: boolean;
}

/**
 * Custom map navigation controls replacing MapLibre's built-in
 * NavigationControl and GlobeControl with buttons matching our design system.
 */
export function MapControls({ map, terrainAvailable }: MapControlsProps) {
  // Track whether the component is still mounted so map event handlers
  // don't trigger React state updates after the map has been destroyed.
  // Without this guard, navigating away from the map page causes a cascade:
  // map.remove() fires final rotate/pitch events -> setState -> re-render ->
  // sibling components call methods on the destroyed map -> crash.
  const mountedRef = useRef(true);
  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const [bearing, setBearing] = useState(() => map.getBearing());
  const [pitch, setPitch] = useState(() => map.getPitch());
  const [isGlobe, setIsGlobe] = useState(() => {
    try {
      const projection = map.getProjection();
      return projection?.type === "globe";
    } catch {
      return false;
    }
  });
  const [isTerrain, setIsTerrain] = useState(false);

  useEffect(() => {
    const onRotate = () => {
      if (!mountedRef.current) return;
      try { setBearing(map.getBearing()); } catch { /* map destroyed */ }
    };
    const onPitch = () => {
      if (!mountedRef.current) return;
      try { setPitch(map.getPitch()); } catch { /* map destroyed */ }
    };

    map.on("rotate", onRotate);
    map.on("pitch", onPitch);

    return () => {
      map.off("rotate", onRotate);
      map.off("pitch", onPitch);
    };
  }, [map]);

  const handleZoomIn = useCallback(() => {
    try { map.zoomIn(); } catch { /* map destroyed */ }
  }, [map]);
  const handleZoomOut = useCallback(() => {
    try { map.zoomOut(); } catch { /* map destroyed */ }
  }, [map]);

  const handleResetNorth = useCallback(() => {
    try { map.easeTo({ bearing: 0, pitch: 0 }); } catch { /* map destroyed */ }
  }, [map]);

  const handleToggleGlobe = useCallback(() => {
    try {
      if (!isGlobe) {
        // Switching to globe — disable terrain first (incompatible)
        if (isTerrain) {
          map.setTerrain(null);
          setIsTerrain(false);
        }
        map.setProjection({ type: "globe" } as maplibregl.ProjectionSpecification);
        setIsGlobe(true);
      } else {
        map.setProjection({ type: "mercator" } as maplibregl.ProjectionSpecification);
        setIsGlobe(false);
      }
    } catch { /* map destroyed */ }
  }, [map, isGlobe, isTerrain]);

  const handleToggleTerrain = useCallback(() => {
    try {
      if (isTerrain) {
        map.setTerrain(null);
        setIsTerrain(false);
      } else {
        // Enabling terrain — switch away from globe first (incompatible)
        if (isGlobe) {
          map.setProjection({ type: "mercator" } as maplibregl.ProjectionSpecification);
          setIsGlobe(false);
        }
        map.setTerrain({ source: "terrain-dem", exaggeration: 1.0 });
        setIsTerrain(true);
      }
    } catch { /* map destroyed */ }
  }, [map, isTerrain, isGlobe]);

  const isRotated = bearing !== 0 || pitch !== 0;

  return (
    <TooltipProvider>
      <div className="absolute top-3 right-3 z-10 flex flex-col bg-card/90 backdrop-blur-sm border rounded-lg shadow-lg overflow-hidden">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={handleZoomIn}
              aria-label="Zoom in"
            >
              <Plus className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">Zoom in</TooltipContent>
        </Tooltip>

        <Separator />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={handleZoomOut}
              aria-label="Zoom out"
            >
              <Minus className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">Zoom out</TooltipContent>
        </Tooltip>

        <Separator />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={handleResetNorth}
              aria-label="Reset north"
              className={isRotated ? "text-foreground" : "text-muted-foreground"}
            >
              <Compass
                className="size-4 transition-transform duration-200"
                style={{ transform: `rotate(${-bearing}deg)` }}
              />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">Reset north</TooltipContent>
        </Tooltip>

        <Separator />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={handleToggleGlobe}
              aria-label="Toggle globe"
              className={isGlobe ? "text-foreground" : "text-muted-foreground"}
            >
              <Globe className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">
            {isGlobe ? "Switch to flat map" : "Switch to globe"}
          </TooltipContent>
        </Tooltip>

        {terrainAvailable && (
          <>
            <Separator />

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={handleToggleTerrain}
                  aria-label="Toggle terrain"
                  className={isTerrain ? "text-foreground" : "text-muted-foreground"}
                >
                  <Mountain className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="left">
                {isTerrain ? "Disable 3D terrain" : "Enable 3D terrain"}
              </TooltipContent>
            </Tooltip>
          </>
        )}
      </div>
    </TooltipProvider>
  );
}

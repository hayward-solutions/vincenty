"use client";

import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { formatDistance, formatArea, type MeasureResult } from "./measure-tool";

interface MeasurePanelProps {
  mode: "line" | "circle";
  onModeChange: (mode: "line" | "circle") => void;
  measurements: MeasureResult;
  onClear: () => void;
  onClose: () => void;
}

/**
 * Panel displayed below the toolbar when the Measure tool is active.
 * Shows mode selector, live measurement results, and clear/close actions.
 */
export function MeasurePanel({
  mode,
  onModeChange,
  measurements,
  onClear,
  onClose,
}: MeasurePanelProps) {
  const hasSegments = measurements.segments.length > 0;
  const hasRadius =
    measurements.radius != null && measurements.radius > 0;

  return (
    <div className="bg-card/95 backdrop-blur-sm border rounded-lg p-3 shadow-lg sm:w-64 space-y-3">
      <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
        Measure
      </h4>

      {/* Mode selector */}
      <div className="space-y-1.5">
        <p className="text-xs font-medium">Mode</p>
        <div className="grid grid-cols-2 gap-1.5">
          <button
            onClick={() => onModeChange("line")}
            className={`h-7 rounded-md text-xs font-medium transition-colors ${
              mode === "line"
                ? "bg-primary text-primary-foreground"
                : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
            }`}
          >
            Distance
          </button>
          <button
            onClick={() => onModeChange("circle")}
            className={`h-7 rounded-md text-xs font-medium transition-colors ${
              mode === "circle"
                ? "bg-primary text-primary-foreground"
                : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
            }`}
          >
            Radius
          </button>
        </div>
      </div>

      {/* Instructions / Results */}
      {mode === "line" && !hasSegments && (
        <p className="text-xs text-muted-foreground">
          Click on the map to place points. Double-click to finish.
        </p>
      )}

      {mode === "circle" && !hasRadius && (
        <p className="text-xs text-muted-foreground">
          Click to place the centre, then click again to set the radius.
        </p>
      )}

      {/* Line mode: total distance */}
      {mode === "line" && hasSegments && (
        <div className="flex items-center justify-between text-xs">
          <span className="font-medium">Total</span>
          <span className="font-medium tabular-nums">
            {formatDistance(measurements.total)}
          </span>
        </div>
      )}

      {/* Circle mode: radius + area */}
      {mode === "circle" && hasRadius && (
        <div className="space-y-1">
          <div className="flex items-center justify-between text-xs">
            <span className="text-muted-foreground">Radius</span>
            <span className="font-medium tabular-nums">
              {formatDistance(measurements.radius!)}
            </span>
          </div>
          {measurements.area != null && measurements.area > 0 && (
            <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground">Area</span>
              <span className="font-medium tabular-nums">
                {formatArea(measurements.area)}
              </span>
            </div>
          )}
        </div>
      )}

      <Separator />

      {/* Actions */}
      <div className="flex gap-2">
        <Button size="sm" variant="outline" onClick={onClear} className="flex-1">
          Clear
        </Button>
        <Button size="sm" variant="ghost" onClick={onClose}>
          Close
        </Button>
      </div>
    </div>
  );
}

"use client";

import { useCallback, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";

export type ReplayScope = "all" | "group" | "user";

export interface ReplayStartParams {
  from: Date;
  to: Date;
}

type TimePreset = "1h" | "6h" | "24h" | "custom";

interface ReplayPanelProps {
  isLoading: boolean;
  onStart: (params: ReplayStartParams) => void;
  onExportGPX: (from: Date, to: Date) => void;
  onCancel: () => void;
}

/**
 * Replay setup panel — time range picker only.
 * Group/user filtering is handled by the shared FilterPanel.
 * Styled to match the FilterPanel (glass-card dropdown below toolbar).
 */
export function ReplayPanel({
  isLoading,
  onStart,
  onExportGPX,
  onCancel,
}: ReplayPanelProps) {
  const [preset, setPreset] = useState<TimePreset>("1h");

  // Custom range inputs (only used when preset === "custom")
  const [customFrom, setCustomFrom] = useState(() => {
    const d = new Date();
    d.setHours(d.getHours() - 1);
    return d.toISOString().slice(0, 16);
  });
  const [customTo, setCustomTo] = useState(() =>
    new Date().toISOString().slice(0, 16)
  );

  /** Resolve the selected time range to concrete Date objects. */
  const resolveRange = useCallback((): {
    from: Date;
    to: Date;
  } | null => {
    if (preset === "custom") {
      const from = new Date(customFrom);
      const to = new Date(customTo);
      if (to <= from) {
        toast.error("End time must be after start time");
        return null;
      }
      if (to.getTime() - from.getTime() > 24 * 60 * 60 * 1000) {
        toast.error("Time range must not exceed 24 hours");
        return null;
      }
      return { from, to };
    }

    const to = new Date();
    const from = new Date(to);
    switch (preset) {
      case "1h":
        from.setHours(from.getHours() - 1);
        break;
      case "6h":
        from.setHours(from.getHours() - 6);
        break;
      case "24h":
        from.setDate(from.getDate() - 1);
        break;
    }
    return { from, to };
  }, [preset, customFrom, customTo]);

  const handleStart = useCallback(() => {
    const range = resolveRange();
    if (!range) return;
    onStart(range);
  }, [resolveRange, onStart]);

  const handleExport = useCallback(() => {
    const range = resolveRange();
    if (!range) return;
    onExportGPX(range.from, range.to);
  }, [resolveRange, onExportGPX]);

  const presets: { value: TimePreset; label: string }[] = [
    { value: "1h", label: "1h" },
    { value: "6h", label: "6h" },
    { value: "24h", label: "24h" },
  ];

  return (
    <div className="bg-card/95 backdrop-blur-sm border rounded-lg p-3 shadow-lg sm:w-64 space-y-3">
      <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
        Replay
      </h4>

      {/* Time range presets */}
      <div className="space-y-1.5">
        <p className="text-xs font-medium">Time Range</p>

        <div className="grid grid-cols-3 gap-1.5">
          {presets.map((p) => (
            <button
              key={p.value}
              onClick={() => setPreset(p.value)}
              className={`h-7 rounded-md text-xs font-medium transition-colors ${
                preset === p.value
                  ? "bg-primary text-primary-foreground"
                  : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>

        <button
          onClick={() => setPreset("custom")}
          className={`h-7 w-full rounded-md text-xs font-medium transition-colors ${
            preset === "custom"
              ? "bg-primary text-primary-foreground"
              : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
          }`}
        >
          Custom
        </button>
      </div>

      {/* Custom datetime inputs */}
      {preset === "custom" && (
        <div className="space-y-2">
          <div className="space-y-1">
            <p className="text-xs font-medium">From</p>
            <Input
              type="datetime-local"
              value={customFrom}
              onChange={(e) => setCustomFrom(e.target.value)}
              className="h-7 text-xs"
            />
          </div>
          <div className="space-y-1">
            <p className="text-xs font-medium">To</p>
            <Input
              type="datetime-local"
              value={customTo}
              onChange={(e) => setCustomTo(e.target.value)}
              className="h-7 text-xs"
            />
          </div>
        </div>
      )}

      <Separator />

      {/* Actions */}
      <div className="flex gap-2">
        <Button
          size="sm"
          onClick={handleStart}
          disabled={isLoading}
          className="flex-1"
        >
          {isLoading ? "Loading..." : "Start"}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={handleExport}
          className="flex-1"
        >
          Export GPX
        </Button>
        <Button size="sm" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  );
}

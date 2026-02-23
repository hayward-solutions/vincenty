"use client";

import { useState } from "react";
import {
  Minus,
  Circle,
  Square,
  Trash2,
  Share2,
  Save,
  X,
  Eye,
  EyeOff,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { DrawMode, DrawStyle, CompletedShape } from "./draw-tool";
import type { Group, DrawingResponse, DrawingShareInfo } from "@/types/api";

// ---------------------------------------------------------------------------
// Color presets
// ---------------------------------------------------------------------------

const STROKE_COLORS = [
  "#ef4444", // red
  "#f97316", // orange
  "#eab308", // yellow
  "#22c55e", // green
  "#06b6d4", // cyan
  "#3b82f6", // blue
  "#8b5cf6", // violet
  "#ec4899", // pink
  "#ffffff", // white
  "#000000", // black
];

const FILL_COLORS = [
  "transparent",
  "#ef4444",
  "#f97316",
  "#eab308",
  "#22c55e",
  "#06b6d4",
  "#3b82f6",
  "#8b5cf6",
  "#ec4899",
  "#000000",
];

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface ShareTarget {
  type: "group" | "user";
  id: string;
  name: string;
}

interface DrawPanelProps {
  mode: DrawMode;
  onModeChange: (mode: DrawMode) => void;
  style: DrawStyle;
  onStyleChange: (style: DrawStyle) => void;
  shapes: CompletedShape[];
  onRemoveShape: (index: number) => void;
  onClearShapes: () => void;
  drawingName: string;
  onDrawingNameChange: (name: string) => void;
  onSave: () => void;
  isSaving: boolean;
  /** Available groups for sharing. */
  groups: Group[];
  /** Called when the user shares to a target after saving. */
  onShare: (target: ShareTarget) => void;
  isSharing: boolean;
  /** ID of a saved drawing (set after first save). */
  savedDrawingId: string | null;
  onClose: () => void;

  /** Saved drawings — own + shared with visibility & management. */
  ownDrawings: DrawingResponse[];
  sharedDrawings: DrawingResponse[];
  visibleDrawingIds: Set<string>;
  onDrawingToggle: (drawingId: string) => void;

  /** Drawing management (own drawings only). */
  onDrawingDelete: (drawingId: string) => void;
  onDrawingShare: (drawingId: string, groupId: string, groupName: string) => void;
  onDrawingUnshare: (drawingId: string, messageId: string) => void;
  /** Shares for the currently expanded drawing (null = none expanded). */
  managingDrawingId: string | null;
  onManagingDrawingChange: (drawingId: string | null) => void;
  drawingShares: DrawingShareInfo[];
  drawingSharesLoading: boolean;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export type { ShareTarget };

export function DrawPanel({
  mode,
  onModeChange,
  style,
  onStyleChange,
  shapes,
  onRemoveShape,
  onClearShapes,
  drawingName,
  onDrawingNameChange,
  onSave,
  isSaving,
  groups,
  onShare,
  isSharing,
  savedDrawingId,
  onClose,
  ownDrawings,
  sharedDrawings,
  visibleDrawingIds,
  onDrawingToggle,
  onDrawingDelete,
  onDrawingShare,
  onDrawingUnshare,
  managingDrawingId,
  onManagingDrawingChange,
  drawingShares,
  drawingSharesLoading,
}: DrawPanelProps) {
  const [showShareList, setShowShareList] = useState(false);

  return (
    <div className="bg-card/95 backdrop-blur-sm border rounded-lg p-3 shadow-lg sm:w-72 space-y-3 max-h-[70vh] overflow-y-auto">
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          Draw
        </h4>
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={onClose}
          className="text-muted-foreground h-5 w-5"
        >
          <X className="size-3.5" />
        </Button>
      </div>

      {/* Mode selector */}
      <div className="space-y-1.5">
        <p className="text-xs font-medium">Tool</p>
        <div className="grid grid-cols-3 gap-1.5">
          <button
            onClick={() => onModeChange("line")}
            className={`h-7 rounded-md text-xs font-medium transition-colors flex items-center justify-center gap-1 ${
              mode === "line"
                ? "bg-primary text-primary-foreground"
                : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
            }`}
          >
            <Minus className="size-3" />
            Line
          </button>
          <button
            onClick={() => onModeChange("circle")}
            className={`h-7 rounded-md text-xs font-medium transition-colors flex items-center justify-center gap-1 ${
              mode === "circle"
                ? "bg-primary text-primary-foreground"
                : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
            }`}
          >
            <Circle className="size-3" />
            Circle
          </button>
          <button
            onClick={() => onModeChange("rectangle")}
            className={`h-7 rounded-md text-xs font-medium transition-colors flex items-center justify-center gap-1 ${
              mode === "rectangle"
                ? "bg-primary text-primary-foreground"
                : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
            }`}
          >
            <Square className="size-3" />
            Rect
          </button>
        </div>
      </div>

      {/* Stroke color */}
      <div className="space-y-1.5">
        <p className="text-xs font-medium">Stroke</p>
        <div className="flex flex-wrap gap-1">
          {STROKE_COLORS.map((color) => (
            <button
              key={`stroke-${color}`}
              onClick={() => onStyleChange({ ...style, stroke: color })}
              className={`w-5 h-5 rounded-full border-2 transition-transform ${
                style.stroke === color
                  ? "border-foreground scale-125"
                  : "border-muted hover:scale-110"
              }`}
              style={{ backgroundColor: color }}
              title={color}
            />
          ))}
        </div>
      </div>

      {/* Fill color */}
      <div className="space-y-1.5">
        <p className="text-xs font-medium">Fill</p>
        <div className="flex flex-wrap gap-1">
          {FILL_COLORS.map((color) => (
            <button
              key={`fill-${color}`}
              onClick={() => onStyleChange({ ...style, fill: color })}
              className={`w-5 h-5 rounded-full border-2 transition-transform ${
                style.fill === color
                  ? "border-foreground scale-125"
                  : "border-muted hover:scale-110"
              } ${color === "transparent" ? "bg-background" : ""}`}
              style={
                color === "transparent"
                  ? {
                      backgroundImage:
                        "linear-gradient(45deg, #ccc 25%, transparent 25%, transparent 75%, #ccc 75%), linear-gradient(45deg, #ccc 25%, transparent 25%, transparent 75%, #ccc 75%)",
                      backgroundSize: "6px 6px",
                      backgroundPosition: "0 0, 3px 3px",
                    }
                  : { backgroundColor: color }
              }
              title={color === "transparent" ? "No fill" : color}
            />
          ))}
        </div>
      </div>

      {/* Instructions */}
      {shapes.length === 0 && (
        <p className="text-xs text-muted-foreground">
          {mode === "line"
            ? "Click to place points. Double-click to finish the line."
            : mode === "circle"
            ? "Click to place centre, then click to set radius."
            : "Click for first corner, then click for opposite corner."}
        </p>
      )}

      {/* Shape list */}
      {shapes.length > 0 && (
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <p className="text-xs font-medium">
              Shapes ({shapes.length})
            </p>
            <Button
              variant="ghost"
              size="sm"
              onClick={onClearShapes}
              className="h-5 px-1.5 text-xs text-muted-foreground"
            >
              Clear all
            </Button>
          </div>
          <div className="space-y-1 max-h-24 overflow-y-auto">
            {shapes.map((shape, i) => {
              const shapeType =
                (shape.feature.properties?.shapeType as string) ?? "shape";
              return (
                <div
                  key={i}
                  className="flex items-center justify-between text-xs bg-secondary/50 rounded px-2 py-1"
                >
                  <div className="flex items-center gap-1.5">
                    <span
                      className="w-2.5 h-2.5 rounded-full"
                      style={{
                        backgroundColor:
                          (shape.feature.properties?.stroke as string) ?? "#fff",
                      }}
                    />
                    <span className="capitalize">{shapeType}</span>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => onRemoveShape(i)}
                    className="h-4 w-4 text-muted-foreground"
                  >
                    <Trash2 className="size-3" />
                  </Button>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Name + Save */}
      {shapes.length > 0 && (
        <>
          <Separator />
          <div className="space-y-2">
            <Input
              value={drawingName}
              onChange={(e) => onDrawingNameChange(e.target.value)}
              placeholder="Drawing name"
              className="h-7 text-xs"
            />
            <div className="flex gap-2">
              <Button
                size="sm"
                onClick={onSave}
                disabled={isSaving || !drawingName.trim()}
                className="flex-1 gap-1"
              >
                <Save className="size-3" />
                {savedDrawingId ? "Update" : "Save"}
              </Button>
              {savedDrawingId && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setShowShareList(!showShareList)}
                  disabled={isSharing}
                  className="gap-1"
                >
                  <Share2 className="size-3" />
                  Share
                </Button>
              )}
            </div>
          </div>
        </>
      )}

      {/* Share target list */}
      {showShareList && savedDrawingId && (
        <>
          <Separator />
          <div className="space-y-1.5">
            <p className="text-xs font-medium">Share to group</p>
            {groups.length === 0 && (
              <p className="text-xs text-muted-foreground">
                No groups available.
              </p>
            )}
            <div className="space-y-1 max-h-32 overflow-y-auto">
              {groups.map((group) => (
                <button
                  key={group.id}
                  onClick={() => {
                    onShare({ type: "group", id: group.id, name: group.name });
                    setShowShareList(false);
                  }}
                  disabled={isSharing}
                  className="w-full text-left text-xs px-2 py-1.5 rounded hover:bg-secondary/80 transition-colors"
                >
                  {group.name}
                </button>
              ))}
            </div>
          </div>
        </>
      )}

      {/* ----------------------------------------------------------------- */}
      {/* Saved Drawings                                                     */}
      {/* ----------------------------------------------------------------- */}
      {(ownDrawings.length > 0 || sharedDrawings.length > 0) && (
        <>
          <Separator />

          <div className="space-y-1.5">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Saved Drawings
            </h4>

            {ownDrawings.length > 0 && (
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground">Mine</p>
                {ownDrawings.map((d) => {
                  const sharedGroupIds = new Set(
                    managingDrawingId === d.id
                      ? drawingShares.filter((s) => s.type === "group").map((s) => s.id)
                      : []
                  );
                  return (
                    <div key={d.id} className="space-y-0.5">
                      <div className="flex items-center gap-1.5 text-sm">
                        <input
                          type="checkbox"
                          checked={visibleDrawingIds.has(d.id)}
                          onChange={() => onDrawingToggle(d.id)}
                          className="h-3.5 w-3.5 cursor-pointer"
                        />
                        <span className="truncate flex-1 text-xs">{d.name}</span>

                        {/* Share management popover */}
                        <Popover
                          open={managingDrawingId === d.id}
                          onOpenChange={(open) =>
                            onManagingDrawingChange(open ? d.id : null)
                          }
                        >
                          <PopoverTrigger asChild>
                            <button
                              className="text-muted-foreground hover:text-foreground flex-shrink-0"
                              title="Manage shares"
                            >
                              <Share2 className="size-3" />
                            </button>
                          </PopoverTrigger>
                          <PopoverContent
                            side="right"
                            align="start"
                            className="w-56 p-2 space-y-2"
                          >
                            <p className="text-xs font-medium">Shared with</p>
                            {drawingSharesLoading ? (
                              <p className="text-xs text-muted-foreground">Loading...</p>
                            ) : drawingShares.length === 0 ? (
                              <p className="text-xs text-muted-foreground">Not shared yet</p>
                            ) : (
                              <div className="space-y-1 max-h-28 overflow-y-auto">
                                {drawingShares.map((share) => (
                                  <div
                                    key={share.message_id}
                                    className="flex items-center justify-between text-xs px-1.5 py-1 rounded bg-secondary/50"
                                  >
                                    <span className="truncate">{share.name}</span>
                                    <button
                                      onClick={() => onDrawingUnshare(d.id, share.message_id)}
                                      className="text-muted-foreground hover:text-destructive flex-shrink-0 ml-1"
                                      title="Unshare"
                                    >
                                      <X className="size-3" />
                                    </button>
                                  </div>
                                ))}
                              </div>
                            )}

                            {/* Add share to new groups */}
                            {groups.filter((g) => !sharedGroupIds.has(g.id)).length > 0 && (
                              <>
                                <Separator />
                                <p className="text-xs font-medium">Share to</p>
                                <div className="space-y-0.5 max-h-28 overflow-y-auto">
                                  {groups
                                    .filter((g) => !sharedGroupIds.has(g.id))
                                    .map((g) => (
                                      <button
                                        key={g.id}
                                        onClick={() => onDrawingShare(d.id, g.id, g.name)}
                                        className="w-full text-left text-xs px-1.5 py-1 rounded hover:bg-secondary/80 transition-colors"
                                      >
                                        {g.name}
                                      </button>
                                    ))}
                                </div>
                              </>
                            )}
                          </PopoverContent>
                        </Popover>

                        {/* Delete */}
                        <button
                          onClick={() => onDrawingDelete(d.id)}
                          className="text-muted-foreground hover:text-destructive flex-shrink-0"
                          title="Delete drawing"
                        >
                          <Trash2 className="size-3" />
                        </button>

                        {visibleDrawingIds.has(d.id) ? (
                          <Eye className="size-3 text-muted-foreground flex-shrink-0" />
                        ) : (
                          <EyeOff className="size-3 text-muted-foreground flex-shrink-0" />
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {sharedDrawings.length > 0 && (
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground">Shared with me</p>
                {sharedDrawings.map((d) => (
                  <label
                    key={d.id}
                    className="flex items-center gap-2 text-sm cursor-pointer"
                  >
                    <input
                      type="checkbox"
                      checked={visibleDrawingIds.has(d.id)}
                      onChange={() => onDrawingToggle(d.id)}
                      className="h-3.5 w-3.5"
                    />
                    <span className="truncate flex-1 text-xs">
                      {d.name}
                      <span className="text-muted-foreground ml-1">
                        ({d.display_name || d.username})
                      </span>
                    </span>
                    {visibleDrawingIds.has(d.id) ? (
                      <Eye className="size-3 text-muted-foreground flex-shrink-0" />
                    ) : (
                      <EyeOff className="size-3 text-muted-foreground flex-shrink-0" />
                    )}
                  </label>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}

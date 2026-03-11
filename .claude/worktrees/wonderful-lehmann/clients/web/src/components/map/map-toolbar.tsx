"use client";

import { History, Filter, Ruler, PenTool } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Separator } from "@/components/ui/separator";

interface MapToolbarProps {
  onReplayClick: () => void;
  replayActive: boolean;
  filterActive: boolean;
  onFilterClick: () => void;
  measureActive: boolean;
  onMeasureClick: () => void;
  drawActive: boolean;
  onDrawClick: () => void;
}

/**
 * Horizontal toolbar displayed top-left on the map.
 * Mirrors the styling of MapControls (right-side nav bar) but laid out
 * horizontally with icon-only buttons and bottom-facing tooltips.
 */
export function MapToolbar({
  onReplayClick,
  replayActive,
  filterActive,
  onFilterClick,
  measureActive,
  onMeasureClick,
  drawActive,
  onDrawClick,
}: MapToolbarProps) {
  return (
    <TooltipProvider>
      <div className="flex flex-row bg-card/90 backdrop-blur-sm border rounded-lg shadow-lg overflow-hidden">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={onReplayClick}
              aria-label="Replay"
              className={
                replayActive ? "text-foreground" : "text-muted-foreground"
              }
            >
              <History className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Replay</TooltipContent>
        </Tooltip>

        <Separator orientation="vertical" />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={onFilterClick}
              aria-label="Filters"
              className={
                filterActive ? "text-foreground" : "text-muted-foreground"
              }
            >
              <Filter className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Filters</TooltipContent>
        </Tooltip>

        <Separator orientation="vertical" />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={onMeasureClick}
              aria-label="Measure"
              className={
                measureActive ? "text-foreground" : "text-muted-foreground"
              }
            >
              <Ruler className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Measure</TooltipContent>
        </Tooltip>

        <Separator orientation="vertical" />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={onDrawClick}
              aria-label="Draw"
              className={
                drawActive ? "text-foreground" : "text-muted-foreground"
              }
            >
              <PenTool className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Draw</TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  );
}

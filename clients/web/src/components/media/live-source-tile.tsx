"use client";

import { FeedViewer } from "./feed-viewer";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  Camera,
  Video,
  Mic,
  Maximize2,
  Minimize2,
  X,
  Signal,
} from "lucide-react";

interface LiveSourceTileProps {
  sourceType: "feed" | "call" | "ptt";
  name: string;
  groupName: string;
  feedType?: string;
  token: string;
  serverUrl: string;
  isExpanded?: boolean;
  onToggleExpand?: () => void;
  onRemove?: () => void;
}

const sourceIcons = {
  feed: Camera,
  call: Video,
  ptt: Mic,
};

const sourceLabels: Record<string, string> = {
  rtsp: "RTSP",
  rtmp: "RTMP",
  whip: "WHIP",
  phone_cam: "Phone",
  call: "Call",
  ptt: "PTT",
};

export function LiveSourceTile({
  sourceType,
  name,
  groupName,
  feedType,
  token,
  serverUrl,
  isExpanded,
  onToggleExpand,
  onRemove,
}: LiveSourceTileProps) {
  const Icon = sourceIcons[sourceType];
  const typeLabel = feedType ? sourceLabels[feedType] ?? feedType : sourceLabels[sourceType] ?? sourceType;

  return (
    <div
      className={cn(
        "relative rounded-lg overflow-hidden bg-gray-900 group/tile",
        isExpanded && "col-span-full"
      )}
    >
      {/* Video content */}
      <FeedViewer token={token} serverUrl={serverUrl} feedName={name} />

      {/* Top overlay — always visible */}
      <div className="absolute top-0 inset-x-0 p-2 flex items-start justify-between pointer-events-none">
        <div className="flex items-center gap-1.5">
          <Badge variant="secondary" className="bg-black/60 text-white border-0 text-[10px] pointer-events-auto">
            <Icon className="h-3 w-3 mr-1" />
            {typeLabel}
          </Badge>
          <Badge variant="secondary" className="bg-black/60 text-white border-0 text-[10px]">
            <Signal className="h-2.5 w-2.5 mr-1 text-green-400" />
            Live
          </Badge>
        </div>

        {/* Controls — visible on hover via group-hover */}
        <div className="flex items-center gap-1 transition-opacity pointer-events-auto opacity-0 group-hover/tile:opacity-100">
          {onToggleExpand && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 bg-black/60 hover:bg-black/80 text-white"
              onClick={onToggleExpand}
            >
              {isExpanded ? (
                <Minimize2 className="h-3.5 w-3.5" />
              ) : (
                <Maximize2 className="h-3.5 w-3.5" />
              )}
            </Button>
          )}
          {onRemove && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 bg-black/60 hover:bg-red-600/80 text-white"
              onClick={onRemove}
            >
              <X className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      </div>

      {/* Bottom overlay — source name and group */}
      <div className="absolute bottom-0 inset-x-0 bg-gradient-to-t from-black/80 to-transparent p-2 pt-6">
        <p className="text-white text-sm font-medium truncate">{name}</p>
        <p className="text-gray-300 text-xs truncate">{groupName}</p>
      </div>
    </div>
  );
}

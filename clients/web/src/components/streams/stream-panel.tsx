"use client";

import { useState } from "react";
import { useStreams } from "@/lib/hooks/use-streams";
import { BroadcastDialog } from "@/components/streams/broadcast-dialog";
import { StreamViewer } from "@/components/streams/stream-viewer";
import type { StreamResponse } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Video, Radio } from "lucide-react";

interface StreamPanelProps {
  className?: string;
}

export function StreamPanel({ className }: StreamPanelProps) {
  const { streams, refetch } = useStreams("live");
  const [broadcastOpen, setBroadcastOpen] = useState(false);
  const [viewingStream, setViewingStream] = useState<StreamResponse | null>(
    null
  );

  return (
    <>
      <div
        className={`flex flex-col bg-card border rounded-lg shadow-lg ${className ?? ""}`}
        style={{ width: 280 }}
      >
        <div className="flex items-center justify-between px-3 py-2 border-b">
          <div className="flex items-center gap-1.5">
            <Radio className="h-4 w-4 text-red-500" />
            <span className="text-sm font-medium">Live Streams</span>
            {streams.length > 0 && (
              <Badge variant="secondary" className="text-xs px-1.5 py-0">
                {streams.length}
              </Badge>
            )}
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs"
            onClick={() => setBroadcastOpen(true)}
          >
            <Video className="h-3.5 w-3.5 mr-1" />
            Go Live
          </Button>
        </div>

        <ScrollArea className="flex-1 max-h-80">
          {streams.length === 0 ? (
            <div className="p-4 text-center text-sm text-muted-foreground">
              No active streams
            </div>
          ) : (
            <div className="p-1.5 space-y-1">
              {streams.map((stream) => (
                <button
                  key={stream.id}
                  onClick={() => setViewingStream(stream)}
                  className="w-full text-left rounded-md px-2.5 py-2 hover:bg-accent transition-colors"
                >
                  <div className="flex items-center gap-2">
                    <Badge
                      variant="destructive"
                      className="text-[10px] px-1 py-0 animate-pulse flex-shrink-0"
                    >
                      LIVE
                    </Badge>
                    <span className="text-sm font-medium truncate">
                      {stream.title}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="text-xs text-muted-foreground truncate">
                      {stream.display_name || stream.username}
                    </span>
                    <span className="text-[10px] text-muted-foreground capitalize">
                      {stream.source_type}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          )}
        </ScrollArea>
      </div>

      {/* Broadcast dialog */}
      <BroadcastDialog
        open={broadcastOpen}
        onOpenChange={setBroadcastOpen}
        onStreamStarted={() => refetch()}
        onStreamEnded={() => refetch()}
      />

      {/* Stream viewer dialog */}
      <StreamViewerDialog
        stream={viewingStream}
        onClose={() => setViewingStream(null)}
      />
    </>
  );
}

function StreamViewerDialog({
  stream,
  onClose,
}: {
  stream: StreamResponse | null;
  onClose: () => void;
}) {
  if (!stream) return null;

  return (
    <Dialog open={!!stream} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-2xl p-0 gap-0">
        <DialogHeader className="px-4 pt-4 pb-2">
          <DialogTitle className="flex items-center gap-2">
            {stream.status === "live" && (
              <Badge variant="destructive" className="animate-pulse text-xs">
                LIVE
              </Badge>
            )}
            {stream.title}
          </DialogTitle>
        </DialogHeader>
        <div className="px-4 pb-4">
          <StreamViewer stream={stream} className="aspect-video" />
        </div>
      </DialogContent>
    </Dialog>
  );
}

export { StreamViewerDialog };

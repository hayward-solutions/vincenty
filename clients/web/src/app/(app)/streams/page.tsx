"use client";

import { useState } from "react";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { useStreams, useDeleteStream } from "@/lib/hooks/use-streams";
import { BroadcastDialog } from "@/components/streams/broadcast-dialog";
import { StreamViewer } from "@/components/streams/stream-viewer";
import type { StreamResponse } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Video,
  Radio,
  Camera,
  Monitor,
  MoreVertical,
  Trash2,
} from "lucide-react";

type Tab = "live" | "recordings";

export default function StreamsPage() {
  const [tab, setTab] = useState<Tab>("live");
  const { streams, isLoading, refetch } = useStreams(
    tab === "live" ? "live" : "ended"
  );
  const [broadcastOpen, setBroadcastOpen] = useState(false);
  const [viewingStream, setViewingStream] = useState<StreamResponse | null>(
    null
  );

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Streams</h1>
        <Button onClick={() => setBroadcastOpen(true)}>
          <Video className="h-4 w-4 mr-1" />
          Go Live
        </Button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b">
        <button
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            tab === "live"
              ? "border-primary text-foreground"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => setTab("live")}
        >
          <div className="flex items-center gap-1.5">
            <Radio className="h-3.5 w-3.5" />
            Live
          </div>
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            tab === "recordings"
              ? "border-primary text-foreground"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => setTab("recordings")}
        >
          Recordings
        </button>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-48 rounded-lg" />
          ))}
        </div>
      ) : streams.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
          <Video className="h-12 w-12 mb-3 opacity-40" />
          <p className="text-sm">
            {tab === "live"
              ? "No live streams right now"
              : "No recordings yet"}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {streams.map((stream) => (
            <StreamCard
              key={stream.id}
              stream={stream}
              onClick={() => setViewingStream(stream)}
              onDeleted={refetch}
            />
          ))}
        </div>
      )}

      {/* Broadcast dialog */}
      <BroadcastDialog
        open={broadcastOpen}
        onOpenChange={setBroadcastOpen}
        onStreamStarted={() => refetch()}
        onStreamEnded={() => refetch()}
      />

      {/* Stream viewer dialog */}
      {viewingStream && (
        <Dialog
          open={!!viewingStream}
          onOpenChange={(open) => !open && setViewingStream(null)}
        >
          <DialogContent className="sm:max-w-2xl p-0 gap-0">
            <DialogHeader className="px-4 pt-4 pb-2">
              <DialogTitle className="flex items-center gap-2">
                {viewingStream.status === "live" && (
                  <Badge
                    variant="destructive"
                    className="animate-pulse text-xs"
                  >
                    LIVE
                  </Badge>
                )}
                {viewingStream.title}
              </DialogTitle>
            </DialogHeader>
            <div className="px-4 pb-4">
              <StreamViewer
                stream={viewingStream}
                className="aspect-video"
              />
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}

function StreamCard({
  stream,
  onClick,
  onDeleted,
}: {
  stream: StreamResponse;
  onClick: () => void;
  onDeleted: () => void;
}) {
  const { deleteStream } = useDeleteStream();
  const isLive = stream.status === "live";

  const sourceIcon =
    stream.source_type === "browser" ? (
      <Camera className="h-3 w-3" />
    ) : (
      <Monitor className="h-3 w-3" />
    );

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm(`Delete "${stream.title}"? This cannot be undone.`)) return;
    try {
      await deleteStream(stream.id);
      toast.success(`Stream "${stream.title}" deleted`);
      onDeleted();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to delete stream"
      );
    }
  };

  return (
    <Card
      className="cursor-pointer hover:border-primary/50 transition-colors overflow-hidden"
      onClick={onClick}
    >
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2 min-w-0">
            {isLive && (
              <Badge
                variant="destructive"
                className="animate-pulse text-xs flex-shrink-0"
              >
                LIVE
              </Badge>
            )}
            <CardTitle className="text-sm truncate">{stream.title}</CardTitle>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-7 p-0 flex-shrink-0"
                onClick={(e) => e.stopPropagation()}
              >
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={handleDelete}
                className="text-destructive"
              >
                <Trash2 className="h-4 w-4 mr-1" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
        <CardDescription className="flex items-center gap-2 text-xs">
          {stream.display_name || stream.username}
          <span className="flex items-center gap-0.5 text-muted-foreground">
            {sourceIcon}
            {stream.source_type}
          </span>
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-0">
        <p className="text-xs text-muted-foreground">
          {isLive ? "Started" : "Recorded"} {formatTime(stream.started_at)}
        </p>
      </CardContent>
    </Card>
  );
}

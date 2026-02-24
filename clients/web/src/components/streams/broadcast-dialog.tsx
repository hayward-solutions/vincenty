"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { api } from "@/lib/api";
import { useWebSocket } from "@/lib/websocket-context";
import { useCreateStream, useEndStream } from "@/lib/hooks/use-streams";
import { useWebRTCPublish } from "@/lib/hooks/use-webrtc";
import type { Group, StreamResponse } from "@/types/api";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
  Camera,
  Monitor,
  StopCircle,
  Video,
  Check,
  Loader2,
} from "lucide-react";

const MEDIA_BASE = process.env.NEXT_PUBLIC_MEDIA_URL || "";

interface BroadcastDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onStreamStarted?: (stream: StreamResponse) => void;
  onStreamEnded?: () => void;
}

type SourceType = "camera" | "screen";

export function BroadcastDialog({
  open,
  onOpenChange,
  onStreamStarted,
  onStreamEnded,
}: BroadcastDialogProps) {
  const { sendMessage } = useWebSocket();
  const { createStream, isLoading: isCreating } = useCreateStream();
  const { endStream, isLoading: isEnding } = useEndStream();
  const { publish, stop: stopPublish, isPublishing } = useWebRTCPublish();

  const [title, setTitle] = useState("");
  const [sourceType, setSourceType] = useState<SourceType>("camera");
  const [groups, setGroups] = useState<Group[]>([]);
  const [selectedGroupIds, setSelectedGroupIds] = useState<Set<string>>(
    new Set()
  );
  const [mediaStream, setMediaStream] = useState<MediaStream | null>(null);
  const [activeStream, setActiveStream] = useState<StreamResponse | null>(null);
  const [elapsedSeconds, setElapsedSeconds] = useState(0);

  const videoRef = useRef<HTMLVideoElement>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const locationWatchRef = useRef<number | null>(null);
  const locationIntervalRef = useRef<ReturnType<typeof setInterval> | null>(
    null
  );
  const lastLocationRef = useRef<GeolocationPosition | null>(null);

  // Fetch user's groups on open
  useEffect(() => {
    if (!open) return;
    api
      .get<Group[]>("/api/v1/users/me/groups")
      .then((result) => setGroups(result ?? []))
      .catch(() => setGroups([]));
  }, [open]);

  // Preview video in the <video> element
  useEffect(() => {
    if (videoRef.current && mediaStream) {
      videoRef.current.srcObject = mediaStream;
    }
  }, [mediaStream]);

  // Elapsed timer while live
  useEffect(() => {
    if (activeStream) {
      const startTime = new Date(activeStream.started_at).getTime();
      timerRef.current = setInterval(() => {
        setElapsedSeconds(
          Math.floor((Date.now() - startTime) / 1000)
        );
      }, 1000);
    }
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [activeStream]);

  // Location broadcasting at 1Hz while live
  const startLocationBroadcast = useCallback(
    (streamId: string) => {
      if (typeof navigator === "undefined" || !navigator.geolocation) return;

      locationWatchRef.current = navigator.geolocation.watchPosition(
        (position) => {
          lastLocationRef.current = position;
        },
        () => {
          // Silently ignore location errors during broadcast
        },
        { enableHighAccuracy: true, timeout: 10_000, maximumAge: 2_000 }
      );

      locationIntervalRef.current = setInterval(() => {
        const pos = lastLocationRef.current;
        if (!pos) return;
        sendMessage("stream_location", {
          stream_id: streamId,
          lat: pos.coords.latitude,
          lng: pos.coords.longitude,
          altitude: pos.coords.altitude ?? undefined,
          heading: pos.coords.heading ?? undefined,
          speed: pos.coords.speed ?? undefined,
        });
      }, 1000);
    },
    [sendMessage]
  );

  const stopLocationBroadcast = useCallback(() => {
    if (locationWatchRef.current !== null) {
      navigator.geolocation.clearWatch(locationWatchRef.current);
      locationWatchRef.current = null;
    }
    if (locationIntervalRef.current) {
      clearInterval(locationIntervalRef.current);
      locationIntervalRef.current = null;
    }
    lastLocationRef.current = null;
  }, []);

  // Capture media
  const captureMedia = useCallback(async (source: SourceType) => {
    // Stop any existing stream
    if (mediaStream) {
      for (const track of mediaStream.getTracks()) {
        track.stop();
      }
    }

    try {
      let stream: MediaStream;
      if (source === "camera") {
        stream = await navigator.mediaDevices.getUserMedia({
          video: true,
          audio: true,
        });
      } else {
        stream = await navigator.mediaDevices.getDisplayMedia({
          video: true,
          audio: true,
        });
      }
      setMediaStream(stream);
      setSourceType(source);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to access media device"
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mediaStream]);

  // Go Live
  const handleGoLive = useCallback(async () => {
    if (!mediaStream || !title.trim() || selectedGroupIds.size === 0) return;

    try {
      const stream = await createStream({
        title: title.trim(),
        group_ids: Array.from(selectedGroupIds),
      });

      const token = localStorage.getItem("access_token") ?? "";
      const whipUrl = `${MEDIA_BASE}/${stream.media_path}/whip?token=${encodeURIComponent(token)}`;

      await publish(mediaStream, whipUrl);

      setActiveStream(stream);
      startLocationBroadcast(stream.id);
      onStreamStarted?.(stream);
      toast.success("You are now live!");
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to start stream"
      );
    }
  }, [
    mediaStream,
    title,
    selectedGroupIds,
    createStream,
    publish,
    startLocationBroadcast,
    onStreamStarted,
  ]);

  // End Stream
  const handleEndStream = useCallback(async () => {
    if (!activeStream) return;

    try {
      await endStream(activeStream.id);
      await stopPublish();
      stopLocationBroadcast();

      // Stop media tracks
      if (mediaStream) {
        for (const track of mediaStream.getTracks()) {
          track.stop();
        }
      }

      setActiveStream(null);
      setMediaStream(null);
      setElapsedSeconds(0);
      onStreamEnded?.();
      toast.success("Stream ended");
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to end stream"
      );
    }
  }, [
    activeStream,
    endStream,
    stopPublish,
    stopLocationBroadcast,
    mediaStream,
    onStreamEnded,
  ]);

  // Cleanup on close
  const handleClose = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        if (activeStream) {
          // Don't close while live — must end stream first
          return;
        }
        // Stop media
        if (mediaStream) {
          for (const track of mediaStream.getTracks()) {
            track.stop();
          }
          setMediaStream(null);
        }
        stopLocationBroadcast();
        setTitle("");
        setSelectedGroupIds(new Set());
        setElapsedSeconds(0);
      }
      onOpenChange(nextOpen);
    },
    [activeStream, mediaStream, stopLocationBroadcast, onOpenChange]
  );

  const toggleGroup = useCallback((groupId: string) => {
    setSelectedGroupIds((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) {
        next.delete(groupId);
      } else {
        next.add(groupId);
      }
      return next;
    });
  }, []);

  const formatDuration = (seconds: number) => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    if (h > 0) {
      return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
    }
    return `${m}:${String(s).padStart(2, "0")}`;
  };

  const canGoLive =
    !!mediaStream && title.trim().length > 0 && selectedGroupIds.size > 0;

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {activeStream ? "Live Broadcast" : "Start Broadcast"}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Video Preview */}
          <div className="relative aspect-video bg-black rounded-lg overflow-hidden">
            {mediaStream ? (
              <video
                ref={videoRef}
                autoPlay
                muted
                playsInline
                className="w-full h-full object-contain"
              />
            ) : (
              <div className="flex items-center justify-center h-full text-muted-foreground">
                <Video className="h-12 w-12 opacity-50" />
              </div>
            )}
            {activeStream && (
              <div className="absolute top-2 left-2 flex items-center gap-2">
                <Badge variant="destructive" className="animate-pulse">
                  LIVE
                </Badge>
                <span className="text-white text-sm font-mono bg-black/60 px-2 py-0.5 rounded">
                  {formatDuration(elapsedSeconds)}
                </span>
              </div>
            )}
          </div>

          {/* Source selection (only before going live) */}
          {!activeStream && (
            <>
              <div className="flex gap-2">
                <Button
                  variant={
                    sourceType === "camera" && mediaStream
                      ? "default"
                      : "outline"
                  }
                  size="sm"
                  onClick={() => captureMedia("camera")}
                  className="flex-1"
                >
                  <Camera className="h-4 w-4 mr-1" />
                  Camera
                </Button>
                <Button
                  variant={
                    sourceType === "screen" && mediaStream
                      ? "default"
                      : "outline"
                  }
                  size="sm"
                  onClick={() => captureMedia("screen")}
                  className="flex-1"
                >
                  <Monitor className="h-4 w-4 mr-1" />
                  Screen
                </Button>
              </div>

              <Separator />

              <div className="space-y-2">
                <Label htmlFor="broadcast-title">Title</Label>
                <Input
                  id="broadcast-title"
                  placeholder="Stream title"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label>Share with Groups</Label>
                {groups.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    No groups available
                  </p>
                ) : (
                  <ScrollArea className="max-h-32">
                    <div className="flex flex-wrap gap-1.5">
                      {groups.map((group) => {
                        const selected = selectedGroupIds.has(group.id);
                        return (
                          <Badge
                            key={group.id}
                            variant={selected ? "default" : "outline"}
                            className="cursor-pointer select-none"
                            onClick={() => toggleGroup(group.id)}
                          >
                            {selected && <Check className="h-3 w-3 mr-1" />}
                            {group.name}
                          </Badge>
                        );
                      })}
                    </div>
                  </ScrollArea>
                )}
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          {activeStream ? (
            <Button
              variant="destructive"
              onClick={handleEndStream}
              disabled={isEnding}
            >
              {isEnding ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <StopCircle className="h-4 w-4 mr-1" />
              )}
              End Stream
            </Button>
          ) : (
            <>
              <Button
                variant="outline"
                onClick={() => handleClose(false)}
              >
                Cancel
              </Button>
              <Button
                onClick={handleGoLive}
                disabled={!canGoLive || isCreating || isPublishing}
              >
                {isCreating || isPublishing ? (
                  <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                ) : (
                  <Video className="h-4 w-4 mr-1" />
                )}
                Go Live
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

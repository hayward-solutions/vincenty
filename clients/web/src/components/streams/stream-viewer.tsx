"use client";

import { useEffect, useRef, useState } from "react";
import { useWebRTCView } from "@/lib/hooks/use-webrtc";
import type { StreamResponse } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Loader2, WifiOff } from "lucide-react";

const MEDIA_BASE = process.env.NEXT_PUBLIC_MEDIA_URL || "";
const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

interface StreamViewerProps {
  stream: StreamResponse;
  className?: string;
  /** Called when the video element's time updates (for synced playback). */
  onTimeUpdate?: (currentTime: number) => void;
}

export function StreamViewer({
  stream,
  className,
  onTimeUpdate,
}: StreamViewerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const {
    connect,
    disconnect,
    mediaStream,
    isConnected,
    error,
  } = useWebRTCView();
  const [connecting, setConnecting] = useState(false);

  const isLive = stream.status === "live";

  // Connect to live WHEP stream
  useEffect(() => {
    if (!isLive) return;

    let cancelled = false;

    async function connectToStream() {
      setConnecting(true);
      try {
        const token = localStorage.getItem("access_token") ?? "";
        const whepUrl = `${MEDIA_BASE}/${stream.media_path}/whep?token=${encodeURIComponent(token)}`;
        await connect(whepUrl);
      } catch {
        // Error is set in the hook
      } finally {
        if (!cancelled) setConnecting(false);
      }
    }

    connectToStream();

    return () => {
      cancelled = true;
      disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stream.id, isLive]);

  // Attach the WebRTC mediaStream to the video element
  useEffect(() => {
    if (videoRef.current && mediaStream) {
      videoRef.current.srcObject = mediaStream;
    }
  }, [mediaStream]);

  // Handle timeupdate for synced playback of recordings
  useEffect(() => {
    if (!onTimeUpdate || !videoRef.current) return;

    const video = videoRef.current;
    const handler = () => {
      onTimeUpdate(video.currentTime);
    };
    video.addEventListener("timeupdate", handler);
    return () => video.removeEventListener("timeupdate", handler);
  }, [onTimeUpdate]);

  // Calculate duration
  const duration =
    stream.ended_at && stream.started_at
      ? Math.floor(
          (new Date(stream.ended_at).getTime() -
            new Date(stream.started_at).getTime()) /
            1000
        )
      : null;

  const formatDuration = (seconds: number) => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    if (h > 0) {
      return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
    }
    return `${m}:${String(s).padStart(2, "0")}`;
  };

  // Recording URL with auth token
  const recordingUrl = stream.recording_url
    ? `${API_BASE}${stream.recording_url}?token=${encodeURIComponent(localStorage.getItem("access_token") ?? "")}`
    : null;

  return (
    <div className={`relative bg-black rounded-lg overflow-hidden ${className ?? ""}`}>
      {isLive ? (
        <>
          <video
            ref={videoRef}
            autoPlay
            playsInline
            className="w-full h-full object-contain"
          />
          {(connecting || (!isConnected && !error)) && (
            <div className="absolute inset-0 flex items-center justify-center">
              <Loader2 className="h-8 w-8 text-white animate-spin" />
            </div>
          )}
          {error && !isConnected && (
            <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-white">
              <WifiOff className="h-8 w-8 opacity-60" />
              <p className="text-sm opacity-80">Failed to connect</p>
              <Button
                variant="secondary"
                size="sm"
                onClick={async () => {
                  setConnecting(true);
                  try {
                    const token = localStorage.getItem("access_token") ?? "";
                    const whepUrl = `${MEDIA_BASE}/${stream.media_path}/whep?token=${encodeURIComponent(token)}`;
                    await connect(whepUrl);
                  } catch {
                    // Error shown in hook
                  } finally {
                    setConnecting(false);
                  }
                }}
              >
                Retry
              </Button>
            </div>
          )}
        </>
      ) : recordingUrl ? (
        <video
          ref={videoRef}
          src={recordingUrl}
          controls
          playsInline
          className="w-full h-full object-contain"
        />
      ) : (
        <div className="aspect-video flex items-center justify-center text-muted-foreground">
          <p className="text-sm">Recording not available</p>
        </div>
      )}

      {/* Info overlay */}
      <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent p-3">
        <div className="flex items-center gap-2">
          {isLive && (
            <Badge variant="destructive" className="animate-pulse text-xs">
              LIVE
            </Badge>
          )}
          <span className="text-white text-sm font-medium truncate">
            {stream.title}
          </span>
        </div>
        <div className="flex items-center gap-2 mt-1">
          {(stream.display_name || stream.username) && (
            <span className="text-white/70 text-xs">
              {stream.display_name || stream.username}
            </span>
          )}
          {!isLive && duration !== null && (
            <span className="text-white/50 text-xs">
              {formatDuration(duration)}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

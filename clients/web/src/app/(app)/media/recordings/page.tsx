"use client";

import { useState, useEffect, useCallback } from "react";
import { useAuth } from "@/lib/auth-context";
import { api, ApiError } from "@/lib/api";
import { useActiveCalls } from "@/lib/hooks/use-calls";
import {
  useRoomRecordings,
  useStartRecording,
  useStopRecording,
} from "@/lib/hooks/use-recordings";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Camera,
  Video,
  Circle,
  Play,
  Square,
  Clock,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { toast } from "sonner";
import type { MediaRoom, Recording, VideoFeed, Group } from "@/types/api";

interface SelectedSource {
  id: string;
  name: string;
  type: "call" | "feed";
  isActive: boolean;
}

export default function RecordingsPage() {
  useAuth();
  const { calls, isLoading: callsLoading } = useActiveCalls();
  const { startRecording, isLoading: startingRecording } = useStartRecording();
  const { stopRecording, isLoading: stoppingRecording } = useStopRecording();

  const [selectedSource, setSelectedSource] = useState<SelectedSource | null>(
    null
  );
  const [callsExpanded, setCallsExpanded] = useState(true);
  const [feedsExpanded, setFeedsExpanded] = useState(true);

  // Fetch video feeds from user's groups
  const [feeds, setFeeds] = useState<VideoFeed[]>([]);
  const [feedsLoading, setFeedsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function fetchFeeds() {
      setFeedsLoading(true);
      try {
        const groups = await api.get<Group[]>("/api/v1/users/me/groups");
        const allFeeds: VideoFeed[] = [];
        for (const group of groups) {
          try {
            const groupFeeds = await api.get<VideoFeed[]>(
              `/api/v1/groups/${group.id}/feeds`
            );
            allFeeds.push(...groupFeeds);
          } catch {
            // skip groups where feeds fail to load
          }
        }
        if (!cancelled) setFeeds(allFeeds);
      } catch {
        // silently fail — feeds section will show empty
      } finally {
        if (!cancelled) setFeedsLoading(false);
      }
    }
    fetchFeeds();
    return () => {
      cancelled = true;
    };
  }, []);

  // Fetch recordings for the selected source
  const {
    recordings,
    isLoading: recordingsLoading,
    refetch: refetchRecordings,
  } = useRoomRecordings(selectedSource?.id ?? "");

  const handleStartRecording = useCallback(
    async (roomId: string) => {
      try {
        await startRecording(roomId);
        refetchRecordings();
        toast("Recording started");
      } catch (err) {
        toast.error(
          err instanceof ApiError ? err.message : "Failed to start recording"
        );
      }
    },
    [startRecording, refetchRecordings]
  );

  const handleStopRecording = useCallback(
    async (recordingId: string) => {
      try {
        await stopRecording(recordingId);
        refetchRecordings();
        toast("Recording stopped");
      } catch (err) {
        toast.error(
          err instanceof ApiError ? err.message : "Failed to stop recording"
        );
      }
    },
    [stopRecording, refetchRecordings]
  );

  const formatDuration = (secs?: number): string => {
    if (!secs) return "--:--";
    const m = Math.floor(secs / 60);
    const s = Math.floor(secs % 60);
    return `${m}:${s.toString().padStart(2, "0")}`;
  };

  const formatFileSize = (bytes?: number): string => {
    if (!bytes) return "--";
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const formatDate = (dateStr: string): string => {
    return new Date(dateStr).toLocaleString();
  };

  const statusBadgeVariant = (
    status: Recording["status"]
  ): "default" | "secondary" | "destructive" | "outline" => {
    switch (status) {
      case "recording":
        return "destructive";
      case "processing":
        return "secondary";
      case "complete":
        return "default";
      case "failed":
        return "destructive";
      default:
        return "secondary";
    }
  };

  const activeRecording = recordings.find((r) => r.status === "recording");
  const isSourceActive = selectedSource?.isActive ?? false;

  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      {/* Left panel: sources list */}
      <div
        className={`w-full md:w-72 shrink-0 border-r flex flex-col ${
          selectedSource ? "hidden md:flex" : "flex"
        }`}
      >
        <div className="p-3 border-b">
          <h2 className="text-sm font-semibold flex items-center gap-1.5">
            <Video className="h-4 w-4" />
            Recordings
          </h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Select a source to browse recordings
          </p>
        </div>

        <ScrollArea className="flex-1">
          <div className="p-2 space-y-1">
            {/* Calls section */}
            <button
              type="button"
              className="flex items-center gap-1.5 w-full px-2 py-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wide hover:text-foreground transition-colors"
              onClick={() => setCallsExpanded(!callsExpanded)}
            >
              {callsExpanded ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              )}
              Calls
            </button>
            {callsExpanded && (
              <>
                {callsLoading ? (
                  <>
                    <Skeleton className="h-12 w-full" />
                    <Skeleton className="h-12 w-full" />
                  </>
                ) : calls.length === 0 ? (
                  <p className="text-xs text-muted-foreground text-center py-3 px-2">
                    No active calls
                  </p>
                ) : (
                  calls.map((call) => (
                    <Button
                      key={call.id}
                      variant={
                        selectedSource?.id === call.id ? "secondary" : "ghost"
                      }
                      className="w-full justify-start h-auto py-2 px-3"
                      onClick={() =>
                        setSelectedSource({
                          id: call.id,
                          name: call.name,
                          type: "call",
                          isActive: call.is_active,
                        })
                      }
                    >
                      <div className="flex items-center gap-2 min-w-0 w-full">
                        <Video className="h-4 w-4 shrink-0 text-muted-foreground" />
                        <div className="min-w-0 flex-1 text-left">
                          <p className="text-sm font-medium truncate">
                            {call.name}
                          </p>
                          <div className="flex items-center gap-1.5">
                            <Badge
                              variant="outline"
                              className="text-[10px] px-1 py-0"
                            >
                              call
                            </Badge>
                            <span className="text-xs text-muted-foreground">
                              {call.is_active ? "Active" : "Ended"}
                            </span>
                          </div>
                        </div>
                        {call.is_active && (
                          <span className="h-2 w-2 rounded-full bg-green-500 shrink-0" />
                        )}
                      </div>
                    </Button>
                  ))
                )}
              </>
            )}

            {/* Video Feeds section */}
            <button
              type="button"
              className="flex items-center gap-1.5 w-full px-2 py-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wide hover:text-foreground transition-colors mt-2"
              onClick={() => setFeedsExpanded(!feedsExpanded)}
            >
              {feedsExpanded ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              )}
              Video Feeds
            </button>
            {feedsExpanded && (
              <>
                {feedsLoading ? (
                  <>
                    <Skeleton className="h-12 w-full" />
                    <Skeleton className="h-12 w-full" />
                  </>
                ) : feeds.length === 0 ? (
                  <p className="text-xs text-muted-foreground text-center py-3 px-2">
                    No video feeds
                  </p>
                ) : (
                  feeds.map((feed) => (
                    <Button
                      key={feed.id}
                      variant={
                        selectedSource?.id === feed.id ? "secondary" : "ghost"
                      }
                      className="w-full justify-start h-auto py-2 px-3"
                      onClick={() =>
                        setSelectedSource({
                          id: feed.id,
                          name: feed.name,
                          type: "feed",
                          isActive: feed.is_active,
                        })
                      }
                    >
                      <div className="flex items-center gap-2 min-w-0 w-full">
                        <Camera className="h-4 w-4 shrink-0 text-muted-foreground" />
                        <div className="min-w-0 flex-1 text-left">
                          <p className="text-sm font-medium truncate">
                            {feed.name}
                          </p>
                          <div className="flex items-center gap-1.5">
                            <Badge
                              variant="outline"
                              className="text-[10px] px-1 py-0"
                            >
                              {feed.feed_type}
                            </Badge>
                            <span className="text-xs text-muted-foreground">
                              {feed.is_active ? "Active" : "Inactive"}
                            </span>
                          </div>
                        </div>
                        {feed.is_active && (
                          <span className="h-2 w-2 rounded-full bg-green-500 shrink-0" />
                        )}
                      </div>
                    </Button>
                  ))
                )}
              </>
            )}
          </div>
        </ScrollArea>
      </div>

      <Separator orientation="vertical" className="hidden md:block" />

      {/* Right panel: recordings for selected source */}
      <div
        className={`flex-1 flex flex-col min-w-0 ${
          selectedSource ? "flex" : "hidden md:flex"
        }`}
      >
        {selectedSource ? (
          <>
            {/* Header */}
            <div className="p-3 border-b flex items-center justify-between">
              <div className="flex items-center gap-2 min-w-0">
                <Button
                  variant="ghost"
                  size="sm"
                  className="md:hidden h-8 w-8 p-0 shrink-0"
                  onClick={() => setSelectedSource(null)}
                >
                  <Clock className="h-4 w-4" />
                </Button>
                <h3 className="text-sm font-semibold truncate">
                  {selectedSource.name} — Recordings
                </h3>
                <Badge variant="outline" className="text-xs shrink-0">
                  {selectedSource.type}
                </Badge>
              </div>
              {isSourceActive && selectedSource.type === "call" && (
                <div className="flex items-center gap-2 shrink-0">
                  {activeRecording ? (
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleStopRecording(activeRecording.id)}
                      disabled={stoppingRecording}
                    >
                      <Square className="h-4 w-4 mr-1" />
                      {stoppingRecording ? "Stopping..." : "Stop Recording"}
                    </Button>
                  ) : (
                    <Button
                      variant="default"
                      size="sm"
                      onClick={() => handleStartRecording(selectedSource.id)}
                      disabled={startingRecording}
                    >
                      <Circle className="h-4 w-4 mr-1 fill-current" />
                      {startingRecording ? "Starting..." : "Start Recording"}
                    </Button>
                  )}
                </div>
              )}
            </div>

            {/* Recordings list */}
            <ScrollArea className="flex-1">
              <div className="p-4">
                {recordingsLoading ? (
                  <div className="space-y-3">
                    <Skeleton className="h-24 w-full" />
                    <Skeleton className="h-24 w-full" />
                  </div>
                ) : recordings.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-16 gap-3 text-muted-foreground">
                    <Camera className="h-12 w-12" />
                    <p className="text-sm">
                      No recordings for this source
                    </p>
                    {isSourceActive && selectedSource.type === "call" && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          handleStartRecording(selectedSource.id)
                        }
                        disabled={startingRecording}
                      >
                        <Circle className="h-4 w-4 mr-1 fill-current" />
                        Start Recording
                      </Button>
                    )}
                  </div>
                ) : (
                  <div className="space-y-3">
                    {recordings.map((recording) => (
                      <Card key={recording.id}>
                        <CardContent className="p-4">
                          <div className="flex items-start justify-between">
                            <div className="space-y-1.5">
                              <div className="flex items-center gap-2">
                                <Badge
                                  variant={statusBadgeVariant(recording.status)}
                                >
                                  {recording.status === "recording" && (
                                    <Circle className="h-2 w-2 mr-1 fill-current animate-pulse" />
                                  )}
                                  {recording.status}
                                </Badge>
                                <span className="text-xs text-muted-foreground">
                                  {recording.file_type}
                                </span>
                              </div>
                              <div className="flex items-center gap-4 text-xs text-muted-foreground">
                                <span className="flex items-center gap-1">
                                  <Clock className="h-3 w-3" />
                                  {formatDuration(recording.duration_secs)}
                                </span>
                                <span>
                                  {formatFileSize(recording.file_size_bytes)}
                                </span>
                                <span>{formatDate(recording.started_at)}</span>
                              </div>
                            </div>
                            <div className="flex items-center gap-2 shrink-0 ml-4">
                              {recording.status === "recording" ? (
                                <Button
                                  variant="destructive"
                                  size="sm"
                                  onClick={() =>
                                    handleStopRecording(recording.id)
                                  }
                                  disabled={stoppingRecording}
                                >
                                  <Square className="h-4 w-4" />
                                </Button>
                              ) : recording.status === "complete" &&
                                recording.playback_url ? (
                                <Button variant="outline" size="sm" asChild>
                                  <a
                                    href={recording.playback_url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                  >
                                    <Play className="h-4 w-4 mr-1" />
                                    Play
                                  </a>
                                </Button>
                              ) : null}
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                )}
              </div>
            </ScrollArea>
          </>
        ) : (
          <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
            <Camera className="h-12 w-12" />
            <p className="text-sm">Select a source to view recordings</p>
          </div>
        )}
      </div>
    </div>
  );
}

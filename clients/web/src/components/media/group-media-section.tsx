"use client";

import { useState } from "react";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { useGroupCalls, useCreateCall, useJoinCall } from "@/lib/hooks/use-calls";
import {
  useGroupFeeds,
  useCreateFeed,
  useStartFeed,
  useStopFeed,
  useViewFeed,
  useDeleteFeed,
} from "@/lib/hooks/use-feeds";
import {
  usePTTChannels,
  useCreatePTTChannel,
  useJoinPTTChannel,
  useDeletePTTChannel,
} from "@/lib/hooks/use-ptt";
import { CallView } from "@/components/media/call-view";
import { AudioCallView } from "@/components/media/audio-call-view";
import { PTTButton } from "@/components/media/ptt-button";
import { FeedViewer } from "@/components/media/feed-viewer";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Phone,
  Video,
  Mic,
  Plus,
  Play,
  Square,
  Trash2,
  Eye,
  Radio,
  ChevronDown,
  ChevronRight,
  MoreVertical,
} from "lucide-react";
import type {
  JoinRoomResponse,
  VideoFeedStartResponse,
} from "@/types/api";

const LIVEKIT_URL =
  process.env.NEXT_PUBLIC_LIVEKIT_URL || "ws://localhost:7880";

interface GroupMediaSectionProps {
  groupId: string;
}

export function GroupMediaSection({ groupId }: GroupMediaSectionProps) {
  return (
    <div className="space-y-4">
      <h2 className="text-lg font-medium">Media</h2>
      <FeedsSection groupId={groupId} />
      <PTTSection groupId={groupId} />
      <CallsSection groupId={groupId} />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Calls Section
// ---------------------------------------------------------------------------

function CallsSection({ groupId }: { groupId: string }) {
  const { calls, isLoading, refetch } = useGroupCalls(groupId);
  const { createCall, isLoading: creating } = useCreateCall();
  const { joinCall, isLoading: joining } = useJoinCall();

  const [expanded, setExpanded] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [callName, setCallName] = useState("");
  const [videoEnabled, setVideoEnabled] = useState(false);
  const [activeCall, setActiveCall] = useState<JoinRoomResponse | null>(null);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      const resp = await createCall({
        group_id: groupId,
        name: callName || undefined,
        video_enabled: videoEnabled,
      });
      toast.success("Call started");
      setActiveCall(resp);
      setDialogOpen(false);
      setCallName("");
      setVideoEnabled(false);
      refetch();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to start call"
      );
    }
  }

  async function handleJoin(roomId: string) {
    try {
      const resp = await joinCall(roomId);
      setActiveCall(resp);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to join call"
      );
    }
  }

  return (
    <Card>
      <CardHeader
        className="cursor-pointer select-none"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {expanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <Phone className="h-4 w-4" />
            <CardTitle className="text-base">Calls</CardTitle>
            <Badge variant="secondary">{calls.length}</Badge>
          </div>
          <Button
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setDialogOpen(true);
            }}
          >
            <Plus className="h-4 w-4 mr-1" />
            Start Call
          </Button>
        </div>
      </CardHeader>

      {expanded && (
        <CardContent className="space-y-3">
          {activeCall && (
            <div className="border rounded-lg overflow-hidden">
              {activeCall.room.room_type === "call" ? (
                <CallView
                  token={activeCall.token}
                  serverUrl={activeCall.url || LIVEKIT_URL}
                  roomName={activeCall.room.name}
                  onDisconnected={() => {
                    setActiveCall(null);
                    refetch();
                  }}
                />
              ) : (
                <AudioCallView
                  token={activeCall.token}
                  serverUrl={activeCall.url || LIVEKIT_URL}
                  roomName={activeCall.room.name}
                  onDisconnected={() => {
                    setActiveCall(null);
                    refetch();
                  }}
                />
              )}
            </div>
          )}

          {isLoading ? (
            <p className="text-sm text-muted-foreground">Loading calls...</p>
          ) : calls.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No active calls in this group.
            </p>
          ) : (
            <div className="space-y-2">
              {calls.map((call) => (
                <div
                  key={call.id}
                  className="flex items-center justify-between rounded-md border px-3 py-2"
                >
                  <div className="flex items-center gap-2">
                    <Phone className="h-4 w-4 text-muted-foreground" />
                    <span className="text-sm font-medium">{call.name}</span>
                    <Badge variant="outline" className="text-xs">
                      {call.max_participants} participants
                    </Badge>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={joining || activeCall?.room.id === call.id}
                    onClick={() => handleJoin(call.id)}
                  >
                    Join
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      )}

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Start Call</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="call-name">Name</Label>
              <Input
                id="call-name"
                value={callName}
                onChange={(e) => setCallName(e.target.value)}
                placeholder="Optional call name"
              />
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="call-video"
                checked={videoEnabled}
                onCheckedChange={(checked) =>
                  setVideoEnabled(checked === true)
                }
              />
              <Label htmlFor="call-video">Enable video</Label>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setDialogOpen(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={creating}>
                {creating ? "Starting..." : "Start Call"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ---------------------------------------------------------------------------
// Video Feeds Section
// ---------------------------------------------------------------------------

function FeedsSection({ groupId }: { groupId: string }) {
  const { feeds, isLoading, refetch } = useGroupFeeds(groupId);
  const { createFeed, isLoading: creating } = useCreateFeed();
  const { startFeed, isLoading: starting } = useStartFeed();
  const { stopFeed, isLoading: stopping } = useStopFeed();
  const { viewFeed, isLoading: loadingView } = useViewFeed();
  const { deleteFeed, isLoading: deleting } = useDeleteFeed();

  const [expanded, setExpanded] = useState(true);
  const [addOpen, setAddOpen] = useState(false);
  const [watchData, setWatchData] = useState<{
    token: string;
    url: string;
    name: string;
  } | null>(null);
  const [startResult, setStartResult] =
    useState<VideoFeedStartResponse | null>(null);

  // Add feed form state
  const [feedName, setFeedName] = useState("");
  const [feedType, setFeedType] = useState<
    "rtsp" | "rtmp" | "whip" | "phone_cam"
  >("rtsp");
  const [sourceUrl, setSourceUrl] = useState("");

  async function handleAddFeed(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createFeed({
        name: feedName,
        feed_type: feedType,
        source_url:
          feedType === "rtsp" || feedType === "rtmp" ? sourceUrl : undefined,
        group_id: groupId,
      });
      toast.success("Feed created");
      setAddOpen(false);
      setFeedName("");
      setFeedType("rtsp");
      setSourceUrl("");
      refetch();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to create feed"
      );
    }
  }

  async function handleStart(feedId: string) {
    try {
      const result = await startFeed(feedId);
      setStartResult(result);
      toast.success("Feed started");
      refetch();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to start feed"
      );
    }
  }

  async function handleStop(feedId: string) {
    try {
      await stopFeed(feedId);
      toast.success("Feed stopped");
      setStartResult(null);
      refetch();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to stop feed"
      );
    }
  }

  async function handleWatch(feedId: string, feedName: string) {
    try {
      const resp = await viewFeed(feedId);
      setWatchData({
        token: resp.token,
        url: resp.url || LIVEKIT_URL,
        name: feedName,
      });
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to view feed"
      );
    }
  }

  async function handleDelete(feedId: string) {
    if (!confirm("Delete this feed?")) return;
    try {
      await deleteFeed(feedId);
      toast.success("Feed deleted");
      refetch();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to delete feed"
      );
    }
  }

  const needsSourceUrl = feedType === "rtsp" || feedType === "rtmp";

  return (
    <Card>
      <CardHeader
        className="cursor-pointer select-none"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {expanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <Video className="h-4 w-4" />
            <CardTitle className="text-base">Video Feeds</CardTitle>
            <Badge variant="secondary">{feeds.length}</Badge>
          </div>
          <Button
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setAddOpen(true);
            }}
          >
            <Plus className="h-4 w-4 mr-1" />
            Add Feed
          </Button>
        </div>
      </CardHeader>

      {expanded && (
        <CardContent className="space-y-3">
          {startResult &&
            (startResult.ingest_url || startResult.stream_key) && (
              <div className="rounded-md border border-blue-500/30 bg-blue-500/10 p-3 space-y-1">
                <p className="text-sm font-medium">Feed Ingest Details</p>
                {startResult.ingest_url && (
                  <p className="text-xs text-muted-foreground break-all">
                    <span className="font-medium">Ingest URL:</span>{" "}
                    {startResult.ingest_url}
                  </p>
                )}
                {startResult.stream_key && (
                  <p className="text-xs text-muted-foreground break-all">
                    <span className="font-medium">Stream Key:</span>{" "}
                    {startResult.stream_key}
                  </p>
                )}
              </div>
            )}

          {isLoading ? (
            <p className="text-sm text-muted-foreground">Loading feeds...</p>
          ) : feeds.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No video feeds in this group.
            </p>
          ) : (
            <div className="space-y-2">
              {feeds.map((feed) => (
                <div
                  key={feed.id}
                  className="flex items-center justify-between rounded-md border px-3 py-2"
                >
                  <div className="flex items-center gap-2">
                    <Video className="h-4 w-4 text-muted-foreground" />
                    <span className="text-sm font-medium">{feed.name}</span>
                    <Badge variant="outline" className="text-xs">
                      {feed.feed_type}
                    </Badge>
                    {feed.is_active ? (
                      <Badge className="bg-green-600 text-white text-xs">
                        Active
                      </Badge>
                    ) : (
                      <Badge variant="secondary" className="text-xs">
                        Inactive
                      </Badge>
                    )}
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                        <MoreVertical className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      {!feed.is_active ? (
                        <DropdownMenuItem
                          onClick={() => handleStart(feed.id)}
                          disabled={starting}
                        >
                          <Play className="h-4 w-4 mr-2" />
                          Start
                        </DropdownMenuItem>
                      ) : (
                        <DropdownMenuItem
                          onClick={() => handleStop(feed.id)}
                          disabled={stopping}
                        >
                          <Square className="h-4 w-4 mr-2" />
                          Stop
                        </DropdownMenuItem>
                      )}
                      {feed.is_active && (
                        <DropdownMenuItem
                          onClick={() => handleWatch(feed.id, feed.name)}
                          disabled={loadingView}
                        >
                          <Eye className="h-4 w-4 mr-2" />
                          Watch
                        </DropdownMenuItem>
                      )}
                      <DropdownMenuItem
                        onClick={() => handleDelete(feed.id)}
                        disabled={deleting}
                        className="text-destructive"
                      >
                        <Trash2 className="h-4 w-4 mr-2" />
                        Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      )}

      {/* Add Feed Dialog */}
      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Video Feed</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleAddFeed} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="feed-name">Name</Label>
              <Input
                id="feed-name"
                value={feedName}
                onChange={(e) => setFeedName(e.target.value)}
                placeholder="Feed name"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="feed-type">Type</Label>
              <select
                id="feed-type"
                value={feedType}
                onChange={(e) =>
                  setFeedType(
                    e.target.value as "rtsp" | "rtmp" | "whip" | "phone_cam"
                  )
                }
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <option value="rtsp">RTSP</option>
                <option value="rtmp">RTMP</option>
                <option value="whip">WHIP</option>
                <option value="phone_cam">Phone Camera</option>
              </select>
            </div>
            {needsSourceUrl && (
              <div className="space-y-2">
                <Label htmlFor="feed-source">Source URL</Label>
                <Input
                  id="feed-source"
                  value={sourceUrl}
                  onChange={(e) => setSourceUrl(e.target.value)}
                  placeholder={
                    feedType === "rtsp"
                      ? "rtsp://camera-ip/stream"
                      : "rtmp://server/live/stream"
                  }
                  required
                />
              </div>
            )}
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setAddOpen(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={creating}>
                {creating ? "Creating..." : "Add Feed"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Watch Feed Dialog */}
      <Dialog
        open={!!watchData}
        onOpenChange={(open) => !open && setWatchData(null)}
      >
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Watching: {watchData?.name}</DialogTitle>
          </DialogHeader>
          {watchData && (
            <FeedViewer
              token={watchData.token}
              serverUrl={watchData.url}
              feedName={watchData.name}
            />
          )}
        </DialogContent>
      </Dialog>
    </Card>
  );
}

// ---------------------------------------------------------------------------
// PTT Channels Section
// ---------------------------------------------------------------------------

function PTTSection({ groupId }: { groupId: string }) {
  const { channels, isLoading, refetch } = usePTTChannels(groupId);
  const { createChannel, isLoading: creating } = useCreatePTTChannel();
  const { joinChannel, isLoading: joining } = useJoinPTTChannel();
  const { deleteChannel, isLoading: deleting } = useDeletePTTChannel();

  const [expanded, setExpanded] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [channelName, setChannelName] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [activePTT, setActivePTT] = useState<{
    channelId: string;
    channelName: string;
    token: string;
    url: string;
  } | null>(null);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createChannel(groupId, {
        name: channelName,
        is_default: isDefault,
      });
      toast.success("PTT channel created");
      setDialogOpen(false);
      setChannelName("");
      setIsDefault(false);
      refetch();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to create channel"
      );
    }
  }

  async function handleJoin(channelId: string, name: string) {
    try {
      const resp = await joinChannel(groupId, channelId);
      setActivePTT({
        channelId,
        channelName: name,
        token: resp.token,
        url: resp.url || LIVEKIT_URL,
      });
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to join channel"
      );
    }
  }

  async function handleDelete(channelId: string) {
    if (!confirm("Delete this PTT channel?")) return;
    try {
      await deleteChannel(groupId, channelId);
      toast.success("PTT channel deleted");
      if (activePTT?.channelId === channelId) {
        setActivePTT(null);
      }
      refetch();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to delete channel"
      );
    }
  }

  return (
    <Card>
      <CardHeader
        className="cursor-pointer select-none"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {expanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <Radio className="h-4 w-4" />
            <CardTitle className="text-base">PTT Channels</CardTitle>
            <Badge variant="secondary">{channels.length}</Badge>
          </div>
          <Button
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setDialogOpen(true);
            }}
          >
            <Plus className="h-4 w-4 mr-1" />
            Create Channel
          </Button>
        </div>
      </CardHeader>

      {expanded && (
        <CardContent className="space-y-3">
          {activePTT && (
            <div className="border rounded-lg p-4">
              <PTTButton
                token={activePTT.token}
                serverUrl={activePTT.url}
                channelId={activePTT.channelId}
                channelName={activePTT.channelName}
              />
            </div>
          )}

          {isLoading ? (
            <p className="text-sm text-muted-foreground">
              Loading PTT channels...
            </p>
          ) : channels.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No PTT channels in this group.
            </p>
          ) : (
            <div className="space-y-2">
              {channels.map((ch) => (
                <div
                  key={ch.id}
                  className="flex items-center justify-between rounded-md border px-3 py-2"
                >
                  <div className="flex items-center gap-2">
                    <Mic className="h-4 w-4 text-muted-foreground" />
                    <span className="text-sm font-medium">{ch.name}</span>
                    {ch.is_default && <Badge variant="default">Default</Badge>}
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={joining || activePTT?.channelId === ch.id}
                      onClick={() => handleJoin(ch.id, ch.name)}
                    >
                      Join
                    </Button>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                        >
                          <MoreVertical className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          onClick={() => handleDelete(ch.id)}
                          disabled={deleting}
                          className="text-destructive"
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      )}

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create PTT Channel</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="ptt-name">Name</Label>
              <Input
                id="ptt-name"
                value={channelName}
                onChange={(e) => setChannelName(e.target.value)}
                placeholder="Channel name"
                required
              />
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="ptt-default"
                checked={isDefault}
                onCheckedChange={(checked) => setIsDefault(checked === true)}
              />
              <Label htmlFor="ptt-default">Default channel</Label>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setDialogOpen(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={creating}>
                {creating ? "Creating..." : "Create Channel"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

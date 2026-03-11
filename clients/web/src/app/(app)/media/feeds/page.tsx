"use client";

import { useState, useCallback, useEffect } from "react";
import { useAuth } from "@/lib/auth-context";
import {
  useGroupFeeds,
  useCreateFeed,
  useStartFeed,
  useStopFeed,
  useDeleteFeed,
  useViewFeed,
} from "@/lib/hooks/use-feeds";
import { FeedViewer } from "@/components/media/feed-viewer";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Video,
  Plus,
  MoreVertical,
  Play,
  Square,
  Trash2,
  Copy,
  Rss,
} from "lucide-react";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import type {
  Group,
  VideoFeed,
  CreateVideoFeedRequest,
  VideoFeedStartResponse,
} from "@/types/api";

export default function FeedsPage() {
  useAuth(); // ensure authenticated
  const { createFeed, isLoading: creatingFeed } = useCreateFeed();
  const { startFeed, isLoading: startingFeed } = useStartFeed();
  const { stopFeed, isLoading: stoppingFeed } = useStopFeed();
  const { deleteFeed, isLoading: deletingFeed } = useDeleteFeed();
  const { viewFeed, isLoading: viewingFeed } = useViewFeed();

  const [groups, setGroups] = useState<Group[]>([]);
  const [selectedGroupId, setSelectedGroupId] = useState<string>("");
  const [addFeedOpen, setAddFeedOpen] = useState(false);
  const [watchFeedOpen, setWatchFeedOpen] = useState(false);
  const [ingestInfo, setIngestInfo] = useState<VideoFeedStartResponse | null>(
    null
  );
  const [ingestDialogOpen, setIngestDialogOpen] = useState(false);

  // Watch state
  const [watchToken, setWatchToken] = useState("");
  const [watchUrl, setWatchUrl] = useState("");
  const [watchFeedName, setWatchFeedName] = useState("");

  // New feed form state
  const [newFeedName, setNewFeedName] = useState("");
  const [newFeedType, setNewFeedType] = useState<
    "rtsp" | "rtmp" | "whip" | "phone_cam"
  >("rtsp");
  const [newFeedSourceUrl, setNewFeedSourceUrl] = useState("");
  const [newFeedGroupId, setNewFeedGroupId] = useState("");

  // Fetch user's groups
  useEffect(() => {
    api
      .get<Group[]>("/api/v1/users/me/groups")
      .then((g) => {
        setGroups(g);
        if (g.length > 0 && !selectedGroupId) {
          setSelectedGroupId(g[0].id);
        }
      })
      .catch(() => {});
  }, [selectedGroupId]);

  // Fetch feeds for selected group
  const {
    feeds,
    isLoading: feedsLoading,
    refetch: refetchFeeds,
  } = useGroupFeeds(selectedGroupId);

  const handleAddFeed = useCallback(async () => {
    if (!newFeedName || !newFeedGroupId) {
      toast.error("Name and group are required");
      return;
    }
    try {
      const req: CreateVideoFeedRequest = {
        name: newFeedName,
        feed_type: newFeedType,
        group_id: newFeedGroupId,
      };
      if (newFeedType !== "phone_cam" && newFeedSourceUrl) {
        req.source_url = newFeedSourceUrl;
      }
      await createFeed(req);
      setAddFeedOpen(false);
      setNewFeedName("");
      setNewFeedSourceUrl("");
      setNewFeedType("rtsp");
      setNewFeedGroupId("");
      refetchFeeds();
      toast("Feed created");
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to create feed"
      );
    }
  }, [
    createFeed,
    newFeedName,
    newFeedType,
    newFeedSourceUrl,
    newFeedGroupId,
    refetchFeeds,
  ]);

  const handleStartFeed = useCallback(
    async (feedId: string) => {
      try {
        const resp = await startFeed(feedId);
        refetchFeeds();
        toast("Feed started");
        // Show ingest info if available
        if (resp.ingest_url || resp.stream_key) {
          setIngestInfo(resp);
          setIngestDialogOpen(true);
        }
      } catch (err) {
        toast.error(
          err instanceof ApiError ? err.message : "Failed to start feed"
        );
      }
    },
    [startFeed, refetchFeeds]
  );

  const handleStopFeed = useCallback(
    async (feedId: string) => {
      try {
        await stopFeed(feedId);
        refetchFeeds();
        toast("Feed stopped");
      } catch (err) {
        toast.error(
          err instanceof ApiError ? err.message : "Failed to stop feed"
        );
      }
    },
    [stopFeed, refetchFeeds]
  );

  const handleDeleteFeed = useCallback(
    async (feedId: string) => {
      try {
        await deleteFeed(feedId);
        refetchFeeds();
        toast("Feed deleted");
      } catch (err) {
        toast.error(
          err instanceof ApiError ? err.message : "Failed to delete feed"
        );
      }
    },
    [deleteFeed, refetchFeeds]
  );

  const handleWatchFeed = useCallback(
    async (feed: VideoFeed) => {
      try {
        const resp = await viewFeed(feed.id);
        setWatchToken(resp.token);
        setWatchUrl(resp.url);
        setWatchFeedName(feed.name);
        setWatchFeedOpen(true);
      } catch (err) {
        toast.error(
          err instanceof ApiError ? err.message : "Failed to view feed"
        );
      }
    },
    [viewFeed]
  );

  const copyToClipboard = useCallback((text: string) => {
    navigator.clipboard.writeText(text).then(() => {
      toast("Copied to clipboard");
    });
  }, []);

  const livekitUrl =
    process.env.NEXT_PUBLIC_LIVEKIT_URL || "ws://localhost:7880";

  const feedTypeBadgeVariant = (
    type: string
  ): "default" | "secondary" | "outline" | "destructive" => {
    switch (type) {
      case "rtsp":
        return "default";
      case "rtmp":
        return "secondary";
      case "whip":
        return "outline";
      case "phone_cam":
        return "secondary";
      default:
        return "default";
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-3.5rem)]">
      {/* Top bar */}
      <div className="p-4 border-b flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold flex items-center gap-2">
            <Rss className="h-5 w-5" />
            Video Feeds
          </h1>
          {groups.length > 0 && (
            <select
              className="h-8 rounded-md border border-input bg-transparent px-2 text-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              value={selectedGroupId}
              onChange={(e) => setSelectedGroupId(e.target.value)}
            >
              {groups.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
            </select>
          )}
        </div>
        <Button
          size="sm"
          onClick={() => {
            setNewFeedGroupId(selectedGroupId);
            setAddFeedOpen(true);
          }}
        >
          <Plus className="h-4 w-4 mr-1" />
          Add Feed
        </Button>
      </div>

      {/* Feed grid */}
      <div className="flex-1 overflow-auto p-4">
        {!selectedGroupId ? (
          <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
            No groups available. Join a group to manage feeds.
          </div>
        ) : feedsLoading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <Skeleton className="h-40" />
            <Skeleton className="h-40" />
            <Skeleton className="h-40" />
          </div>
        ) : feeds.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
            <Video className="h-12 w-12" />
            <p className="text-sm">No feeds in this group</p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setNewFeedGroupId(selectedGroupId);
                setAddFeedOpen(true);
              }}
            >
              <Plus className="h-4 w-4 mr-1" />
              Add Feed
            </Button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {feeds.map((feed) => (
              <Card key={feed.id}>
                <CardContent className="p-4">
                  <div className="flex items-start justify-between">
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium truncate">
                        {feed.name}
                      </p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {groups.find((g) => g.id === feed.group_id)?.name ??
                          "Group"}
                      </p>
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
                            onClick={() => handleStartFeed(feed.id)}
                            disabled={startingFeed}
                          >
                            <Play className="h-4 w-4 mr-2" />
                            Start
                          </DropdownMenuItem>
                        ) : (
                          <DropdownMenuItem
                            onClick={() => handleStopFeed(feed.id)}
                            disabled={stoppingFeed}
                          >
                            <Square className="h-4 w-4 mr-2" />
                            Stop
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuItem
                          onClick={() => handleDeleteFeed(feed.id)}
                          disabled={deletingFeed}
                          className="text-destructive"
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>

                  <div className="flex items-center gap-2 mt-3">
                    <Badge variant={feedTypeBadgeVariant(feed.feed_type)}>
                      {feed.feed_type}
                    </Badge>
                    <Badge
                      variant={feed.is_active ? "default" : "secondary"}
                      className={
                        feed.is_active ? "bg-green-600 hover:bg-green-700" : ""
                      }
                    >
                      {feed.is_active ? "Active" : "Inactive"}
                    </Badge>
                  </div>

                  {feed.is_active && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="w-full mt-3"
                      onClick={() => handleWatchFeed(feed)}
                      disabled={viewingFeed}
                    >
                      <Play className="h-4 w-4 mr-1" />
                      Watch
                    </Button>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* Add Feed Dialog */}
      <Dialog open={addFeedOpen} onOpenChange={setAddFeedOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Video Feed</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="feed-name">Feed Name</Label>
              <Input
                id="feed-name"
                placeholder="Front Gate Camera"
                value={newFeedName}
                onChange={(e) => setNewFeedName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="feed-type">Feed Type</Label>
              <select
                id="feed-type"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={newFeedType}
                onChange={(e) =>
                  setNewFeedType(
                    e.target.value as "rtsp" | "rtmp" | "whip" | "phone_cam"
                  )
                }
              >
                <option value="rtsp">RTSP</option>
                <option value="rtmp">RTMP</option>
                <option value="whip">WHIP</option>
                <option value="phone_cam">Phone Camera</option>
              </select>
            </div>
            {newFeedType !== "phone_cam" && (
              <div className="space-y-2">
                <Label htmlFor="feed-source">Source URL</Label>
                <Input
                  id="feed-source"
                  placeholder={
                    newFeedType === "rtsp"
                      ? "rtsp://camera.local:554/stream"
                      : newFeedType === "rtmp"
                        ? "rtmp://ingest.example.com/live"
                        : "https://whip.example.com/endpoint"
                  }
                  value={newFeedSourceUrl}
                  onChange={(e) => setNewFeedSourceUrl(e.target.value)}
                />
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="feed-group">Group</Label>
              <select
                id="feed-group"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={newFeedGroupId}
                onChange={(e) => setNewFeedGroupId(e.target.value)}
              >
                <option value="">Select group...</option>
                {groups.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAddFeedOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleAddFeed} disabled={creatingFeed}>
              {creatingFeed ? "Creating..." : "Create Feed"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Watch Feed Dialog */}
      <Dialog open={watchFeedOpen} onOpenChange={setWatchFeedOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Video className="h-4 w-4" />
              {watchFeedName}
            </DialogTitle>
          </DialogHeader>
          {watchToken && (
            <FeedViewer
              token={watchToken}
              serverUrl={watchUrl || livekitUrl}
              feedName={watchFeedName}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Ingest Info Dialog */}
      <Dialog open={ingestDialogOpen} onOpenChange={setIngestDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Feed Started</DialogTitle>
          </DialogHeader>
          {ingestInfo && (
            <div className="space-y-3 py-2">
              {ingestInfo.ingest_url && (
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">
                    Ingest URL
                  </Label>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 rounded bg-muted px-2 py-1 text-xs break-all">
                      {ingestInfo.ingest_url}
                    </code>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="shrink-0 h-8 w-8 p-0"
                      onClick={() => copyToClipboard(ingestInfo.ingest_url!)}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
              {ingestInfo.stream_key && (
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">
                    Stream Key
                  </Label>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 rounded bg-muted px-2 py-1 text-xs break-all">
                      {ingestInfo.stream_key}
                    </code>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="shrink-0 h-8 w-8 p-0"
                      onClick={() => copyToClipboard(ingestInfo.stream_key!)}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}
          <DialogFooter>
            <Button onClick={() => setIngestDialogOpen(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

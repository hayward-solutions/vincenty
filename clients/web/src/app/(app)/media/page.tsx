"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import Link from "next/link";
import { useWebSocket } from "@/lib/websocket-context";
import { useActiveCalls, useCreateCall, useJoinCall, useLeaveCall } from "@/lib/hooks/use-calls";
import { LiveSourceTile } from "@/components/media/live-source-tile";
import { Button } from "@/components/ui/button";
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
import { Checkbox } from "@/components/ui/checkbox";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Camera,
  Video,
  VideoOff,
  Plus,
  ChevronDown,
  ChevronRight,
  Eye,
  Phone,
  Rss,
} from "lucide-react";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import { cn } from "@/lib/utils";
import type {
  Group,
  VideoFeed,
  JoinRoomResponse,
  MediaRoom,
  WSCallEvent,
  WSFeedEvent,
} from "@/types/api";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface WatchedSource {
  id: string;
  sourceType: "feed" | "call";
  name: string;
  groupName: string;
  feedType?: string;
  token: string;
  serverUrl: string;
}

// ---------------------------------------------------------------------------
// Sidebar section (collapsible)
// ---------------------------------------------------------------------------

interface SidebarSectionProps {
  title: string;
  count: number;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

function SidebarSection({ title, count, defaultOpen = false, children }: SidebarSectionProps) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div>
      <button
        type="button"
        className="w-full flex items-center justify-between px-3 py-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground hover:bg-accent/50 transition-colors"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="flex items-center gap-1.5">
          {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
          {title}
        </span>
        <Badge variant="secondary" className="text-[10px] h-5 min-w-5 justify-center">
          {count}
        </Badge>
      </button>
      {open && <div className="px-2 pb-2">{children}</div>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Feed sidebar item
// ---------------------------------------------------------------------------

interface FeedItemProps {
  feed: VideoFeed;
  groupName: string;
  isWatching: boolean;
  onToggle: () => void;
}

function FeedItem({ feed, groupName, isWatching, onToggle }: FeedItemProps) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors",
        isWatching ? "bg-primary/10 text-primary" : "hover:bg-accent"
      )}
    >
      <Camera className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{feed.name}</p>
        <p className="truncate text-xs text-muted-foreground">{groupName}</p>
      </div>
      <Button
        variant={isWatching ? "secondary" : "ghost"}
        size="sm"
        className="h-7 shrink-0 text-xs"
        onClick={onToggle}
      >
        <Eye className="h-3 w-3 mr-1" />
        {isWatching ? "Watching" : "Watch"}
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Call sidebar item
// ---------------------------------------------------------------------------

interface CallItemProps {
  call: MediaRoom;
  groupName: string;
  isWatching: boolean;
  onToggle: () => void;
}

function CallItem({ call, groupName, isWatching, onToggle }: CallItemProps) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors",
        isWatching ? "bg-primary/10 text-primary" : "hover:bg-accent"
      )}
    >
      {call.room_type === "call" ? (
        <Video className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      ) : (
        <Phone className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      )}
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{call.name}</p>
        <p className="truncate text-xs text-muted-foreground">{groupName}</p>
      </div>
      <Button
        variant={isWatching ? "secondary" : "ghost"}
        size="sm"
        className="h-7 shrink-0 text-xs"
        onClick={onToggle}
      >
        {isWatching ? "Joined" : "Join"}
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

const LIVEKIT_URL = process.env.NEXT_PUBLIC_LIVEKIT_URL || "ws://localhost:7880";

export default function MediaPage() {
  const { subscribe } = useWebSocket();
  const { calls, isLoading: callsLoading, refetch: refetchCalls } = useActiveCalls();
  const { createCall, isLoading: creating } = useCreateCall();
  const { joinCall } = useJoinCall();
  const { leaveCall } = useLeaveCall();

  // Groups + feeds state
  const [groups, setGroups] = useState<Group[]>([]);
  const [groupsLoading, setGroupsLoading] = useState(true);
  const [feedsByGroup, setFeedsByGroup] = useState<Record<string, VideoFeed[]>>({});
  const [feedsLoading, setFeedsLoading] = useState(false);

  // Grid state
  const [watchedSources, setWatchedSources] = useState<WatchedSource[]>([]);
  const [expandedSource, setExpandedSource] = useState<string | null>(null);

  // New call dialog state
  const [newCallOpen, setNewCallOpen] = useState(false);
  const [callName, setCallName] = useState("");
  const [callGroupId, setCallGroupId] = useState("");
  const [callVideoEnabled, setCallVideoEnabled] = useState(true);

  // Track watched source IDs for stable reference in callbacks
  const watchedIdsRef = useRef(new Set<string>());
  useEffect(() => {
    watchedIdsRef.current = new Set(watchedSources.map((s) => s.id));
  }, [watchedSources]);

  // -----------------------------------------------------------------------
  // Fetch groups on mount, then fetch feeds for each group
  // -----------------------------------------------------------------------

  useEffect(() => {
    let cancelled = false;

    async function loadGroupsAndFeeds() {
      setGroupsLoading(true);
      try {
        const loadedGroups = await api.get<Group[]>("/api/v1/users/me/groups");
        if (cancelled) return;
        setGroups(loadedGroups);

        if (loadedGroups.length === 0) {
          setGroupsLoading(false);
          return;
        }

        setFeedsLoading(true);
        const feedMap: Record<string, VideoFeed[]> = {};
        await Promise.all(
          loadedGroups.map(async (group) => {
            try {
              const feeds = await api.get<VideoFeed[]>(`/api/v1/groups/${group.id}/feeds`);
              feedMap[group.id] = feeds;
            } catch {
              feedMap[group.id] = [];
            }
          })
        );
        if (cancelled) return;
        setFeedsByGroup(feedMap);
      } catch {
        // Groups failed to load — fail silently
      } finally {
        if (!cancelled) {
          setGroupsLoading(false);
          setFeedsLoading(false);
        }
      }
    }

    loadGroupsAndFeeds();
    return () => { cancelled = true; };
  }, []);

  // -----------------------------------------------------------------------
  // Helper: find group name by ID
  // -----------------------------------------------------------------------

  const groupName = useCallback(
    (groupId?: string) => {
      if (!groupId) return "Unknown";
      return groups.find((g) => g.id === groupId)?.name ?? "Unknown";
    },
    [groups]
  );

  // -----------------------------------------------------------------------
  // Refresh feeds for a specific group
  // -----------------------------------------------------------------------

  const refreshGroupFeeds = useCallback(async (groupId: string) => {
    try {
      const feeds = await api.get<VideoFeed[]>(`/api/v1/groups/${groupId}/feeds`);
      setFeedsByGroup((prev) => ({ ...prev, [groupId]: feeds }));
    } catch {
      // Silently ignore
    }
  }, []);

  // -----------------------------------------------------------------------
  // WebSocket: listen for feed/call events
  // -----------------------------------------------------------------------

  useEffect(() => {
    const unsubscribe = subscribe((type, payload) => {
      if (type === "feed_started") {
        const evt = payload as WSFeedEvent;
        refreshGroupFeeds(evt.group_id);
      }

      if (type === "feed_stopped") {
        const evt = payload as WSFeedEvent;
        refreshGroupFeeds(evt.group_id);

        // Remove from grid if watching
        if (watchedIdsRef.current.has(evt.feed_id)) {
          setWatchedSources((prev) => prev.filter((s) => s.id !== evt.feed_id));
          toast(`Feed stopped: ${evt.feed_name}`);
        }
      }

      if (type === "call_started" || type === "call_ended") {
        refetchCalls();

        if (type === "call_ended") {
          const evt = payload as WSCallEvent;
          if (watchedIdsRef.current.has(evt.room_id)) {
            setWatchedSources((prev) => prev.filter((s) => s.id !== evt.room_id));
            toast(`Call ended: ${evt.room_name}`);
          }
        }
      }
    });

    return unsubscribe;
  }, [subscribe, refetchCalls, refreshGroupFeeds]);

  // -----------------------------------------------------------------------
  // Toggle feed in/out of the monitoring grid
  // -----------------------------------------------------------------------

  const toggleFeed = useCallback(
    async (feed: VideoFeed) => {
      // If already watching, remove
      if (watchedIdsRef.current.has(feed.id)) {
        setWatchedSources((prev) => prev.filter((s) => s.id !== feed.id));
        setExpandedSource((prev) => (prev === feed.id ? null : prev));
        return;
      }

      // Get a subscribe-only token
      try {
        const resp = await api.get<JoinRoomResponse>(`/api/v1/feeds/${feed.id}/view`);
        setWatchedSources((prev) => [
          ...prev,
          {
            id: feed.id,
            sourceType: "feed",
            name: feed.name,
            groupName: groupName(feed.group_id),
            feedType: feed.feed_type,
            token: resp.token,
            serverUrl: resp.url || LIVEKIT_URL,
          },
        ]);
      } catch (err) {
        toast.error(err instanceof ApiError ? err.message : "Failed to view feed");
      }
    },
    [groupName]
  );

  // -----------------------------------------------------------------------
  // Toggle call in/out of the monitoring grid
  // -----------------------------------------------------------------------

  const toggleCall = useCallback(
    async (call: MediaRoom) => {
      // If already watching, leave and remove
      if (watchedIdsRef.current.has(call.id)) {
        setWatchedSources((prev) => prev.filter((s) => s.id !== call.id));
        setExpandedSource((prev) => (prev === call.id ? null : prev));
        try {
          await leaveCall(call.id);
        } catch {
          // Ignore leave errors
        }
        return;
      }

      try {
        const resp = await joinCall(call.id);
        setWatchedSources((prev) => [
          ...prev,
          {
            id: call.id,
            sourceType: "call",
            name: call.name,
            groupName: groupName(call.group_id),
            token: resp.token,
            serverUrl: resp.url || LIVEKIT_URL,
          },
        ]);
      } catch (err) {
        toast.error(err instanceof ApiError ? err.message : "Failed to join call");
      }
    },
    [joinCall, leaveCall, groupName]
  );

  // -----------------------------------------------------------------------
  // Remove source from grid
  // -----------------------------------------------------------------------

  const removeSource = useCallback(
    async (source: WatchedSource) => {
      setWatchedSources((prev) => prev.filter((s) => s.id !== source.id));
      setExpandedSource((prev) => (prev === source.id ? null : prev));

      // Leave call if it was a call source
      if (source.sourceType === "call") {
        try {
          await leaveCall(source.id);
        } catch {
          // Ignore
        }
      }
    },
    [leaveCall]
  );

  // -----------------------------------------------------------------------
  // Expand / collapse tile
  // -----------------------------------------------------------------------

  const toggleExpand = useCallback((id: string) => {
    setExpandedSource((prev) => (prev === id ? null : id));
  }, []);

  // -----------------------------------------------------------------------
  // Create call
  // -----------------------------------------------------------------------

  const handleCreateCall = useCallback(async () => {
    try {
      const resp = await createCall({
        name: callName || undefined,
        group_id: callGroupId || undefined,
        video_enabled: callVideoEnabled,
      });

      // Add to grid immediately
      setWatchedSources((prev) => [
        ...prev,
        {
          id: resp.room.id,
          sourceType: "call",
          name: resp.room.name,
          groupName: groupName(resp.room.group_id),
          token: resp.token,
          serverUrl: resp.url || LIVEKIT_URL,
        },
      ]);

      setNewCallOpen(false);
      setCallName("");
      setCallGroupId("");
      setCallVideoEnabled(true);
      refetchCalls();
      toast("Call started");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to create call");
    }
  }, [createCall, callName, callGroupId, callVideoEnabled, refetchCalls, groupName]);

  // -----------------------------------------------------------------------
  // Derived data
  // -----------------------------------------------------------------------

  const allFeeds = Object.values(feedsByGroup).flat();
  const activeFeeds = allFeeds.filter((f) => f.is_active);
  const watchedIds = new Set(watchedSources.map((s) => s.id));

  // -----------------------------------------------------------------------
  // Render
  // -----------------------------------------------------------------------

  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      {/* ----------------------------------------------------------------- */}
      {/* Left sidebar                                                      */}
      {/* ----------------------------------------------------------------- */}
      <div
        className={cn(
          "w-full md:w-[300px] shrink-0 border-r flex flex-col bg-background",
          watchedSources.length > 0 ? "hidden md:flex" : "flex"
        )}
      >
        {/* Sidebar header */}
        <div className="p-3 border-b flex items-center justify-between">
          <h2 className="text-sm font-semibold flex items-center gap-1.5">
            <Rss className="h-4 w-4" />
            Sources
            <Badge variant="secondary" className="text-[10px] ml-1">
              {activeFeeds.length + calls.length}
            </Badge>
          </h2>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <Plus className="h-4 w-4 mr-1" />
                Add Source
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem asChild>
                <Link href="/media/feeds">
                  <Camera className="h-4 w-4 mr-2" />
                  Add Video Feed
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setNewCallOpen(true)}>
                <Phone className="h-4 w-4 mr-2" />
                Start Call
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <ScrollArea className="flex-1">
          {/* Video Feeds section */}
          <SidebarSection
            title="Video Feeds"
            count={activeFeeds.length}
            defaultOpen
          >
            {groupsLoading || feedsLoading ? (
              <div className="space-y-2 px-1">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : activeFeeds.length === 0 ? (
              <p className="text-xs text-muted-foreground text-center py-4 px-2">
                No active feeds in your groups
              </p>
            ) : (
              <div className="space-y-0.5">
                {groups.map((group) => {
                  const groupFeeds = (feedsByGroup[group.id] ?? []).filter((f) => f.is_active);
                  if (groupFeeds.length === 0) return null;
                  return groupFeeds.map((feed) => (
                    <FeedItem
                      key={feed.id}
                      feed={feed}
                      groupName={group.name}
                      isWatching={watchedIds.has(feed.id)}
                      onToggle={() => toggleFeed(feed)}
                    />
                  ));
                })}
              </div>
            )}
          </SidebarSection>

          {/* Active Calls section */}
          <SidebarSection
            title="Active Calls"
            count={calls.length}
          >
            {callsLoading ? (
              <div className="space-y-2 px-1">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : calls.length === 0 ? (
              <p className="text-xs text-muted-foreground text-center py-4 px-2">
                No active calls
              </p>
            ) : (
              <div className="space-y-0.5">
                {calls.map((call) => (
                  <CallItem
                    key={call.id}
                    call={call}
                    groupName={groupName(call.group_id)}
                    isWatching={watchedIds.has(call.id)}
                    onToggle={() => toggleCall(call)}
                  />
                ))}
              </div>
            )}
          </SidebarSection>
        </ScrollArea>
      </div>

      {/* ----------------------------------------------------------------- */}
      {/* Main grid area                                                    */}
      {/* ----------------------------------------------------------------- */}
      <div
        className={cn(
          "flex-1 flex flex-col min-w-0 bg-muted/30",
          watchedSources.length > 0 ? "flex" : "hidden md:flex"
        )}
      >
        {/* Mobile back button when viewing sources */}
        {watchedSources.length > 0 && (
          <div className="md:hidden p-2 border-b">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setWatchedSources([])}
            >
              <ChevronRight className="h-4 w-4 mr-1 rotate-180" />
              Back to sources
            </Button>
          </div>
        )}

        {watchedSources.length === 0 ? (
          /* Empty state */
          <div className="flex flex-col items-center justify-center h-full gap-4 text-muted-foreground">
            <Camera className="h-12 w-12" />
            <div className="text-center">
              <p className="text-lg font-medium">No sources being monitored</p>
              <p className="text-sm mt-1">
                Select sources from the sidebar or add new feeds
              </p>
            </div>
            <Button asChild variant="outline">
              <Link href="/media/feeds">
                <Plus className="h-4 w-4 mr-1" />
                Manage Feeds
              </Link>
            </Button>
          </div>
        ) : (
          /* Video grid */
          <div className="flex-1 overflow-auto p-3">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3 auto-rows-min">
              {watchedSources.map((source) => (
                <LiveSourceTile
                  key={source.id}
                  sourceType={source.sourceType}
                  name={source.name}
                  groupName={source.groupName}
                  feedType={source.feedType}
                  token={source.token}
                  serverUrl={source.serverUrl}
                  isExpanded={expandedSource === source.id}
                  onToggleExpand={() => toggleExpand(source.id)}
                  onRemove={() => removeSource(source)}
                />
              ))}
            </div>
          </div>
        )}
      </div>

      {/* ----------------------------------------------------------------- */}
      {/* Start Call Dialog                                                  */}
      {/* ----------------------------------------------------------------- */}
      <Dialog open={newCallOpen} onOpenChange={setNewCallOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Start a New Call</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="call-name">Call Name</Label>
              <Input
                id="call-name"
                placeholder="My Call"
                value={callName}
                onChange={(e) => setCallName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="call-group">Group (optional)</Label>
              <select
                id="call-group"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={callGroupId}
                onChange={(e) => setCallGroupId(e.target.value)}
              >
                <option value="">No group</option>
                {groups.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="call-video"
                checked={callVideoEnabled}
                onCheckedChange={(checked) =>
                  setCallVideoEnabled(checked === true)
                }
              />
              <Label htmlFor="call-video" className="flex items-center gap-1.5">
                {callVideoEnabled ? (
                  <Video className="h-4 w-4" />
                ) : (
                  <VideoOff className="h-4 w-4" />
                )}
                Enable video
              </Label>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setNewCallOpen(false)}
            >
              Cancel
            </Button>
            <Button onClick={handleCreateCall} disabled={creating}>
              {creating ? "Starting..." : "Start Call"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

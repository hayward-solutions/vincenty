"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import maplibregl from "maplibre-gl";
import { toast } from "sonner";
import { MapView } from "@/components/map/map-view";
import { LocationMarkers } from "@/components/map/location-markers";
import { SelfMarker } from "@/components/map/self-marker";
import { GpxOverlay } from "@/components/map/gpx-overlay";
import { HistoryTracks } from "@/components/map/history-tracks";
import { ReplayControls } from "@/components/map/replay-controls";
import {
  ReplayPanel,
  type ReplayScope,
  type ReplayStartParams,
} from "@/components/map/replay-panel";
import { useMapSettings } from "@/lib/hooks/use-map-settings";
import { useLocations } from "@/lib/hooks/use-locations";
import { useLocationSharing } from "@/lib/hooks/use-location-sharing";
import {
  useAllLocations,
  useVisibleHistory,
  useLocationHistory,
  useUserLocationHistory,
  useMyGroups,
} from "@/lib/hooks/use-location-history";
import { useAuth } from "@/lib/auth-context";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type {
  MessageResponse,
  LocationHistoryEntry,
  GroupMember,
} from "@/types/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

export default function MapPage() {
  const { user, isAdmin } = useAuth();
  const { settings, isLoading, error } = useMapSettings();
  const { locations } = useLocations();
  const { lastPosition } = useLocationSharing();
  const { data: adminLocations, fetchAll } = useAllLocations();

  // Three history hooks — one per scope
  const {
    data: visibleHistoryData,
    isLoading: visibleHistoryLoading,
    fetchHistory: fetchVisibleHistory,
    clear: clearVisibleHistory,
  } = useVisibleHistory();
  const {
    data: groupHistoryData,
    isLoading: groupHistoryLoading,
    fetchHistory: fetchGroupHistory,
    clear: clearGroupHistory,
  } = useLocationHistory();
  const {
    data: userHistoryData,
    isLoading: userHistoryLoading,
    fetchHistory: fetchUserHistory,
    clear: clearUserHistory,
  } = useUserLocationHistory();

  const { groups: myGroups } = useMyGroups();
  const searchParams = useSearchParams();

  const mapRef = useRef<maplibregl.Map | null>(null);

  // GPX overlay support: ?gpx=<messageId>
  const gpxMessageId = searchParams.get("gpx");
  const [gpxMessage, setGpxMessage] = useState<MessageResponse | null>(null);

  // Replay state
  const [replayActive, setReplayActive] = useState(false);
  const [replayPanelOpen, setReplayPanelOpen] = useState(false);
  const [replayScope, setReplayScope] = useState<ReplayScope>("all");
  const [replayRange, setReplayRange] = useState<{
    from: Date;
    to: Date;
  } | null>(null);
  const [playbackTime, setPlaybackTime] = useState<Date | undefined>(undefined);

  // Client-side filter state (narrowing during replay)
  const [selectedGroupIds, setSelectedGroupIds] = useState<Set<string>>(
    new Set()
  );
  const [selectedUserIds, setSelectedUserIds] = useState<Set<string>>(
    new Set()
  );
  const [groupMemberCache, setGroupMemberCache] = useState<
    Map<string, string[]>
  >(new Map());

  useEffect(() => {
    if (!gpxMessageId) {
      setGpxMessage(null);
      return;
    }
    api
      .get<MessageResponse>(`/api/v1/messages/${gpxMessageId}`)
      .then(setGpxMessage)
      .catch((err) => {
        console.error("Failed to load GPX message:", err);
        setGpxMessage(null);
      });
  }, [gpxMessageId]);

  const handleMapReady = useCallback(
    (map: maplibregl.Map) => {
      mapRef.current = map;
      if (isAdmin) {
        fetchAll();
      }
    },
    [isAdmin, fetchAll]
  );

  // ---------------------------------------------------------------------------
  // Derive active history from the current replay scope
  // ---------------------------------------------------------------------------

  const activeHistory = useMemo((): LocationHistoryEntry[] => {
    switch (replayScope) {
      case "group":
        return groupHistoryData;
      case "user":
        return userHistoryData;
      default:
        return visibleHistoryData;
    }
  }, [replayScope, groupHistoryData, userHistoryData, visibleHistoryData]);

  const activeHistoryLoading =
    replayScope === "group"
      ? groupHistoryLoading
      : replayScope === "user"
        ? userHistoryLoading
        : visibleHistoryLoading;

  // Extract unique users from active history for the user filter sidebar
  const visibleUsers = useMemo(() => {
    const map = new Map<
      string,
      { user_id: string; username: string; display_name: string }
    >();
    for (const entry of activeHistory) {
      if (!map.has(entry.user_id)) {
        map.set(entry.user_id, {
          user_id: entry.user_id,
          username: entry.username,
          display_name: entry.display_name,
        });
      }
    }
    return Array.from(map.values()).sort((a, b) =>
      (a.display_name || a.username).localeCompare(
        b.display_name || b.username
      )
    );
  }, [activeHistory]);

  // Fetch group members when a group is toggled in the filter sidebar
  const fetchGroupMembers = useCallback(
    async (groupId: string) => {
      if (groupMemberCache.has(groupId)) return;
      try {
        const members = await api.get<GroupMember[]>(
          `/api/v1/groups/${groupId}/members`
        );
        setGroupMemberCache((prev) => {
          const next = new Map(prev);
          next.set(
            groupId,
            members.map((m) => m.user_id)
          );
          return next;
        });
      } catch {
        // Ignore — group filter just won't narrow
      }
    },
    [groupMemberCache]
  );

  // Compute filtered history based on client-side filter sidebar selections
  const filteredHistory = useMemo(() => {
    if (selectedGroupIds.size === 0 && selectedUserIds.size === 0) {
      return activeHistory;
    }

    // Build allowed user_ids from group selections
    let allowedByGroup: Set<string> | null = null;
    if (selectedGroupIds.size > 0) {
      allowedByGroup = new Set<string>();
      for (const gid of selectedGroupIds) {
        const members = groupMemberCache.get(gid);
        if (members) {
          for (const uid of members) {
            allowedByGroup.add(uid);
          }
        }
      }
    }

    return activeHistory.filter((entry) => {
      if (allowedByGroup != null && !allowedByGroup.has(entry.user_id)) {
        return false;
      }
      if (selectedUserIds.size > 0 && !selectedUserIds.has(entry.user_id)) {
        return false;
      }
      return true;
    });
  }, [activeHistory, selectedGroupIds, selectedUserIds, groupMemberCache]);

  // ---------------------------------------------------------------------------
  // Replay lifecycle
  // ---------------------------------------------------------------------------

  const handleReplayStart = useCallback(
    (params: ReplayStartParams) => {
      // Clear all hooks
      clearVisibleHistory();
      clearGroupHistory();
      clearUserHistory();

      // Reset client-side filters
      setSelectedGroupIds(new Set());
      setSelectedUserIds(new Set());
      setPlaybackTime(undefined);

      // Store scope info
      setReplayScope(params.scope);
      setReplayRange({ from: params.from, to: params.to });

      // Fetch from the right endpoint
      switch (params.scope) {
        case "group":
          if (params.groupId)
            fetchGroupHistory(params.groupId, params.from, params.to);
          break;
        case "user":
          if (params.userId)
            fetchUserHistory(params.userId, params.from, params.to);
          break;
        default:
          fetchVisibleHistory(params.from, params.to);
      }

      setReplayActive(true);
      setReplayPanelOpen(false);
    },
    [
      clearVisibleHistory,
      clearGroupHistory,
      clearUserHistory,
      fetchVisibleHistory,
      fetchGroupHistory,
      fetchUserHistory,
    ]
  );

  const stopReplay = useCallback(() => {
    setReplayActive(false);
    setPlaybackTime(undefined);
    setReplayScope("all");
    setReplayRange(null);
    setSelectedGroupIds(new Set());
    setSelectedUserIds(new Set());
    clearVisibleHistory();
    clearGroupHistory();
    clearUserHistory();
  }, [clearVisibleHistory, clearGroupHistory, clearUserHistory]);

  const handleExportGPX = useCallback(
    async (from: Date, to: Date, userId?: string) => {
      try {
        const params = new URLSearchParams({
          from: from.toISOString(),
          to: to.toISOString(),
        });
        const path = userId
          ? `/api/v1/users/${userId}/locations/export`
          : "/api/v1/users/me/locations/export";
        const url = `${API_BASE}${path}?${params}`;
        const token = localStorage.getItem("access_token");
        const res = await window.fetch(url, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (!res.ok) throw new Error("Export failed");
        const blob = await res.blob();
        const a = document.createElement("a");
        a.href = URL.createObjectURL(blob);
        a.download = "track.gpx";
        document.body.appendChild(a);
        a.click();
        a.remove();
        URL.revokeObjectURL(a.href);
        toast.success("GPX track exported");
      } catch {
        toast.error("Export failed");
      }
    },
    []
  );

  // ---------------------------------------------------------------------------
  // Client-side filter sidebar toggles
  // ---------------------------------------------------------------------------

  function toggleGroup(groupId: string) {
    setSelectedGroupIds((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) {
        next.delete(groupId);
      } else {
        next.add(groupId);
        fetchGroupMembers(groupId);
      }
      return next;
    });
  }

  function toggleUser(userId: string) {
    setSelectedUserIds((prev) => {
      const next = new Set(prev);
      if (next.has(userId)) {
        next.delete(userId);
      } else {
        next.add(userId);
      }
      return next;
    });
  }

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-3.5rem)]">
        <Skeleton className="h-full w-full" />
      </div>
    );
  }

  if (error || !settings) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-3.5rem)]">
        <p className="text-muted-foreground">
          {error || "Failed to load map settings"}
        </p>
      </div>
    );
  }

  // Merge admin-fetched locations into the WS-provided locations.
  const displayLocations = new Map(locations);
  if (isAdmin && adminLocations.length > 0) {
    for (const loc of adminLocations) {
      if (!displayLocations.has(loc.user_id)) {
        displayLocations.set(loc.user_id, {
          user_id: loc.user_id,
          username: loc.username,
          display_name: loc.display_name,
          group_id: "",
          lat: loc.lat,
          lng: loc.lng,
          altitude: loc.altitude,
          heading: loc.heading,
          speed: loc.speed,
          timestamp: loc.recorded_at,
        });
      }
    }
  }

  // Show group checkboxes in the filter sidebar only for "all" scope
  const showGroupFilter = replayScope === "all" && myGroups.length > 0;

  return (
    <div className="relative h-[calc(100vh-3.5rem)]">
      <MapView settings={settings} onMapReady={handleMapReady}>
        {mapRef.current && (
          <>
            <SelfMarker
              map={mapRef.current}
              position={lastPosition}
              autoCenter={!gpxMessageId && !replayActive}
            />
            <LocationMarkers
              map={mapRef.current}
              locations={displayLocations}
              currentUserId={user?.id}
            />
            <GpxOverlay map={mapRef.current} message={gpxMessage} />
            {replayActive && filteredHistory.length > 0 && (
              <HistoryTracks
                map={mapRef.current}
                history={filteredHistory}
                playbackTime={playbackTime}
              />
            )}
          </>
        )}
      </MapView>

      {/* Replay button (top-left overlay) */}
      {!replayActive && !replayPanelOpen && (
        <div className="absolute top-3 left-3 z-10">
          <Button
            size="sm"
            variant="secondary"
            onClick={() => setReplayPanelOpen(true)}
          >
            Replay
          </Button>
        </div>
      )}

      {/* Replay setup panel */}
      {replayPanelOpen && !replayActive && (
        <ReplayPanel
          isAdmin={!!isAdmin}
          isLoading={activeHistoryLoading}
          onStart={handleReplayStart}
          onExportGPX={handleExportGPX}
          onCancel={() => setReplayPanelOpen(false)}
        />
      )}

      {/* Replay active: filter sidebar + controls */}
      {replayActive && (
        <>
          {/* Filter sidebar — show when there are tracks and something to filter */}
          {activeHistory.length > 0 &&
            (showGroupFilter || visibleUsers.length > 1) && (
              <div className="absolute top-3 left-3 z-10 bg-card/95 backdrop-blur-sm border rounded-lg p-3 shadow-lg w-56 max-h-[calc(100vh-8rem)] overflow-y-auto space-y-3">
                <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                  Filters
                </h4>

                {/* Group filter — only in "all" scope */}
                {showGroupFilter && (
                  <div className="space-y-1">
                    <p className="text-xs font-medium">Groups</p>
                    {myGroups.map((g) => (
                      <label
                        key={g.id}
                        className="flex items-center gap-2 text-sm cursor-pointer"
                      >
                        <input
                          type="checkbox"
                          checked={selectedGroupIds.has(g.id)}
                          onChange={() => toggleGroup(g.id)}
                          className="h-3.5 w-3.5"
                        />
                        <span className="truncate">{g.name}</span>
                      </label>
                    ))}
                  </div>
                )}

                {/* User filter */}
                {visibleUsers.length > 1 && (
                  <div className="space-y-1">
                    <p className="text-xs font-medium">Users</p>
                    {visibleUsers.map((u) => (
                      <label
                        key={u.user_id}
                        className="flex items-center gap-2 text-sm cursor-pointer"
                      >
                        <input
                          type="checkbox"
                          checked={selectedUserIds.has(u.user_id)}
                          onChange={() => toggleUser(u.user_id)}
                          className="h-3.5 w-3.5"
                        />
                        <span className="truncate">
                          {u.display_name || u.username}
                        </span>
                      </label>
                    ))}
                  </div>
                )}

                <p className="text-xs text-muted-foreground">
                  {filteredHistory.length} points
                  {visibleUsers.length > 0 &&
                    ` from ${new Set(filteredHistory.map((e) => e.user_id)).size} user(s)`}
                </p>
              </div>
            )}

          {/* Replay controls bar */}
          {filteredHistory.length > 0 && replayRange != null && (
            <ReplayControls
              from={replayRange.from}
              to={replayRange.to}
              onTimeChange={setPlaybackTime}
              onReset={stopReplay}
            />
          )}

          {/* No data message */}
          {activeHistory.length === 0 && !activeHistoryLoading && (
            <div className="absolute bottom-4 left-4 right-4 z-10 flex items-center justify-between bg-card/90 backdrop-blur-sm border rounded-lg px-4 py-3 shadow-lg">
              <span className="text-sm text-muted-foreground">
                No location data for the selected time range
              </span>
              <Button variant="ghost" size="sm" onClick={stopReplay}>
                Close
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

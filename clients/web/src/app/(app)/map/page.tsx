"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import maplibregl from "maplibre-gl";
import { toast } from "sonner";
import { MapView } from "@/components/map/map-view";
import { LocationMarkers } from "@/components/map/location-markers";
import { SelfMarker } from "@/components/map/self-marker";
import { MapControls } from "@/components/map/map-controls";
import { GpxOverlay } from "@/components/map/gpx-overlay";
import { HistoryTracks } from "@/components/map/history-tracks";
import { ReplayControls } from "@/components/map/replay-controls";
import {
  ReplayPanel,
  type ReplayScope,
  type ReplayStartParams,
} from "@/components/map/replay-panel";
import { MapToolbar } from "@/components/map/map-toolbar";
import { FilterPanel } from "@/components/map/filter-panel";
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
  Group,
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

  // Group membership cache — maps group_id → user_id[] for history filtering
  const [groupMemberCache, setGroupMemberCache] = useState<
    Map<string, string[]>
  >(new Map());

  // Live map filter state
  const [selectedLiveGroupIds, setSelectedLiveGroupIds] = useState<Set<string>>(
    new Set()
  );
  const [selectedLiveUserIds, setSelectedLiveUserIds] = useState<Set<string>>(
    new Set()
  );
  const [showSelf, setShowSelf] = useState(true);
  const [filterPanelOpen, setFilterPanelOpen] = useState(false);

  // Build a group config lookup map from the user's groups
  const groupConfigMap = useMemo(() => {
    const m = new Map<string, Group>();
    for (const g of myGroups) {
      m.set(g.id, g);
    }
    return m;
  }, [myGroups]);

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

  // Fetch group members when a group is toggled in the filter panel
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

  // Compute filtered history using the shared filter panel selections
  const filteredHistory = useMemo(() => {
    if (
      selectedLiveGroupIds.size === 0 &&
      selectedLiveUserIds.size === 0
    ) {
      return activeHistory;
    }

    // Build allowed user_ids from group selections
    let allowedByGroup: Set<string> | null = null;
    if (selectedLiveGroupIds.size > 0) {
      allowedByGroup = new Set<string>();
      for (const gid of selectedLiveGroupIds) {
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
      if (
        selectedLiveUserIds.size > 0 &&
        !selectedLiveUserIds.has(entry.user_id)
      ) {
        return false;
      }
      return true;
    });
  }, [
    activeHistory,
    selectedLiveGroupIds,
    selectedLiveUserIds,
    groupMemberCache,
  ]);

  // ---------------------------------------------------------------------------
  // Replay lifecycle
  // ---------------------------------------------------------------------------

  const handleReplayStart = useCallback(
    (params: ReplayStartParams) => {
      // Clear all hooks
      clearVisibleHistory();
      clearGroupHistory();
      clearUserHistory();
      setPlaybackTime(undefined);

      setReplayRange({ from: params.from, to: params.to });

      // Derive scope from the shared filter panel selections
      if (selectedLiveUserIds.size === 1) {
        const userId = Array.from(selectedLiveUserIds)[0];
        setReplayScope("user");
        fetchUserHistory(userId, params.from, params.to);
      } else if (selectedLiveGroupIds.size === 1) {
        const groupId = Array.from(selectedLiveGroupIds)[0];
        setReplayScope("group");
        fetchGroupHistory(groupId, params.from, params.to);
      } else {
        setReplayScope("all");
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
      selectedLiveGroupIds,
      selectedLiveUserIds,
    ]
  );

  const stopReplay = useCallback(() => {
    setReplayActive(false);
    setPlaybackTime(undefined);
    setReplayScope("all");
    setReplayRange(null);
    clearVisibleHistory();
    clearGroupHistory();
    clearUserHistory();
  }, [clearVisibleHistory, clearGroupHistory, clearUserHistory]);

  const handleExportGPX = useCallback(
    async (from: Date, to: Date) => {
      try {
        const params = new URLSearchParams({
          from: from.toISOString(),
          to: to.toISOString(),
        });
        // If exactly one user is selected in filters, export their data
        const userId =
          selectedLiveUserIds.size === 1
            ? Array.from(selectedLiveUserIds)[0]
            : undefined;
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
    [selectedLiveUserIds]
  );

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
  let displayLocations = new Map(locations);
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

  // Derive the full list of visible users (before filtering) for the filter panel
  const allVisibleUsers = Array.from(displayLocations.values()).map((loc) => ({
    user_id: loc.user_id,
    display_name: loc.display_name,
    username: loc.username,
  }));

  // Apply live group filter — when groups are selected, only show their users
  if (selectedLiveGroupIds.size > 0) {
    const filtered = new Map(displayLocations);
    for (const [userId, loc] of filtered) {
      if (!loc.group_id || !selectedLiveGroupIds.has(loc.group_id)) {
        filtered.delete(userId);
      }
    }
    displayLocations = filtered;
  }

  // Apply live user filter — when users are selected, only show those users
  if (selectedLiveUserIds.size > 0) {
    const filtered = new Map(displayLocations);
    for (const [userId] of filtered) {
      if (!selectedLiveUserIds.has(userId)) {
        filtered.delete(userId);
      }
    }
    displayLocations = filtered;
  }

  // Whether any live filter is actively applied
  const filterActive =
    selectedLiveGroupIds.size > 0 ||
    selectedLiveUserIds.size > 0 ||
    !showSelf;

  return (
    <div className="relative h-[calc(100vh-3.5rem)]">
      <MapView settings={settings} onMapReady={handleMapReady}>
        {mapRef.current && (
          <>
            <MapControls map={mapRef.current} terrainAvailable={!!settings.terrain_url} position={lastPosition} />
            <SelfMarker
              map={mapRef.current}
              position={showSelf ? lastPosition : null}
              autoCenter={!gpxMessageId && !replayActive}
              icon={user?.marker_icon}
              color={user?.marker_color}
            />
            <LocationMarkers
              map={mapRef.current}
              locations={displayLocations}
              currentUserId={user?.id}
              groups={groupConfigMap}
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

      {/* Top-left controls: Toolbar + panel area */}
      <div className="absolute top-3 left-3 z-10 flex flex-col gap-2">
        <MapToolbar
          onReplayClick={() => {
            setReplayPanelOpen((v) => !v);
            setFilterPanelOpen(false);
          }}
          replayActive={replayActive}
          filterActive={filterActive}
          onFilterClick={() => {
            setFilterPanelOpen((v) => !v);
            setReplayPanelOpen(false);
          }}
        />

        {/* Replay setup panel (below toolbar, mutually exclusive with filter) */}
        {replayPanelOpen && !replayActive && (
          <ReplayPanel
            isLoading={activeHistoryLoading}
            onStart={handleReplayStart}
            onExportGPX={handleExportGPX}
            onCancel={() => setReplayPanelOpen(false)}
          />
        )}

        {/* Filter panel (below toolbar, mutually exclusive with replay) */}
        {filterPanelOpen && !replayPanelOpen && (
          <FilterPanel
            showSelf={showSelf}
            onShowSelfChange={setShowSelf}
            groups={myGroups}
            selectedGroupIds={selectedLiveGroupIds}
            onGroupToggle={(groupId) => {
              setSelectedLiveGroupIds((prev) => {
                const next = new Set(prev);
                if (next.has(groupId)) {
                  next.delete(groupId);
                } else {
                  next.add(groupId);
                  fetchGroupMembers(groupId);
                }
                return next;
              });
            }}
            onGroupsClear={() => setSelectedLiveGroupIds(new Set())}
            users={allVisibleUsers}
            selectedUserIds={selectedLiveUserIds}
            onUserToggle={(userId) => {
              setSelectedLiveUserIds((prev) => {
                const next = new Set(prev);
                if (next.has(userId)) {
                  next.delete(userId);
                } else {
                  next.add(userId);
                }
                return next;
              });
            }}
            onUsersClear={() => setSelectedLiveUserIds(new Set())}
          />
        )}
      </div>

      {/* Replay controls bar (bottom) */}
      {replayActive && filteredHistory.length > 0 && replayRange != null && (
        <ReplayControls
          from={replayRange.from}
          to={replayRange.to}
          onTimeChange={setPlaybackTime}
          onReset={stopReplay}
        />
      )}

      {/* Replay no-data message */}
      {replayActive && activeHistory.length === 0 && !activeHistoryLoading && (
        <div className="absolute bottom-4 left-4 right-4 z-10 flex items-center justify-between bg-card/90 backdrop-blur-sm border rounded-lg px-4 py-3 shadow-lg">
          <span className="text-sm text-muted-foreground">
            No location data for the selected time range
          </span>
          <Button variant="ghost" size="sm" onClick={stopReplay}>
            Close
          </Button>
        </div>
      )}
    </div>
  );
}

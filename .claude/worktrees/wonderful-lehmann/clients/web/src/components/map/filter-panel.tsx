"use client";

import { useState } from "react";
import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { markerSVGString } from "@/components/map/marker-shapes";
import type { Group } from "@/types/api";

interface FilterPanelProps {
  /** Show the current user's marker on the map. */
  showSelf: boolean;
  onShowSelfChange: (show: boolean) => void;

  /** Master toggle for all drawing overlays. */
  showDrawings: boolean;
  onShowDrawingsChange: (show: boolean) => void;

  /** When true, only primary devices are shown on the map. */
  primaryOnly: boolean;
  onPrimaryOnlyChange: (primary: boolean) => void;

  /** Available groups the user belongs to. */
  groups: Group[];
  selectedGroupIds: Set<string>;
  onGroupToggle: (groupId: string) => void;
  onGroupsClear: () => void;

  /** Visible users derived from live/replay locations. */
  users: Array<{ user_id: string; display_name: string; username: string }>;
  selectedUserIds: Set<string>;
  onUserToggle: (userId: string) => void;
  onUsersClear: () => void;
}

/** Threshold above which a search input is shown for a list section. */
const SEARCH_THRESHOLD = 5;

/**
 * Filter panel displayed below the map toolbar.
 * Provides controls for filtering map markers by group, user, device status,
 * and whether the current user's own marker is visible.
 *
 * Designed to be reusable for both live map and replay modes via props.
 */
export function FilterPanel({
  showSelf,
  onShowSelfChange,
  showDrawings,
  onShowDrawingsChange,
  primaryOnly,
  onPrimaryOnlyChange,
  groups,
  selectedGroupIds,
  onGroupToggle,
  onGroupsClear,
  users,
  selectedUserIds,
  onUserToggle,
  onUsersClear,
}: FilterPanelProps) {
  const [groupSearch, setGroupSearch] = useState("");
  const [userSearch, setUserSearch] = useState("");

  const filteredGroups = groupSearch
    ? groups.filter((g) =>
        g.name.toLowerCase().includes(groupSearch.toLowerCase())
      )
    : groups;

  const filteredUsers = userSearch
    ? users.filter(
        (u) =>
          (u.display_name || u.username)
            .toLowerCase()
            .includes(userSearch.toLowerCase())
      )
    : users;

  return (
    <div className="bg-card/95 backdrop-blur-sm border rounded-lg p-3 shadow-lg sm:w-64 max-h-[calc(100vh-8rem)] overflow-y-auto space-y-3">
      {/* Show self */}
      <label className="flex items-center gap-2 text-sm cursor-pointer">
        <input
          type="checkbox"
          checked={showSelf}
          onChange={() => onShowSelfChange(!showSelf)}
          className="h-3.5 w-3.5"
        />
        <span>Show self</span>
      </label>

      {/* Show drawings */}
      <label className="flex items-center gap-2 text-sm cursor-pointer">
        <input
          type="checkbox"
          checked={showDrawings}
          onChange={() => onShowDrawingsChange(!showDrawings)}
          className="h-3.5 w-3.5"
        />
        <span>Show drawings</span>
      </label>

      {/* Groups */}
      {groups.length > 0 && (
        <>
          <Separator />

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Groups
              </h4>
              {selectedGroupIds.size > 0 && (
                <button
                  className="text-xs text-muted-foreground hover:text-foreground"
                  onClick={onGroupsClear}
                >
                  Clear
                </button>
              )}
            </div>

            {groups.length > SEARCH_THRESHOLD && (
              <div className="relative">
                <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground" />
                <Input
                  placeholder="Search groups..."
                  value={groupSearch}
                  onChange={(e) => setGroupSearch(e.target.value)}
                  className="h-7 pl-7 text-xs"
                />
              </div>
            )}

            <div className="space-y-1">
              {filteredGroups.map((g) => (
                <label
                  key={g.id}
                  className="flex items-center gap-2 text-sm cursor-pointer"
                >
                  <input
                    type="checkbox"
                    checked={selectedGroupIds.has(g.id)}
                    onChange={() => onGroupToggle(g.id)}
                    className="h-3.5 w-3.5"
                  />
                  <span
                    className="flex-shrink-0"
                    dangerouslySetInnerHTML={{
                      __html: markerSVGString(
                        g.marker_icon || "circle",
                        g.marker_color || "#3b82f6",
                        14
                      ),
                    }}
                  />
                  <span className="truncate">{g.name}</span>
                </label>
              ))}
              {groupSearch && filteredGroups.length === 0 && (
                <p className="text-xs text-muted-foreground">No matches</p>
              )}
            </div>
          </div>
        </>
      )}

      {/* Users */}
      {users.length > 0 && (
        <>
          <Separator />

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Users
              </h4>
              {selectedUserIds.size > 0 && (
                <button
                  className="text-xs text-muted-foreground hover:text-foreground"
                  onClick={onUsersClear}
                >
                  Clear
                </button>
              )}
            </div>

            {users.length > SEARCH_THRESHOLD && (
              <div className="relative">
                <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground" />
                <Input
                  placeholder="Search users..."
                  value={userSearch}
                  onChange={(e) => setUserSearch(e.target.value)}
                  className="h-7 pl-7 text-xs"
                />
              </div>
            )}

            <div className="space-y-1">
              {filteredUsers.map((u) => (
                <label
                  key={u.user_id}
                  className="flex items-center gap-2 text-sm cursor-pointer"
                >
                  <input
                    type="checkbox"
                    checked={selectedUserIds.has(u.user_id)}
                    onChange={() => onUserToggle(u.user_id)}
                    className="h-3.5 w-3.5"
                  />
                  <span className="truncate">
                    {u.display_name || u.username}
                  </span>
                </label>
              ))}
              {userSearch && filteredUsers.length === 0 && (
                <p className="text-xs text-muted-foreground">No matches</p>
              )}
            </div>
          </div>
        </>
      )}

      {/* Device filters */}
      <Separator />

      <div className="space-y-1.5">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          Devices
        </h4>

        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <input
            type="checkbox"
            checked={primaryOnly}
            onChange={() => onPrimaryOnlyChange(!primaryOnly)}
            className="h-3.5 w-3.5"
          />
          <span>Primary devices only</span>
        </label>
      </div>
    </div>
  );
}

"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Group, GroupMember, ListResponse, User } from "@/types/api";

export type ReplayScope = "all" | "group" | "user";

export interface ReplayStartParams {
  from: Date;
  to: Date;
  scope: ReplayScope;
  groupId?: string;
  userId?: string;
}

interface ReplayPanelProps {
  isAdmin: boolean;
  isLoading: boolean;
  onStart: (params: ReplayStartParams) => void;
  onExportGPX: (from: Date, to: Date, userId?: string) => void;
  onCancel: () => void;
}

export function ReplayPanel({
  isAdmin,
  isLoading,
  onStart,
  onExportGPX,
  onCancel,
}: ReplayPanelProps) {
  // Time range
  const [replayFrom, setReplayFrom] = useState(() => {
    const d = new Date();
    d.setHours(d.getHours() - 1);
    return d.toISOString().slice(0, 16);
  });
  const [replayTo, setReplayTo] = useState(() =>
    new Date().toISOString().slice(0, 16)
  );

  // Admin scope filters
  const [selectedGroupId, setSelectedGroupId] = useState("");
  const [selectedUserId, setSelectedUserId] = useState("");

  // Admin dropdown data
  const [allGroups, setAllGroups] = useState<Group[]>([]);
  const [userOptions, setUserOptions] = useState<
    { id: string; label: string }[]
  >([]);

  // Fetch all groups for admin on mount
  useEffect(() => {
    if (!isAdmin) return;
    api
      .get<ListResponse<Group>>("/api/v1/groups", {
        params: { page: "1", page_size: "100" },
      })
      .then((res) => setAllGroups(res.data ?? []))
      .catch(() => {});
  }, [isAdmin]);

  // Fetch users: all users when no group selected, group members when group selected
  useEffect(() => {
    if (!isAdmin) return;

    if (selectedGroupId) {
      api
        .get<GroupMember[]>(`/api/v1/groups/${selectedGroupId}/members`)
        .then((members) =>
          setUserOptions(
            members
              .map((m) => ({
                id: m.user_id,
                label: m.display_name || m.username,
              }))
              .sort((a, b) => a.label.localeCompare(b.label))
          )
        )
        .catch(() => setUserOptions([]));
    } else {
      api
        .get<ListResponse<User>>("/api/v1/users", {
          params: { page: "1", page_size: "100" },
        })
        .then((res) =>
          setUserOptions(
            (res.data ?? [])
              .map((u) => ({
                id: u.id,
                label: u.display_name || u.username,
              }))
              .sort((a, b) => a.label.localeCompare(b.label))
          )
        )
        .catch(() => setUserOptions([]));
    }
  }, [isAdmin, selectedGroupId]);

  const handleGroupChange = useCallback((groupId: string) => {
    setSelectedGroupId(groupId);
    setSelectedUserId(""); // reset user when group changes
  }, []);

  const handleStart = useCallback(() => {
    const from = new Date(replayFrom);
    const to = new Date(replayTo);
    if (to <= from) {
      toast.error("End time must be after start time");
      return;
    }
    if (to.getTime() - from.getTime() > 24 * 60 * 60 * 1000) {
      toast.error("Time range must not exceed 24 hours");
      return;
    }

    let scope: ReplayScope = "all";
    if (selectedUserId) scope = "user";
    else if (selectedGroupId) scope = "group";

    onStart({
      from,
      to,
      scope,
      groupId: selectedGroupId || undefined,
      userId: selectedUserId || undefined,
    });
  }, [replayFrom, replayTo, selectedGroupId, selectedUserId, onStart]);

  const handleExport = useCallback(() => {
    const from = new Date(replayFrom);
    const to = new Date(replayTo);
    onExportGPX(from, to, selectedUserId || undefined);
  }, [replayFrom, replayTo, selectedUserId, onExportGPX]);

  const selectClassName =
    "flex h-8 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring";

  return (
    <div className="absolute top-3 left-3 z-10 bg-card/95 backdrop-blur-sm border rounded-lg p-4 shadow-lg space-y-3 w-72">
      <h3 className="text-sm font-medium">Replay Tracks</h3>

      {isAdmin && (
        <>
          <div className="space-y-1">
            <Label className="text-xs">Group</Label>
            <select
              value={selectedGroupId}
              onChange={(e) => handleGroupChange(e.target.value)}
              className={selectClassName}
            >
              <option value="">All Groups</option>
              {allGroups.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1">
            <Label className="text-xs">User</Label>
            <select
              value={selectedUserId}
              onChange={(e) => setSelectedUserId(e.target.value)}
              className={selectClassName}
            >
              <option value="">All Users</option>
              {userOptions.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.label}
                </option>
              ))}
            </select>
          </div>
        </>
      )}

      <div className="space-y-1">
        <Label className="text-xs">From</Label>
        <Input
          type="datetime-local"
          value={replayFrom}
          onChange={(e) => setReplayFrom(e.target.value)}
          className="h-8 text-sm"
        />
      </div>
      <div className="space-y-1">
        <Label className="text-xs">To</Label>
        <Input
          type="datetime-local"
          value={replayTo}
          onChange={(e) => setReplayTo(e.target.value)}
          className="h-8 text-sm"
        />
      </div>

      <div className="flex gap-2">
        <Button
          size="sm"
          onClick={handleStart}
          disabled={isLoading}
          className="flex-1"
        >
          {isLoading ? "Loading..." : "Start"}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={handleExport}
          className="flex-1"
        >
          Export GPX
        </Button>
        <Button size="sm" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  );
}

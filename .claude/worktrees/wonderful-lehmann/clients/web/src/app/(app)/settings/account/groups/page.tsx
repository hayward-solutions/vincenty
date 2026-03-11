"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { useAuth } from "@/lib/auth-context";
import { api, ApiError } from "@/lib/api";
import { useUpdateGroupMarker } from "@/lib/hooks/use-groups";
import {
  AVAILABLE_SHAPES,
  MARKER_SHAPES,
  PRESET_COLORS,
  markerSVGString,
  type MarkerShape,
} from "@/components/map/marker-shapes";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import type { Group, GroupMember } from "@/types/api";

/** A group the current user is a group admin of. */
interface AdminGroup extends Group {
  is_group_admin: boolean;
}

export default function AccountGroupsPage() {
  const { user } = useAuth();
  const [groups, setGroups] = useState<AdminGroup[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [editGroup, setEditGroup] = useState<AdminGroup | null>(null);

  const fetchGroups = useCallback(async () => {
    if (!user) return;
    setIsLoading(true);
    try {
      // Fetch the user's groups
      const myGroups = await api.get<Group[]>("/api/v1/users/me/groups");

      // For each group, check if the user is a group admin
      const enriched: AdminGroup[] = [];
      for (const g of myGroups) {
        try {
          const members = await api.get<GroupMember[]>(
            `/api/v1/groups/${g.id}/members`
          );
          const self = members.find((m) => m.user_id === user.id);
          enriched.push({
            ...g,
            is_group_admin: self?.is_group_admin ?? false,
          });
        } catch {
          enriched.push({ ...g, is_group_admin: false });
        }
      }
      setGroups(enriched);
    } catch {
      setGroups([]);
    } finally {
      setIsLoading(false);
    }
  }, [user]);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  if (isLoading) {
    return (
      <div className="p-4 md:p-6 space-y-4">
        <h1 className="text-2xl font-semibold">My Groups</h1>
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    );
  }

  return (
    <div className="p-4 md:p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">My Groups</h1>
        <p className="text-muted-foreground text-sm mt-1">
          Configure how your group members appear on the map. Group admins can
          set a custom marker icon and color.
        </p>
      </div>

      {groups.length === 0 ? (
        <p className="text-muted-foreground">
          You are not a member of any groups.
        </p>
      ) : (
        <div className="space-y-3">
          {groups.map((g) => (
            <div
              key={g.id}
              className="flex items-center justify-between rounded-lg border p-4"
            >
              <div className="flex items-center gap-3">
                <span
                  dangerouslySetInnerHTML={{
                    __html: markerSVGString(
                      g.marker_icon || "circle",
                      g.marker_color || "#3b82f6",
                      24
                    ),
                  }}
                />
                <div>
                  <p className="font-medium">{g.name}</p>
                  {g.description && (
                    <p className="text-sm text-muted-foreground">
                      {g.description}
                    </p>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span
                  className="text-xs px-2 py-0.5 rounded-full"
                  style={{
                    backgroundColor: g.marker_color || "#3b82f6",
                    color: "white",
                  }}
                >
                  {MARKER_SHAPES[(g.marker_icon || "circle") as MarkerShape]
                    ?.label ?? "Circle"}
                </span>
                {g.is_group_admin ? (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setEditGroup(g)}
                  >
                    Edit Marker
                  </Button>
                ) : (
                  <span className="text-xs text-muted-foreground">
                    Admin only
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {editGroup && (
        <MarkerEditorDialog
          group={editGroup}
          open={!!editGroup}
          onOpenChange={(open) => !open && setEditGroup(null)}
          onSaved={() => {
            setEditGroup(null);
            fetchGroups();
          }}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Marker Editor Dialog
// ---------------------------------------------------------------------------

function MarkerEditorDialog({
  group,
  open,
  onOpenChange,
  onSaved,
}: {
  group: AdminGroup;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const { updateMarker, isLoading } = useUpdateGroupMarker();
  const [icon, setIcon] = useState<string>(group.marker_icon || "circle");
  const [color, setColor] = useState<string>(group.marker_color || "#3b82f6");
  const [customColor, setCustomColor] = useState<string>(
    PRESET_COLORS.includes(group.marker_color || "#3b82f6")
      ? ""
      : group.marker_color || ""
  );

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    try {
      await updateMarker(group.id, {
        marker_icon: icon,
        marker_color: color,
      });
      toast.success(`Marker updated for "${group.name}"`);
      onSaved();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update marker"
      );
    }
  }

  function handleCustomColorChange(value: string) {
    setCustomColor(value);
    // Auto-apply if it looks like a valid hex color
    if (/^#[0-9a-fA-F]{6}$/.test(value)) {
      setColor(value);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Marker - {group.name}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSave} className="space-y-5">
          {/* Live preview */}
          <div className="flex items-center justify-center p-4 bg-muted/50 rounded-lg">
            <div className="flex flex-col items-center gap-1">
              <span
                dangerouslySetInnerHTML={{
                  __html: markerSVGString(icon, color, 36),
                }}
              />
              <span className="text-xs text-muted-foreground mt-1">
                Preview
              </span>
            </div>
          </div>

          {/* Icon picker */}
          <div className="space-y-2">
            <Label>Shape</Label>
            <div className="grid grid-cols-5 gap-2">
              {AVAILABLE_SHAPES.map((shape) => (
                <button
                  key={shape}
                  type="button"
                  onClick={() => setIcon(shape)}
                  className={`flex flex-col items-center gap-1 p-2 rounded-md border-2 transition-colors ${
                    icon === shape
                      ? "border-primary bg-primary/10"
                      : "border-transparent hover:bg-muted"
                  }`}
                >
                  <span
                    dangerouslySetInnerHTML={{
                      __html: markerSVGString(shape, color, 20),
                    }}
                  />
                  <span className="text-[10px] text-muted-foreground">
                    {MARKER_SHAPES[shape].label}
                  </span>
                </button>
              ))}
            </div>
          </div>

          {/* Color picker */}
          <div className="space-y-2">
            <Label>Color</Label>
            <div className="flex flex-wrap gap-2">
              {PRESET_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => {
                    setColor(c);
                    setCustomColor("");
                  }}
                  className={`w-7 h-7 rounded-full border-2 transition-all ${
                    color === c && !customColor
                      ? "border-foreground scale-110"
                      : "border-transparent hover:scale-105"
                  }`}
                  style={{ backgroundColor: c }}
                  title={c}
                />
              ))}
            </div>
            <div className="flex items-center gap-2 mt-2">
              <Label htmlFor="custom-color" className="text-xs whitespace-nowrap">
                Custom hex:
              </Label>
              <Input
                id="custom-color"
                value={customColor}
                onChange={(e) => handleCustomColorChange(e.target.value)}
                placeholder="#ff0000"
                className="h-8 text-sm font-mono w-28"
                maxLength={7}
              />
              {customColor && /^#[0-9a-fA-F]{6}$/.test(customColor) && (
                <div
                  className="w-6 h-6 rounded-full border"
                  style={{ backgroundColor: customColor }}
                />
              )}
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

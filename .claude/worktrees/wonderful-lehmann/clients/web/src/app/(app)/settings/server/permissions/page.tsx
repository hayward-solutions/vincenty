"use client";

import { useState } from "react";
import { usePermissionPolicy } from "@/lib/hooks/use-permissions";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import type { PermissionPolicy } from "@/types/api";

// ---------------------------------------------------------------------------
// Constants — must match the Go model exactly
// ---------------------------------------------------------------------------

const COMMUNICATION_ROLES = [
  { key: "server_admin", label: "Server Admins" },
  { key: "group_admin", label: "Group Admins" },
  { key: "writer", label: "Writers" },
  { key: "reader", label: "Readers" },
] as const;

const COMMUNICATION_ACTIONS = [
  { key: "send_messages", label: "Send messages" },
  { key: "read_messages", label: "Read messages" },
  { key: "send_attachments", label: "Send attachments" },
  { key: "share_drawings", label: "Share drawings" },
  { key: "share_location", label: "Share location" },
  { key: "view_locations", label: "View locations" },
] as const;

const MANAGEMENT_ROLES = [
  { key: "group_admin", label: "Group Admins" },
  { key: "member", label: "Group Members" },
] as const;

const MANAGEMENT_ACTIONS = [
  { key: "add_members", label: "Add members" },
  { key: "remove_members", label: "Remove members" },
  { key: "update_members", label: "Update members" },
  { key: "update_marker", label: "Update marker" },
  { key: "view_audit_logs", label: "View audit logs" },
] as const;

// ---------------------------------------------------------------------------
// Matrix component
// ---------------------------------------------------------------------------

interface MatrixProps {
  title: string;
  description: string;
  actions: ReadonlyArray<{ key: string; label: string }>;
  roles: ReadonlyArray<{ key: string; label: string }>;
  data: Record<string, string[]>;
  onChange: (action: string, role: string, checked: boolean) => void;
  dirty: boolean;
}

function PermissionMatrix({
  title,
  description,
  actions,
  roles,
  data,
  onChange,
}: MatrixProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b">
                <th className="text-left py-2 pr-4 font-medium text-muted-foreground">
                  Action
                </th>
                {roles.map((role) => (
                  <th
                    key={role.key}
                    className="text-center py-2 px-3 font-medium text-muted-foreground"
                  >
                    {role.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {actions.map((action) => {
                const allowedRoles = data[action.key] ?? [];
                return (
                  <tr key={action.key} className="border-b last:border-b-0">
                    <td className="py-3 pr-4">{action.label}</td>
                    {roles.map((role) => {
                      const checked = allowedRoles.includes(role.key);
                      return (
                        <td key={role.key} className="text-center py-3 px-3">
                          <Checkbox
                            checked={checked}
                            onCheckedChange={(v) =>
                              onChange(action.key, role.key, v === true)
                            }
                          />
                        </td>
                      );
                    })}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function ServerPermissionsPage() {
  const { policy, isLoading, update } = usePermissionPolicy();
  const [draft, setDraft] = useState<PermissionPolicy | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  // Use draft if user has started editing, otherwise use loaded policy
  const effective = draft ?? policy;

  function handleChange(
    category: "group_communication" | "group_management",
    action: string,
    role: string,
    checked: boolean
  ) {
    setDraft((prev) => {
      const base = prev ?? policy;
      if (!base) return null;

      const existing = [...(base[category][action] ?? [])];
      const next = checked
        ? [...existing, role]
        : existing.filter((r) => r !== role);

      return {
        ...base,
        [category]: {
          ...base[category],
          [action]: next,
        },
      };
    });
  }

  async function handleSave() {
    if (!draft) return;
    setIsSaving(true);
    try {
      await update(draft);
      setDraft(null);
      toast.success("Permissions updated");
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update permissions"
      );
    } finally {
      setIsSaving(false);
    }
  }

  function handleReset() {
    setDraft(null);
  }

  const isDirty = draft !== null;

  return (
    <div className="p-4 md:p-6 space-y-6 max-w-4xl">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Permissions</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Configure which roles can perform each action. All users must be
            group members. Server admins manage groups via the admin panel.
          </p>
        </div>
        {isDirty && (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleReset}>
              Discard
            </Button>
            <Button size="sm" onClick={handleSave} disabled={isSaving}>
              {isSaving ? "Saving..." : "Save Changes"}
            </Button>
          </div>
        )}
      </div>

      {isLoading || !effective ? (
        <div className="space-y-6">
          <Skeleton className="h-72 w-full" />
          <Skeleton className="h-56 w-full" />
        </div>
      ) : (
        <>
          <PermissionMatrix
            title="Group Communication"
            description="Controls who can send/read messages, share attachments, drawings, and locations within groups. All callers must be group members."
            actions={COMMUNICATION_ACTIONS}
            roles={COMMUNICATION_ROLES}
            data={effective.group_communication}
            onChange={(action, role, checked) =>
              handleChange("group_communication", action, role, checked)
            }
            dirty={isDirty}
          />

          <PermissionMatrix
            title="Group Management"
            description="Controls who can manage group members and settings. Server admins bypass this via the admin panel."
            actions={MANAGEMENT_ACTIONS}
            roles={MANAGEMENT_ROLES}
            data={effective.group_management}
            onChange={(action, role, checked) =>
              handleChange("group_management", action, role, checked)
            }
            dirty={isDirty}
          />

          {isDirty && (
            <div className="rounded-md bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-400">
              You have unsaved changes. Click &quot;Save Changes&quot; to apply the
              new permission policy.
            </div>
          )}
        </>
      )}
    </div>
  );
}

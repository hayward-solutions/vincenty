"use client";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import type { AuditLogResponse } from "@/types/api";

const ACTION_LABELS: Record<string, string> = {
  "auth.login": "Login",
  "auth.logout": "Logout",
  "user.create": "Create User",
  "user.update": "Update User",
  "user.delete": "Delete User",
  "user.update_self": "Update Profile",
  "device.create": "Create Device",
  "device.update": "Update Device",
  "device.delete": "Delete Device",
  "group.create": "Create Group",
  "group.update": "Update Group",
  "group.delete": "Delete Group",
  "group.member_add": "Add Member",
  "group.member_update": "Update Member",
  "group.member_remove": "Remove Member",
  "message.send": "Send Message",
  "message.delete": "Delete Message",
  "map_config.create": "Create Map Config",
  "map_config.update": "Update Map Config",
  "map_config.delete": "Delete Map Config",
};

function actionLabel(action: string): string {
  return ACTION_LABELS[action] ?? action;
}

function actionVariant(
  action: string
): "default" | "secondary" | "destructive" | "outline" {
  if (action.endsWith(".delete") || action.endsWith(".remove")) return "destructive";
  if (action.endsWith(".create") || action.endsWith(".add")) return "default";
  if (action.startsWith("auth.")) return "outline";
  return "secondary";
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

interface AuditLogTableProps {
  logs: AuditLogResponse[];
  showUser?: boolean;
}

export function AuditLogTable({ logs, showUser = false }: AuditLogTableProps) {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            {showUser && <TableHead>User</TableHead>}
            <TableHead>Action</TableHead>
            <TableHead>Resource</TableHead>
            <TableHead>IP Address</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {logs.map((log) => (
            <TableRow key={log.id}>
              <TableCell className="whitespace-nowrap text-sm">
                {formatTime(log.created_at)}
              </TableCell>
              {showUser && (
                <TableCell className="text-sm">
                  {log.display_name || log.username}
                </TableCell>
              )}
              <TableCell>
                <Badge variant={actionVariant(log.action)}>
                  {actionLabel(log.action)}
                </Badge>
              </TableCell>
              <TableCell className="text-sm text-muted-foreground">
                {log.resource_type}
                {log.resource_id ? ` (${log.resource_id.slice(0, 8)}...)` : ""}
              </TableCell>
              <TableCell className="text-sm text-muted-foreground">
                {log.ip_address}
              </TableCell>
            </TableRow>
          ))}
          {logs.length === 0 && (
            <TableRow>
              <TableCell
                colSpan={showUser ? 5 : 4}
                className="text-center text-muted-foreground py-8"
              >
                No audit logs found
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}

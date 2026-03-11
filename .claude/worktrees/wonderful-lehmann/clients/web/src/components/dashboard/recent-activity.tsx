"use client";

import { useEffect } from "react";
import {
  LogIn,
  LogOut,
  MessageSquare,
  MapPin,
  Users,
  Shield,
  Key,
  Settings,
  Activity,
} from "lucide-react";
import { useMyAuditLogs } from "@/lib/hooks/use-audit-logs";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

/** Format a timestamp as a short relative string without external deps. */
function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60) return "just now";
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

interface ActionMeta {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  variant: "default" | "secondary" | "outline" | "destructive";
}

function getActionMeta(action: string, resourceType: string): ActionMeta {
  const key = `${action}:${resourceType}`;

  const map: Record<string, ActionMeta> = {
    "login:session": { icon: LogIn, label: "Login", variant: "secondary" },
    "logout:session": { icon: LogOut, label: "Logout", variant: "outline" },
    "create:message": { icon: MessageSquare, label: "Message sent", variant: "secondary" },
    "delete:message": { icon: MessageSquare, label: "Message deleted", variant: "destructive" },
    "create:location": { icon: MapPin, label: "Location update", variant: "outline" },
    "create:group": { icon: Users, label: "Group created", variant: "secondary" },
    "update:group": { icon: Users, label: "Group updated", variant: "outline" },
    "delete:group": { icon: Users, label: "Group deleted", variant: "destructive" },
    "create:user": { icon: Shield, label: "User created", variant: "secondary" },
    "update:user": { icon: Shield, label: "Profile updated", variant: "outline" },
    "create:api_token": { icon: Key, label: "Token created", variant: "secondary" },
    "delete:api_token": { icon: Key, label: "Token deleted", variant: "destructive" },
    "update:server_settings": { icon: Settings, label: "Settings changed", variant: "outline" },
  };

  return (
    map[key] ??
    map[`${action}:${resourceType}`] ?? {
      icon: Activity,
      label: `${action} ${resourceType}`.replace(/_/g, " "),
      variant: "outline" as const,
    }
  );
}

export function RecentActivity() {
  const { data: logs, isLoading, fetch } = useMyAuditLogs();

  useEffect(() => {
    fetch({ page_size: 10, page: 1 });
  }, [fetch]);

  return (
    <Card className="flex flex-col">
      <CardHeader className="pb-3">
        <CardTitle className="text-base font-semibold">
          Recent Activity
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1 p-0">
        {isLoading ? (
          <ul className="divide-y">
            {Array.from({ length: 5 }).map((_, i) => (
              <li key={i} className="flex items-center gap-3 px-6 py-3">
                <Skeleton className="h-6 w-6 rounded shrink-0" />
                <div className="flex-1 min-w-0 space-y-1">
                  <Skeleton className="h-3.5 w-28" />
                </div>
                <Skeleton className="h-3 w-14 shrink-0" />
              </li>
            ))}
          </ul>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-10 text-center px-6">
            <Activity className="h-8 w-8 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">No recent activity.</p>
          </div>
        ) : (
          <ul className="divide-y">
            {logs.map((log) => {
              const meta = getActionMeta(log.action, log.resource_type);
              const Icon = meta.icon;

              return (
                <li
                  key={log.id}
                  className="flex items-center gap-3 px-6 py-2.5"
                >
                  <Icon className="h-4 w-4 text-muted-foreground shrink-0" />
                  <div className="flex-1 min-w-0">
                    <Badge
                      variant={meta.variant}
                      className="text-[11px] px-1.5 py-0 font-normal"
                    >
                      {meta.label}
                    </Badge>
                  </div>
                  <span className="text-[11px] text-muted-foreground whitespace-nowrap shrink-0">
                    {relativeTime(log.created_at)}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

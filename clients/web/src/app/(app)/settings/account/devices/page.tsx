"use client";

import { useEffect } from "react";
import { toast } from "sonner";
import { useMyDevices, useDeleteDevice } from "@/lib/hooks/use-devices";
import { useWebSocket } from "@/lib/websocket-context";
import { ApiError } from "@/lib/api";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

// ---------------------------------------------------------------------------
// User-agent parsing helpers
// ---------------------------------------------------------------------------

/** Extract a human-readable browser + OS label from a raw UA string. */
function parseBrowserName(ua?: string): string {
  if (!ua) return "Unknown";

  let browser = "Unknown browser";
  let os = "";

  // Browser detection (order matters — check specific engines first)
  if (/Edg(?:e|A|iOS)?\/(\d+)/i.test(ua)) {
    browser = `Edge ${RegExp.$1}`;
  } else if (/OPR\/(\d+)/i.test(ua) || /Opera\/(\d+)/i.test(ua)) {
    browser = `Opera ${RegExp.$1}`;
  } else if (/Firefox\/(\d+)/i.test(ua)) {
    browser = `Firefox ${RegExp.$1}`;
  } else if (/(?:CriOS|Chrome)\/(\d+)/i.test(ua)) {
    browser = `Chrome ${RegExp.$1}`;
  } else if (/Version\/(\d+).*Safari/i.test(ua)) {
    browser = `Safari ${RegExp.$1}`;
  } else if (/Safari\/(\d+)/i.test(ua)) {
    browser = "Safari";
  }

  // OS detection
  if (/Windows NT 10/i.test(ua)) {
    os = "Windows";
  } else if (/Windows/i.test(ua)) {
    os = "Windows";
  } else if (/Mac OS X/i.test(ua)) {
    os = "macOS";
  } else if (/Android/i.test(ua)) {
    os = "Android";
  } else if (/iPhone|iPad|iPod/i.test(ua)) {
    os = "iOS";
  } else if (/Linux/i.test(ua)) {
    os = "Linux";
  } else if (/CrOS/i.test(ua)) {
    os = "ChromeOS";
  }

  return os ? `${browser} on ${os}` : browser;
}

/** Format an ISO timestamp as a relative time string (e.g. "2 hours ago"). */
function relativeTime(iso?: string): string {
  if (!iso) return "Never";
  const now = Date.now();
  const then = new Date(iso).getTime();
  const seconds = Math.floor((now - then) / 1000);

  if (seconds < 60) return "Just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}

/** Format an ISO timestamp as a short date (e.g. "Jan 15, 2026"). */
function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

/** Map device_type to a badge variant. */
function typeVariant(
  type: string
): "default" | "secondary" | "outline" {
  switch (type) {
    case "web":
      return "secondary";
    case "ios":
    case "android":
      return "outline";
    default:
      return "default";
  }
}

// ---------------------------------------------------------------------------
// Page component
// ---------------------------------------------------------------------------

export default function DevicesSettingsPage() {
  const { devices, isLoading, error, fetch } = useMyDevices();
  const { deleteDevice } = useDeleteDevice();
  const { deviceId } = useWebSocket();

  useEffect(() => {
    fetch();
  }, [fetch]);

  async function handleRemove(id: string, name: string) {
    if (!confirm(`Remove device "${name}"? It will need to re-register on next login.`))
      return;
    try {
      await deleteDevice(id);
      toast.success(`Device "${name}" removed`);
      fetch();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to remove device"
      );
    }
  }

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-semibold">Devices</h1>

      <Card>
        <CardHeader>
          <CardTitle>Your Devices</CardTitle>
          <CardDescription>
            Devices that are currently registered to your account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error && (
            <p className="text-sm text-destructive mb-4">{error}</p>
          )}

          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => {
                const key = `skeleton-${i}`;
                return <Skeleton key={key} className="h-12 w-full" />;
              })}
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Device</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Last Active</TableHead>
                    <TableHead>Registered</TableHead>
                    <TableHead className="w-[1%]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {devices.map((device) => {
                    const isCurrent = device.id === deviceId;
                    return (
                      <TableRow key={device.id}>
                        <TableCell>
                          <div className="flex flex-col gap-0.5">
                            <span className="text-sm font-medium flex items-center gap-2">
                              {device.name}
                              {isCurrent && (
                                <Badge variant="default" className="text-xs">
                                  This device
                                </Badge>
                              )}
                            </span>
                            {device.user_agent && (
                              <span className="text-xs text-muted-foreground">
                                {parseBrowserName(device.user_agent)}
                              </span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={typeVariant(device.device_type)}>
                            {device.device_type}
                          </Badge>
                        </TableCell>
                        <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                          {relativeTime(device.last_seen_at)}
                        </TableCell>
                        <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                          {formatDate(device.created_at)}
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive hover:text-destructive"
                            disabled={isCurrent}
                            onClick={() => handleRemove(device.id, device.name)}
                          >
                            Remove
                          </Button>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                  {devices.length === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={5}
                        className="text-center text-muted-foreground py-8"
                      >
                        No devices registered
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

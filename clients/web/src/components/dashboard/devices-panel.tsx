"use client";

import { useEffect } from "react";
import { Smartphone, Monitor, Globe, Star, Cpu } from "lucide-react";
import { useMyDevices } from "@/lib/hooks/use-devices";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

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
  if (d < 30) return `${d}d ago`;
  const mo = Math.floor(d / 30);
  return `${mo}mo ago`;
}

function DeviceIcon({
  type,
  className,
}: {
  type: string;
  className?: string;
}) {
  switch (type.toLowerCase()) {
    case "ios":
    case "android":
    case "mobile":
      return <Smartphone className={cn("h-4 w-4", className)} />;
    case "web":
      return <Globe className={cn("h-4 w-4", className)} />;
    case "desktop":
      return <Monitor className={cn("h-4 w-4", className)} />;
    case "cot":
    case "atak":
      return <Cpu className={cn("h-4 w-4", className)} />;
    default:
      return <Globe className={cn("h-4 w-4", className)} />;
  }
}

export function DevicesPanel() {
  const { devices, isLoading, fetch } = useMyDevices();

  useEffect(() => {
    fetch();
  }, [fetch]);

  return (
    <Card className="flex flex-col">
      <CardHeader className="pb-3">
        <CardTitle className="text-base font-semibold">My Devices</CardTitle>
      </CardHeader>
      <CardContent className="flex-1 p-0">
        {isLoading ? (
          <ul className="divide-y">
            {Array.from({ length: 3 }).map((_, i) => (
              <li key={i} className="flex items-center gap-3 px-6 py-3">
                <Skeleton className="h-6 w-6 rounded shrink-0" />
                <div className="flex-1 min-w-0 space-y-1">
                  <Skeleton className="h-3.5 w-32" />
                  <Skeleton className="h-3 w-20" />
                </div>
                <Skeleton className="h-5 w-14 shrink-0" />
              </li>
            ))}
          </ul>
        ) : devices.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-10 text-center px-6">
            <Smartphone className="h-8 w-8 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">
              No devices registered.
            </p>
          </div>
        ) : (
          <ul className="divide-y">
            {devices.map((device) => (
              <li
                key={device.id}
                className="flex items-center gap-3 px-6 py-3"
              >
                <DeviceIcon
                  type={device.device_type}
                  className="text-muted-foreground shrink-0"
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-1.5">
                    <span className="text-sm font-medium truncate">
                      {device.name}
                    </span>
                    {device.is_primary && (
                      <Star className="h-3 w-3 text-amber-500 fill-amber-500 shrink-0" />
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground capitalize">
                    {device.device_type}
                    {device.app_version ? ` · v${device.app_version}` : ""}
                  </p>
                </div>
                <div className="shrink-0 text-right">
                  {device.last_seen_at ? (
                    <span className="text-[11px] text-muted-foreground whitespace-nowrap">
                      {relativeTime(device.last_seen_at)}
                    </span>
                  ) : (
                    <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                      Never seen
                    </Badge>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

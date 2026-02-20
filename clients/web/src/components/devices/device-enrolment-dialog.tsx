"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useCreateDevice, useClaimDevice } from "@/lib/hooks/use-devices";
import { ApiError } from "@/lib/api";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { Device } from "@/types/api";

// ---------------------------------------------------------------------------
// Helpers (duplicated from devices settings page — small enough to inline)
// ---------------------------------------------------------------------------

function parseBrowserName(ua?: string): string {
  if (!ua) return "Unknown browser";

  let browser = "Unknown browser";
  let os = "";

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

  if (/Mac OS X/i.test(ua)) os = "macOS";
  else if (/Windows/i.test(ua)) os = "Windows";
  else if (/Android/i.test(ua)) os = "Android";
  else if (/iPhone|iPad|iPod/i.test(ua)) os = "iOS";
  else if (/CrOS/i.test(ua)) os = "ChromeOS";
  else if (/Linux/i.test(ua)) os = "Linux";

  return os ? `${browser} on ${os}` : browser;
}

function relativeTime(iso?: string): string {
  if (!iso) return "Never";
  const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
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

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface DeviceEnrolmentDialogProps {
  existingDevices: Device[];
  onResolved: (deviceId: string) => void;
}

export function DeviceEnrolmentDialog({
  existingDevices,
  onResolved,
}: DeviceEnrolmentDialogProps) {
  const { createDevice, isLoading: isCreating } = useCreateDevice();
  const { claimDevice, isLoading: isClaiming } = useClaimDevice();
  const [claimingId, setClaimingId] = useState<string | null>(null);

  const busy = isCreating || isClaiming;

  async function handleRegisterNew() {
    try {
      const device = await createDevice();
      onResolved(device.id);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to register device"
      );
    }
  }

  async function handleClaim(id: string) {
    setClaimingId(id);
    try {
      const device = await claimDevice(id);
      onResolved(device.id);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to claim device"
      );
    } finally {
      setClaimingId(null);
    }
  }

  return (
    <Dialog open onOpenChange={() => {}}>
      <DialogContent showCloseButton={false} onPointerDownOutside={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>Device Not Recognised</DialogTitle>
          <DialogDescription>
            We don&apos;t recognise this browser. Would you like to register it
            as a new device, or re-use an existing one?
          </DialogDescription>
        </DialogHeader>

        {existingDevices.length > 0 && (
          <div className="space-y-2">
            <p className="text-sm font-medium">Your existing devices</p>
            <div className="rounded-md border divide-y max-h-60 overflow-y-auto">
              {existingDevices.map((device) => (
                <div
                  key={device.id}
                  className="flex items-center justify-between gap-4 px-3 py-2.5"
                >
                  <div className="flex flex-col gap-0.5 min-w-0">
                    <span className="text-sm font-medium flex items-center gap-2">
                      {device.name}
                      <Badge variant="secondary" className="text-xs">
                        {device.device_type}
                      </Badge>
                    </span>
                    <span className="text-xs text-muted-foreground truncate">
                      {parseBrowserName(device.user_agent)}
                      {" \u00b7 "}
                      Last seen: {relativeTime(device.last_seen_at)}
                    </span>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={busy}
                    onClick={() => handleClaim(device.id)}
                  >
                    {claimingId === device.id ? "Claiming..." : "Use this"}
                  </Button>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="pt-2">
          <Button
            className="w-full"
            disabled={busy}
            onClick={handleRegisterNew}
          >
            {isCreating ? "Registering..." : "Register as new device"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

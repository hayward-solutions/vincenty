"use client";

import { cn } from "@/lib/utils";
import type { MessageResponse } from "@/types/api";
import { Download, MapPin, FileText, Info } from "lucide-react";
import Link from "next/link";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

function attachmentUrl(attachmentId: string): string {
  const base = `${API_BASE}/api/v1/attachments/${attachmentId}/download`;
  const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
  return token ? `${base}?token=${encodeURIComponent(token)}` : base;
}

interface MessageBubbleProps {
  message: MessageResponse;
  isOwn: boolean;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatFullDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "medium",
  });
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatCoord(lat: number, lng: number): string {
  return `${lat.toFixed(5)}, ${lng.toFixed(5)}`;
}

function MessageTypeLabel(type: string): string {
  switch (type) {
    case "text":
      return "Text";
    case "file":
      return "File";
    case "gpx":
      return "GPX Track";
    default:
      return type;
  }
}

export function MessageBubble({ message, isOwn }: MessageBubbleProps) {
  const hasAttachments =
    message.attachments != null && message.attachments.length > 0;
  const hasLocation = message.lat != null && message.lng != null;
  const isGpx = message.message_type === "gpx" && message.metadata != null;
  const hasText = !!message.content;
  const hasTrailingContent = isGpx;

  // EXIF location from metadata (primary display source)
  const exifLocations = message.metadata?.exif_locations;
  const firstExif =
    exifLocations && exifLocations.length > 0 ? exifLocations[0] : null;

  return (
    <div
      className={cn(
        "flex flex-col max-w-[75%] gap-1",
        isOwn ? "ml-auto items-end" : "mr-auto items-start"
      )}
    >
      {/* Sender name (not for own messages) */}
      {!isOwn && (
        <span className="text-xs text-muted-foreground px-1">
          {message.display_name || message.username}
        </span>
      )}

      {/* Bubble */}
      <div
        className={cn(
          "rounded-lg overflow-hidden text-sm break-words",
          isOwn
            ? "bg-primary text-primary-foreground"
            : "bg-muted text-foreground"
        )}
      >
        {/* Text content */}
        {hasText && (
          <p
            className={cn(
              "whitespace-pre-wrap px-3 pt-2",
              hasAttachments || hasTrailingContent ? "pb-1" : "pb-2"
            )}
          >
            {message.content}
          </p>
        )}

        {/* Attachments */}
        {hasAttachments && (
          <div
            className={cn(
              "flex flex-col",
              hasTrailingContent ? "" : "last:*:pb-0"
            )}
          >
            {message.attachments.map((att) => {
              const isImage = att.content_type.startsWith("image/");
              const downloadUrl = attachmentUrl(att.id);

              return (
                <div key={att.id}>
                  {isImage ? (
                    <a
                      href={downloadUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="block"
                    >
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img
                        src={downloadUrl}
                        alt={att.filename}
                        className="max-w-full max-h-48 w-full object-cover"
                        loading="lazy"
                      />
                    </a>
                  ) : (
                    <a
                      href={downloadUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={cn(
                        "flex items-center gap-2 px-3 py-1.5 text-xs",
                        isOwn
                          ? "bg-primary-foreground/10 hover:bg-primary-foreground/20"
                          : "bg-background/50 hover:bg-background/80"
                      )}
                    >
                      <Download className="h-3.5 w-3.5 shrink-0" />
                      <span className="truncate">{att.filename}</span>
                      <span className="shrink-0 text-muted-foreground">
                        {formatFileSize(att.size_bytes)}
                      </span>
                    </a>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {/* GPX "View on Map" link */}
        {isGpx && (
          <Link
            href={`/map?gpx=${message.id}`}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium",
              isOwn
                ? "text-primary-foreground/80 hover:text-primary-foreground"
                : "text-primary hover:underline"
            )}
          >
            <FileText className="h-3.5 w-3.5" />
            View GPX on Map
          </Link>
        )}
      </div>

      {/* Footer: time + location + info */}
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground px-1">
        <span>{formatTime(message.created_at)}</span>
        {hasLocation && (
          <span className="flex items-center gap-0.5">
            <MapPin className="h-3 w-3" />
            {message.lat!.toFixed(4)}, {message.lng!.toFixed(4)}
          </span>
        )}

        {/* Info popover */}
        <Popover>
          <PopoverTrigger asChild>
            <button
              type="button"
              className="inline-flex items-center justify-center h-4 w-4 rounded-full hover:bg-muted-foreground/20 transition-colors"
              aria-label="Message info"
            >
              <Info className="h-3 w-3" />
            </button>
          </PopoverTrigger>
          <PopoverContent
            side={isOwn ? "left" : "right"}
            align="start"
            className="w-64 p-3 text-xs space-y-2"
          >
            <p className="font-semibold text-sm">Message Info</p>

            <div className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
              {/* From */}
              <span className="text-muted-foreground">From</span>
              <span className="truncate">
                {message.display_name || message.username}
                {message.display_name && (
                  <span className="text-muted-foreground">
                    {" "}
                    ({message.username})
                  </span>
                )}
              </span>

              {/* Sent at */}
              <span className="text-muted-foreground">Sent</span>
              <span>{formatFullDate(message.created_at)}</span>

              {/* Type */}
              <span className="text-muted-foreground">Type</span>
              <span>{MessageTypeLabel(message.message_type)}</span>

              {/* Sent-from location (browser geolocation) */}
              {hasLocation && (
                <>
                  <span className="text-muted-foreground">Sent from</span>
                  <span className="flex items-center gap-1">
                    <MapPin className="h-3 w-3 shrink-0" />
                    {formatCoord(message.lat!, message.lng!)}
                  </span>
                </>
              )}

              {/* Photo location (EXIF) */}
              {firstExif && (
                <>
                  <span className="text-muted-foreground">Photo location</span>
                  <span className="flex items-center gap-1">
                    <MapPin className="h-3 w-3 shrink-0" />
                    {formatCoord(firstExif.lat, firstExif.lng)}
                    {firstExif.altitude != null && (
                      <span className="text-muted-foreground">
                        {firstExif.altitude.toFixed(0)}m
                      </span>
                    )}
                  </span>
                </>
              )}
              {firstExif?.taken_at && (
                <>
                  <span className="text-muted-foreground">Photo taken</span>
                  <span>{formatFullDate(firstExif.taken_at)}</span>
                </>
              )}
            </div>

            {/* Attachments */}
            {hasAttachments && (
              <div className="border-t pt-2 space-y-1">
                <p className="text-muted-foreground font-medium">
                  Attachments ({message.attachments.length})
                </p>
                {message.attachments.map((att) => (
                  <div
                    key={att.id}
                    className="flex items-center justify-between gap-2"
                  >
                    <span className="truncate">{att.filename}</span>
                    <span className="shrink-0 text-muted-foreground">
                      {formatFileSize(att.size_bytes)}
                    </span>
                  </div>
                ))}
              </div>
            )}

            {/* Message ID */}
            <div className="border-t pt-2">
              <span className="text-muted-foreground">ID: </span>
              <span className="font-mono">
                {message.id.substring(0, 8)}
              </span>
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </div>
  );
}

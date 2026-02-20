"use client";

import { cn } from "@/lib/utils";
import type { MessageResponse } from "@/types/api";
import { Download, MapPin, FileText } from "lucide-react";
import Link from "next/link";

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

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function MessageBubble({ message, isOwn }: MessageBubbleProps) {
  const hasAttachments =
    message.attachments != null && message.attachments.length > 0;
  const hasLocation = message.lat != null && message.lng != null;
  const isGpx = message.message_type === "gpx" && message.metadata != null;

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
          "rounded-lg px-3 py-2 text-sm break-words",
          isOwn
            ? "bg-primary text-primary-foreground"
            : "bg-muted text-foreground"
        )}
      >
        {/* Text content */}
        {message.content && (
          <p className="whitespace-pre-wrap">{message.content}</p>
        )}

        {/* Attachments */}
        {hasAttachments ? (
          <div className="mt-1 flex flex-col gap-1">
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
                        className="max-w-full max-h-48 rounded object-cover"
                        loading="lazy"
                      />
                    </a>
                  ) : (
                    <a
                      href={downloadUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={cn(
                        "flex items-center gap-2 rounded px-2 py-1 text-xs",
                        isOwn
                          ? "bg-primary-foreground/20 hover:bg-primary-foreground/30"
                          : "bg-background hover:bg-accent"
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
        ) : null}

        {/* GPX "View on Map" link */}
        {isGpx && (
          <Link
            href={`/map?gpx=${message.id}`}
            className={cn(
              "flex items-center gap-1.5 mt-1 text-xs font-medium",
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

      {/* Footer: time + location */}
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground px-1">
        <span>{formatTime(message.created_at)}</span>
        {hasLocation && (
          <span className="flex items-center gap-0.5">
            <MapPin className="h-3 w-3" />
            {message.lat!.toFixed(4)}, {message.lng!.toFixed(4)}
          </span>
        )}
      </div>
    </div>
  );
}

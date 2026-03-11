"use client";

import { useEffect, useState } from "react";
import { FeedViewer } from "./feed-viewer";
import { useViewFeed } from "@/lib/hooks/use-feeds";
import { Maximize2 } from "lucide-react";
import type { VideoFeed } from "@/types/api";

interface FeedGridProps {
  feeds: VideoFeed[];
  serverUrl: string;
}

interface FeedConnection {
  feed: VideoFeed;
  token: string;
}

export function FeedGrid({ feeds, serverUrl }: FeedGridProps) {
  const [connections, setConnections] = useState<FeedConnection[]>([]);
  const [expandedFeed, setExpandedFeed] = useState<string | null>(null);
  const { viewFeed } = useViewFeed();

  useEffect(() => {
    const activeFeeds = feeds.filter((f) => f.is_active);

    // Connect to each active feed
    const connect = async () => {
      const newConnections: FeedConnection[] = [];
      for (const feed of activeFeeds) {
        try {
          const resp = await viewFeed(feed.id);
          newConnections.push({ feed, token: resp.token });
        } catch (err) {
          console.error(`Failed to connect to feed ${feed.name}:`, err);
        }
      }
      setConnections(newConnections);
    };

    connect();
  }, [feeds, viewFeed]);

  if (connections.length === 0) {
    return (
      <div className="flex items-center justify-center h-48 text-muted-foreground">
        No active feeds
      </div>
    );
  }

  const gridCols =
    connections.length <= 1
      ? "grid-cols-1"
      : connections.length <= 4
        ? "grid-cols-2"
        : "grid-cols-3";

  return (
    <div className={`grid ${gridCols} gap-2`}>
      {connections.map(({ feed, token }) => (
        <div
          key={feed.id}
          className={
            expandedFeed === feed.id ? "col-span-full row-span-2" : ""
          }
        >
          <div className="relative group">
            <FeedViewer
              token={token}
              serverUrl={serverUrl}
              feedName={feed.name}
            />
            <button
              type="button"
              onClick={() =>
                setExpandedFeed(expandedFeed === feed.id ? null : feed.id)
              }
              className="absolute top-2 right-2 bg-black/60 rounded p-1 opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <Maximize2 className="h-4 w-4 text-white" />
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

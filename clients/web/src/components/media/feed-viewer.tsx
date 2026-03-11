"use client";

import {
  LiveKitRoom,
  RoomAudioRenderer,
  VideoTrack,
  useTracks,
} from "@livekit/components-react";
import "@livekit/components-styles";
import { Track } from "livekit-client";
import { Video, VideoOff } from "lucide-react";

interface FeedViewerProps {
  token: string;
  serverUrl: string;
  feedName: string;
}

export function FeedViewer({ token, serverUrl, feedName }: FeedViewerProps) {
  return (
    <div className="bg-gray-900 rounded-lg overflow-hidden">
      <LiveKitRoom
        token={token}
        serverUrl={serverUrl}
        connect={true}
        audio={true}
        video={false}
      >
        <FeedContent feedName={feedName} />
        <RoomAudioRenderer />
      </LiveKitRoom>
    </div>
  );
}

function FeedContent({ feedName }: { feedName: string }) {
  const tracks = useTracks([Track.Source.Camera, Track.Source.ScreenShare]);
  const videoTrack = tracks.find(
    (t) => t.source === Track.Source.Camera || t.source === Track.Source.ScreenShare
  );

  if (!videoTrack) {
    return (
      <div className="flex flex-col items-center justify-center h-48 gap-2 text-gray-400">
        <VideoOff className="h-8 w-8" />
        <span className="text-sm">Waiting for {feedName}...</span>
      </div>
    );
  }

  return (
    <div className="relative">
      <VideoTrack
        trackRef={videoTrack}
        className="w-full h-auto rounded-lg"
      />
      <div className="absolute top-2 left-2 bg-black/60 rounded px-2 py-1 text-xs text-white flex items-center gap-1">
        <Video className="h-3 w-3" />
        {feedName}
      </div>
    </div>
  );
}

"use client";

import { useCallback } from "react";
import {
  LiveKitRoom,
  VideoConference,
  RoomAudioRenderer,
} from "@livekit/components-react";
import "@livekit/components-styles";

interface CallViewProps {
  token: string;
  serverUrl: string;
  roomName: string;
  onDisconnected?: () => void;
}

export function CallView({
  token,
  serverUrl,
  onDisconnected,
}: CallViewProps) {
  const handleDisconnected = useCallback(() => {
    onDisconnected?.();
  }, [onDisconnected]);

  return (
    <div className="h-full w-full bg-gray-900 rounded-lg overflow-hidden">
      <LiveKitRoom
        token={token}
        serverUrl={serverUrl}
        connect={true}
        audio={true}
        video={true}
        onDisconnected={handleDisconnected}
      >
        <VideoConference />
        <RoomAudioRenderer />
      </LiveKitRoom>
    </div>
  );
}

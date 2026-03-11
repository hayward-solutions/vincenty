"use client";

import { useCallback, useState } from "react";
import {
  LiveKitRoom,
  RoomAudioRenderer,
  useLocalParticipant,
} from "@livekit/components-react";
import { Mic } from "lucide-react";
import { usePTTFloor } from "@/lib/hooks/use-ptt";
import { cn } from "@/lib/utils";

interface PTTButtonProps {
  token: string;
  serverUrl: string;
  channelId: string;
  channelName: string;
}

export function PTTButton({
  token,
  serverUrl,
  channelId,
  channelName,
}: PTTButtonProps) {
  return (
    <LiveKitRoom
      token={token}
      serverUrl={serverUrl}
      connect={true}
      audio={true}
      video={false}
    >
      <PTTButtonInner channelId={channelId} channelName={channelName} />
      <RoomAudioRenderer />
    </LiveKitRoom>
  );
}

function PTTButtonInner({
  channelId,
  channelName,
}: {
  channelId: string;
  channelName: string;
}) {
  const { localParticipant } = useLocalParticipant();
  const { floorHolder, requestFloor, releaseFloor } = usePTTFloor(channelId);
  const [isTalking, setIsTalking] = useState(false);

  const handlePress = useCallback(async () => {
    requestFloor();
    await localParticipant.setMicrophoneEnabled(true);
    setIsTalking(true);
  }, [localParticipant, requestFloor]);

  const handleRelease = useCallback(async () => {
    await localParticipant.setMicrophoneEnabled(false);
    releaseFloor();
    setIsTalking(false);
  }, [localParticipant, releaseFloor]);

  return (
    <div className="flex flex-col items-center gap-2">
      <span className="text-sm text-muted-foreground">{channelName}</span>

      <button
        type="button"
        onPointerDown={handlePress}
        onPointerUp={handleRelease}
        onPointerLeave={handleRelease}
        className={cn(
          "rounded-full p-6 transition-all select-none touch-none",
          isTalking
            ? "bg-green-600 scale-110 shadow-lg shadow-green-500/30"
            : "bg-gray-700 hover:bg-gray-600"
        )}
      >
        <Mic
          className={cn(
            "h-8 w-8",
            isTalking ? "text-white" : "text-gray-300"
          )}
        />
      </button>

      <span className="text-xs text-muted-foreground">
        {isTalking
          ? "Speaking..."
          : floorHolder
            ? `${floorHolder.name} is speaking`
            : "Hold to talk"}
      </span>
    </div>
  );
}

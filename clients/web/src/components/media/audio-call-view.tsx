"use client";

import { useCallback } from "react";
import {
  LiveKitRoom,
  RoomAudioRenderer,
  useLocalParticipant,
  useParticipants,
  useRoomContext,
} from "@livekit/components-react";
import "@livekit/components-styles";
import { Mic, MicOff, PhoneOff, Users } from "lucide-react";

interface AudioCallViewProps {
  token: string;
  serverUrl: string;
  roomName: string;
  onDisconnected?: () => void;
}

export function AudioCallView({
  token,
  serverUrl,
  roomName,
  onDisconnected,
}: AudioCallViewProps) {
  return (
    <div className="bg-gray-900 rounded-lg p-6">
      <LiveKitRoom
        token={token}
        serverUrl={serverUrl}
        connect={true}
        audio={true}
        video={false}
        onDisconnected={onDisconnected}
      >
        <AudioCallContent roomName={roomName} />
        <RoomAudioRenderer />
      </LiveKitRoom>
    </div>
  );
}

function AudioCallContent({ roomName }: { roomName: string }) {
  const room = useRoomContext();
  const { localParticipant, isMicrophoneEnabled } = useLocalParticipant();
  const participants = useParticipants();

  const toggleMic = useCallback(async () => {
    await localParticipant.setMicrophoneEnabled(!isMicrophoneEnabled);
  }, [localParticipant, isMicrophoneEnabled]);

  return (
    <div className="flex flex-col items-center gap-4">
      <h3 className="text-lg font-medium text-white">{roomName}</h3>

      <div className="flex items-center gap-2 text-gray-300">
        <Users className="h-4 w-4" />
        <span>{participants.length} participant{participants.length !== 1 ? "s" : ""}</span>
      </div>

      <div className="flex flex-wrap gap-2 justify-center">
        {participants.map((p) => (
          <div
            key={p.identity}
            className="bg-gray-800 rounded-full px-3 py-1 text-sm text-gray-200"
          >
            {p.name || p.identity}
          </div>
        ))}
      </div>

      <div className="flex gap-3 mt-4">
        <button
          type="button"
          onClick={toggleMic}
          className={`rounded-full p-3 ${
            isMicrophoneEnabled
              ? "bg-gray-700 hover:bg-gray-600"
              : "bg-red-600 hover:bg-red-500"
          }`}
        >
          {isMicrophoneEnabled ? (
            <Mic className="h-5 w-5 text-white" />
          ) : (
            <MicOff className="h-5 w-5 text-white" />
          )}
        </button>

        <button
          type="button"
          onClick={() => room.disconnect()}
          className="rounded-full p-3 bg-red-600 hover:bg-red-500"
        >
          <PhoneOff className="h-5 w-5 text-white" />
        </button>
      </div>
    </div>
  );
}

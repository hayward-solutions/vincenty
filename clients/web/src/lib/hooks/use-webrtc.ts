"use client";

import { useCallback, useRef, useState } from "react";

// ---------------------------------------------------------------------------
// WHIP Publishing (browser → MediaMTX)
// ---------------------------------------------------------------------------

export function useWebRTCPublish() {
  const [isPublishing, setIsPublishing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const resourceUrlRef = useRef<string | null>(null);

  const publish = useCallback(
    async (mediaStream: MediaStream, whipUrl: string) => {
      setError(null);

      try {
        const pc = new RTCPeerConnection({
          iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
        });
        pcRef.current = pc;

        // Add all tracks from the media stream
        for (const track of mediaStream.getTracks()) {
          pc.addTrack(track, mediaStream);
        }

        // Create and set local SDP offer
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);

        // Wait for ICE gathering to complete (or timeout)
        await waitForIceGathering(pc, 5000);

        const localSDP = pc.localDescription;
        if (!localSDP) {
          throw new Error("Failed to create local SDP");
        }

        // Send the offer to the WHIP endpoint
        const response = await fetch(whipUrl, {
          method: "POST",
          headers: { "Content-Type": "application/sdp" },
          body: localSDP.sdp,
        });

        if (!response.ok) {
          const text = await response.text().catch(() => response.statusText);
          throw new Error(`WHIP publish failed: ${response.status} ${text}`);
        }

        // Store the resource URL for later teardown
        const location = response.headers.get("Location");
        if (location) {
          resourceUrlRef.current = new URL(location, whipUrl).href;
        }

        // Set the remote SDP answer
        const answerSDP = await response.text();
        await pc.setRemoteDescription(
          new RTCSessionDescription({ type: "answer", sdp: answerSDP })
        );

        setIsPublishing(true);

        // Monitor connection state
        pc.onconnectionstatechange = () => {
          if (
            pc.connectionState === "failed" ||
            pc.connectionState === "disconnected" ||
            pc.connectionState === "closed"
          ) {
            setIsPublishing(false);
          }
        };
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to publish stream";
        setError(message);
        pcRef.current?.close();
        pcRef.current = null;
        setIsPublishing(false);
        throw err;
      }
    },
    []
  );

  const stop = useCallback(async () => {
    // Teardown the WHIP resource if we have one
    if (resourceUrlRef.current) {
      try {
        await fetch(resourceUrlRef.current, { method: "DELETE" });
      } catch {
        // Best-effort teardown
      }
      resourceUrlRef.current = null;
    }

    if (pcRef.current) {
      pcRef.current.close();
      pcRef.current = null;
    }
    setIsPublishing(false);
    setError(null);
  }, []);

  return { publish, stop, isPublishing, error };
}

// ---------------------------------------------------------------------------
// WHEP Viewing (MediaMTX → browser)
// ---------------------------------------------------------------------------

export function useWebRTCView() {
  const [isConnected, setIsConnected] = useState(false);
  const [mediaStream, setMediaStream] = useState<MediaStream | null>(null);
  const [error, setError] = useState<string | null>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const resourceUrlRef = useRef<string | null>(null);

  const connect = useCallback(async (whepUrl: string) => {
    setError(null);

    try {
      const pc = new RTCPeerConnection({
        iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
      });
      pcRef.current = pc;

      // Set up receiving tracks
      const stream = new MediaStream();
      pc.ontrack = (event) => {
        stream.addTrack(event.track);
        setMediaStream(new MediaStream(stream.getTracks()));
      };

      // We need to add transceivers for receiving
      pc.addTransceiver("video", { direction: "recvonly" });
      pc.addTransceiver("audio", { direction: "recvonly" });

      // Create and set local SDP offer
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      // Wait for ICE gathering
      await waitForIceGathering(pc, 5000);

      const localSDP = pc.localDescription;
      if (!localSDP) {
        throw new Error("Failed to create local SDP");
      }

      // Send the offer to the WHEP endpoint
      const response = await fetch(whepUrl, {
        method: "POST",
        headers: { "Content-Type": "application/sdp" },
        body: localSDP.sdp,
      });

      if (!response.ok) {
        const text = await response.text().catch(() => response.statusText);
        throw new Error(`WHEP connect failed: ${response.status} ${text}`);
      }

      // Store the resource URL for later teardown
      const location = response.headers.get("Location");
      if (location) {
        resourceUrlRef.current = new URL(location, whepUrl).href;
      }

      // Set the remote SDP answer
      const answerSDP = await response.text();
      await pc.setRemoteDescription(
        new RTCSessionDescription({ type: "answer", sdp: answerSDP })
      );

      setIsConnected(true);

      // Monitor connection state
      pc.onconnectionstatechange = () => {
        if (
          pc.connectionState === "failed" ||
          pc.connectionState === "disconnected" ||
          pc.connectionState === "closed"
        ) {
          setIsConnected(false);
          setMediaStream(null);
        }
      };
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to connect to stream";
      setError(message);
      pcRef.current?.close();
      pcRef.current = null;
      setIsConnected(false);
      throw err;
    }
  }, []);

  const disconnect = useCallback(async () => {
    // Teardown the WHEP resource if we have one
    if (resourceUrlRef.current) {
      try {
        await fetch(resourceUrlRef.current, { method: "DELETE" });
      } catch {
        // Best-effort teardown
      }
      resourceUrlRef.current = null;
    }

    if (pcRef.current) {
      pcRef.current.close();
      pcRef.current = null;
    }
    setIsConnected(false);
    setMediaStream(null);
    setError(null);
  }, []);

  return { connect, disconnect, mediaStream, isConnected, error };
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Wait for ICE gathering to complete or timeout. */
function waitForIceGathering(
  pc: RTCPeerConnection,
  timeoutMs: number
): Promise<void> {
  return new Promise<void>((resolve) => {
    if (pc.iceGatheringState === "complete") {
      resolve();
      return;
    }

    const timeout = setTimeout(() => {
      resolve(); // Proceed with what we have
    }, timeoutMs);

    pc.onicegatheringstatechange = () => {
      if (pc.iceGatheringState === "complete") {
        clearTimeout(timeout);
        resolve();
      }
    };
  });
}

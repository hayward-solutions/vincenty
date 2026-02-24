"use client";

import { useEffect, useRef, useState } from "react";
import maplibregl from "maplibre-gl";
import {
  useStreams,
  useLiveStreamLocations,
} from "@/lib/hooks/use-streams";
import { StreamViewerDialog } from "@/components/streams/stream-panel";
import type { StreamResponse } from "@/types/api";

interface StreamMarkersProps {
  map: maplibregl.Map;
}

/**
 * StreamMarkers renders pulsing map markers for each active live stream
 * that has a known location. Clicking a marker opens the stream viewer.
 */
export function StreamMarkers({ map }: StreamMarkersProps) {
  const { streams } = useStreams("live");
  const { locations } = useLiveStreamLocations();
  const markersRef = useRef<Map<string, maplibregl.Marker>>(new Map());
  const [viewingStream, setViewingStream] = useState<StreamResponse | null>(
    null
  );

  // Build a lookup from stream_id → stream
  const streamMap = useRef<Map<string, StreamResponse>>(new Map());
  useEffect(() => {
    streamMap.current = new Map(streams.map((s) => [s.id, s]));
  }, [streams]);

  useEffect(() => {
    const existingIds = new Set(markersRef.current.keys());

    for (const [streamId, loc] of locations) {
      existingIds.delete(streamId);

      const stream = streamMap.current.get(streamId);
      if (!stream) continue;

      const existing = markersRef.current.get(streamId);
      if (existing) {
        // Update position
        existing.setLngLat([loc.lng, loc.lat]);

        // Update popup content
        const popup = existing.getPopup();
        if (popup) {
          popup.setHTML(buildPopupHTML(stream));
        }
      } else {
        // Create new marker
        const el = createStreamMarkerElement(stream.title);

        const popup = new maplibregl.Popup({
          offset: 16,
          closeButton: false,
          maxWidth: "200px",
        }).setHTML(buildPopupHTML(stream));

        const marker = new maplibregl.Marker({ element: el, anchor: "center" })
          .setLngLat([loc.lng, loc.lat])
          .setPopup(popup)
          .addTo(map);

        // Click to open viewer
        el.addEventListener("click", (e) => {
          e.stopPropagation();
          const currentStream = streamMap.current.get(streamId);
          if (currentStream) {
            setViewingStream(currentStream);
          }
        });

        markersRef.current.set(streamId, marker);
      }
    }

    // Remove markers for streams that no longer have a location
    for (const streamId of existingIds) {
      const marker = markersRef.current.get(streamId);
      if (marker) {
        marker.remove();
        markersRef.current.delete(streamId);
      }
    }
  }, [map, locations, streams]);

  // Also remove markers for streams that have ended
  useEffect(() => {
    const liveIds = new Set(streams.map((s) => s.id));
    for (const [streamId, marker] of markersRef.current) {
      if (!liveIds.has(streamId)) {
        marker.remove();
        markersRef.current.delete(streamId);
      }
    }
  }, [streams]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      for (const marker of markersRef.current.values()) {
        marker.remove();
      }
      markersRef.current.clear();
    };
  }, []);

  return (
    <>
      {/* Stream viewer dialog triggered by marker click */}
      <StreamViewerDialog
        stream={viewingStream}
        onClose={() => setViewingStream(null)}
      />
    </>
  );
}

function createStreamMarkerElement(title: string): HTMLElement {
  const wrapper = document.createElement("div");
  wrapper.className = "sa-stream-marker";
  wrapper.style.cssText =
    "position:relative;width:24px;height:24px;cursor:pointer;";

  // Outer pulsing ring
  const pulse = document.createElement("div");
  pulse.style.cssText = `
    position:absolute;top:-4px;left:-4px;
    width:32px;height:32px;border-radius:50%;
    background:rgba(239,68,68,0.3);
    animation:sa-pulse 1.5s ease-in-out infinite;
  `;
  wrapper.appendChild(pulse);

  // Inner red dot
  const dot = document.createElement("div");
  dot.style.cssText = `
    position:absolute;top:0;left:0;
    width:24px;height:24px;border-radius:50%;
    background:#ef4444;border:2px solid white;
    box-shadow:0 2px 6px rgba(0,0,0,0.4);
    display:flex;align-items:center;justify-content:center;
  `;

  // Video icon inside dot
  const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  icon.setAttribute("width", "12");
  icon.setAttribute("height", "12");
  icon.setAttribute("viewBox", "0 0 24 24");
  icon.setAttribute("fill", "none");
  icon.setAttribute("stroke", "white");
  icon.setAttribute("stroke-width", "2.5");
  icon.setAttribute("stroke-linecap", "round");
  icon.setAttribute("stroke-linejoin", "round");
  const path1 = document.createElementNS("http://www.w3.org/2000/svg", "path");
  path1.setAttribute("d", "m16 13 5.223 3.482a.5.5 0 0 0 .777-.416V7.87a.5.5 0 0 0-.752-.432L16 10.5");
  const rect = document.createElementNS("http://www.w3.org/2000/svg", "rect");
  rect.setAttribute("x", "2");
  rect.setAttribute("y", "6");
  rect.setAttribute("width", "14");
  rect.setAttribute("height", "12");
  rect.setAttribute("rx", "2");
  icon.appendChild(path1);
  icon.appendChild(rect);
  dot.appendChild(icon);
  wrapper.appendChild(dot);

  // Label below
  const text = document.createElement("div");
  text.textContent = title;
  text.style.cssText = `
    position:absolute;top:26px;left:50%;transform:translateX(-50%);
    font-size:11px;font-weight:600;color:white;
    background:rgba(239,68,68,0.85);padding:1px 5px;
    border-radius:3px;white-space:nowrap;
    max-width:120px;overflow:hidden;text-overflow:ellipsis;
  `;
  wrapper.appendChild(text);

  // Inject pulse keyframes if not already present
  if (!document.getElementById("sa-stream-pulse-style")) {
    const style = document.createElement("style");
    style.id = "sa-stream-pulse-style";
    style.textContent = `
      @keyframes sa-pulse {
        0%, 100% { transform: scale(1); opacity: 0.6; }
        50% { transform: scale(1.4); opacity: 0; }
      }
    `;
    document.head.appendChild(style);
  }

  return wrapper;
}

function buildPopupHTML(stream: StreamResponse): string {
  const name = stream.display_name || stream.username || "Unknown";
  return [
    `<div style="font-weight:600;color:#ef4444;margin-bottom:2px">${escapeHtml(stream.title)}</div>`,
    `<div style="font-size:12px;color:#aaa">${escapeHtml(name)}</div>`,
    `<div style="font-size:11px;color:#888;margin-top:2px">${stream.source_type} stream</div>`,
  ].join("");
}

function escapeHtml(s: string): string {
  const div = document.createElement("div");
  div.textContent = s;
  return div.innerHTML;
}

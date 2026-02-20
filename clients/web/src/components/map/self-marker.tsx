"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";

interface SelfMarkerProps {
  map: maplibregl.Map;
  position: { lat: number; lng: number; heading: number | null } | null;
  /** If true, fly to the user's position on first fix (once). */
  autoCenter?: boolean;
}

/**
 * SelfMarker renders a blue pulsing dot for the current user's own position.
 * It uses the browser Geolocation position (from useLocationSharing) rather
 * than server-echoed data.
 */
export function SelfMarker({ map, position, autoCenter = true }: SelfMarkerProps) {
  const markerRef = useRef<maplibregl.Marker | null>(null);
  const hasCenteredRef = useRef(false);

  useEffect(() => {
    if (!position) return;

    if (!markerRef.current) {
      // Create the marker element
      const el = createSelfElement();

      const popup = new maplibregl.Popup({
        offset: 20,
        closeButton: false,
        maxWidth: "180px",
      });

      markerRef.current = new maplibregl.Marker({ element: el, anchor: "center" })
        .setLngLat([position.lng, position.lat])
        .setPopup(popup)
        .addTo(map);
    }

    // Update position
    markerRef.current.setLngLat([position.lng, position.lat]);

    // Update popup content
    const popup = markerRef.current.getPopup();
    if (popup) {
      popup.setHTML(
        `<div style="font-weight:600;color:#3b82f6;margin-bottom:2px">You</div>` +
        `<div style="font-size:12px">${position.lat.toFixed(5)}, ${position.lng.toFixed(5)}</div>`
      );
    }

    // Auto-center on first fix
    if (autoCenter && !hasCenteredRef.current) {
      hasCenteredRef.current = true;
      map.flyTo({
        center: [position.lng, position.lat],
        zoom: Math.max(map.getZoom(), 14),
        duration: 1500,
      });
    }
  }, [map, position, autoCenter]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (markerRef.current) {
        markerRef.current.remove();
        markerRef.current = null;
      }
    };
  }, []);

  return null;
}

function createSelfElement(): HTMLElement {
  const wrapper = document.createElement("div");
  wrapper.style.cssText = "position:relative;width:22px;height:22px;cursor:pointer;";

  // Pulsing ring
  const pulse = document.createElement("div");
  pulse.style.cssText = `
    position:absolute;inset:0;border-radius:50%;
    background:rgba(59,130,246,0.25);
    animation:sa-pulse 2s ease-out infinite;
  `;
  wrapper.appendChild(pulse);

  // Inner dot
  const dot = document.createElement("div");
  dot.style.cssText = `
    position:absolute;top:5px;left:5px;width:12px;height:12px;
    border-radius:50%;background:#3b82f6;
    border:2px solid white;box-shadow:0 1px 4px rgba(0,0,0,0.4);
  `;
  wrapper.appendChild(dot);

  // Inject keyframes if not already present
  if (!document.getElementById("sa-pulse-style")) {
    const style = document.createElement("style");
    style.id = "sa-pulse-style";
    style.textContent = `
      @keyframes sa-pulse {
        0% { transform:scale(1); opacity:1; }
        100% { transform:scale(2.5); opacity:0; }
      }
    `;
    document.head.appendChild(style);
  }

  return wrapper;
}

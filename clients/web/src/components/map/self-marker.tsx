"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import { createMarkerSVG } from "./marker-shapes";

interface SelfMarkerProps {
  map: maplibregl.Map;
  position: { lat: number; lng: number; heading: number | null } | null;
  /** If true, fly to the user's position on first fix (once). */
  autoCenter?: boolean;
  /** Marker shape name (default: "circle") */
  icon?: string;
  /** Marker color as hex string (default: "#3b82f6") */
  color?: string;
}

// Inject the CSS pulse animation once (idempotent)
let styleInjected = false;
function injectPulseStyle() {
  if (styleInjected || typeof document === "undefined") return;
  const style = document.createElement("style");
  style.textContent = `
    @keyframes sa-self-pulse {
      0%   { transform: translate(-50%,-50%) scale(1); opacity: 0.4; }
      100% { transform: translate(-50%,-50%) scale(2.5); opacity: 0; }
    }
    .sa-self-pulse {
      animation: sa-self-pulse 2s ease-out infinite;
    }
  `;
  document.head.appendChild(style);
  styleInjected = true;
}

/**
 * SelfMarker renders the current user's position on the map using a
 * DOM-based SVG marker with a CSS pulse animation ring behind it.
 * The shape and color are configurable via props.
 */
export function SelfMarker({
  map,
  position,
  autoCenter = true,
  icon = "circle",
  color = "#3b82f6",
}: SelfMarkerProps) {
  const markerRef = useRef<maplibregl.Marker | null>(null);
  const hasCenteredRef = useRef(false);
  // Track current style to detect prop changes
  const styleKeyRef = useRef("");

  useEffect(() => {
    if (!position) return;

    injectPulseStyle();

    const currentStyleKey = `${icon}:${color}`;

    if (markerRef.current && styleKeyRef.current === currentStyleKey) {
      // Same style — just update position
      markerRef.current.setLngLat([position.lng, position.lat]);
    } else {
      // Style changed or first render — (re)create marker
      if (markerRef.current) {
        markerRef.current.remove();
      }

      const el = createSelfMarkerElement(icon, color);

      const popup = new maplibregl.Popup({
        offset: 12,
        closeButton: false,
        maxWidth: "180px",
      }).setHTML(
        `<div style="font-weight:600;color:${color};margin-bottom:2px">You</div>` +
          `<div style="font-size:12px">${position.lat.toFixed(5)}, ${position.lng.toFixed(5)}</div>`
      );

      markerRef.current = new maplibregl.Marker({
        element: el,
        anchor: "center",
      })
        .setLngLat([position.lng, position.lat])
        .setPopup(popup)
        .addTo(map);

      styleKeyRef.current = currentStyleKey;
    }

    // Update popup content with latest coordinates
    const popup = markerRef.current?.getPopup();
    if (popup) {
      popup.setHTML(
        `<div style="font-weight:600;color:${color};margin-bottom:2px">You</div>` +
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
  }, [map, position, autoCenter, icon, color]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      markerRef.current?.remove();
      markerRef.current = null;
    };
  }, []);

  return null;
}

/**
 * Creates a DOM element for the self-marker:
 * - A pulse ring (CSS animated div) behind the icon
 * - The SVG shape on top
 */
function createSelfMarkerElement(icon: string, color: string): HTMLElement {
  const wrapper = document.createElement("div");
  wrapper.style.cssText =
    "position:relative;width:24px;height:24px;cursor:pointer;";

  // Pulse ring — positioned behind the icon, centered
  const pulse = document.createElement("div");
  pulse.className = "sa-self-pulse";
  pulse.style.cssText = `
    position:absolute;top:50%;left:50%;
    width:24px;height:24px;border-radius:50%;
    background:${color};
    transform:translate(-50%,-50%) scale(1);
    pointer-events:none;
  `;
  wrapper.appendChild(pulse);

  // SVG shape — centered in the wrapper
  const svg = createMarkerSVG(icon, color, 22);
  svg.style.cssText =
    "position:absolute;top:1px;left:1px;filter:drop-shadow(0 1px 3px rgba(0,0,0,0.4));";
  wrapper.appendChild(svg);

  return wrapper;
}

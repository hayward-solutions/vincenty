"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import type { UserLocation } from "@/lib/hooks/use-locations";
import type { Group } from "@/types/api";
import { createMarkerSVG } from "./marker-shapes";

interface LocationMarkersProps {
  map: maplibregl.Map;
  locations: Map<string, UserLocation>;
  /** Current user's ID — their marker is excluded */
  currentUserId?: string;
  /** Group config lookup by group_id — provides marker_icon and marker_color */
  groups?: Map<string, Group>;
}

// Fallback colors when no group config is available (cycling for multiple users)
const FALLBACK_COLORS = [
  "#3b82f6", // blue
  "#ef4444", // red
  "#22c55e", // green
  "#f59e0b", // amber
  "#8b5cf6", // violet
  "#ec4899", // pink
  "#06b6d4", // cyan
  "#f97316", // orange
];

function getFallbackColor(index: number): string {
  return FALLBACK_COLORS[index % FALLBACK_COLORS.length];
}

function formatAge(timestamp: string): string {
  const age = Date.now() - new Date(timestamp).getTime();
  if (age < 60_000) return "just now";
  if (age < 3_600_000) return `${Math.floor(age / 60_000)}m ago`;
  return `${Math.floor(age / 3_600_000)}h ago`;
}

/** Resolve the icon shape and color for a location entry. */
function resolveMarkerStyle(
  loc: UserLocation,
  groups: Map<string, Group> | undefined,
  fallbackColorIndex: number
): { icon: string; color: string } {
  if (groups && loc.group_id) {
    const group = groups.get(loc.group_id);
    if (group) {
      return {
        icon: group.marker_icon || "circle",
        color: group.marker_color || "#3b82f6",
      };
    }
  }
  return { icon: "circle", color: getFallbackColor(fallbackColorIndex) };
}

/**
 * LocationMarkers manages MapLibre markers for all tracked user locations.
 * It creates/updates/removes markers as the locations map changes.
 *
 * When a `groups` map is provided, markers are rendered using each group's
 * configured icon shape and color. Otherwise, falls back to colored circles.
 */
export function LocationMarkers({
  map,
  locations,
  currentUserId,
  groups,
}: LocationMarkersProps) {
  const markersRef = useRef<Map<string, maplibregl.Marker>>(new Map());
  const colorIndexRef = useRef<Map<string, number>>(new Map());
  const nextColorRef = useRef(0);
  // Track per-marker style to detect when we need to recreate the element
  const styleRef = useRef<Map<string, string>>(new Map());

  useEffect(() => {
    const existingIds = new Set(markersRef.current.keys());

    for (const [userId, loc] of locations) {
      // Skip current user
      if (userId === currentUserId) continue;

      existingIds.delete(userId);

      // Assign a stable fallback color index per user
      if (!colorIndexRef.current.has(userId)) {
        colorIndexRef.current.set(userId, nextColorRef.current++);
      }

      const { icon, color } = resolveMarkerStyle(
        loc,
        groups,
        colorIndexRef.current.get(userId)!
      );
      const styleKey = `${icon}:${color}`;

      const existing = markersRef.current.get(userId);
      const prevStyle = styleRef.current.get(userId);

      if (existing && prevStyle === styleKey) {
        // Same style — just update position and popup
        existing.setLngLat([loc.lng, loc.lat]);

        const popup = existing.getPopup();
        if (popup) {
          popup.setHTML(buildPopupHTML(loc, color));
        }

        if (loc.heading != null) {
          existing.setRotation(loc.heading);
        }
      } else {
        // Style changed or new marker — (re)create
        if (existing) {
          existing.remove();
          markersRef.current.delete(userId);
        }

        const el = createMarkerElement(
          loc.display_name || loc.username,
          icon,
          color
        );

        const popup = new maplibregl.Popup({
          offset: 12,
          closeButton: false,
          maxWidth: "220px",
        }).setHTML(buildPopupHTML(loc, color));

        const marker = new maplibregl.Marker({ element: el, anchor: "center" })
          .setLngLat([loc.lng, loc.lat])
          .setPopup(popup)
          .addTo(map);

        if (loc.heading != null) {
          marker.setRotation(loc.heading);
        }

        markersRef.current.set(userId, marker);
        styleRef.current.set(userId, styleKey);
      }
    }

    // Remove markers for users no longer in the locations map
    for (const userId of existingIds) {
      const marker = markersRef.current.get(userId);
      if (marker) {
        marker.remove();
        markersRef.current.delete(userId);
        styleRef.current.delete(userId);
      }
    }
  }, [map, locations, currentUserId, groups]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      for (const marker of markersRef.current.values()) {
        marker.remove();
      }
      markersRef.current.clear();
      styleRef.current.clear();
    };
  }, []);

  return null; // markers are managed imperatively
}

function createMarkerElement(
  label: string,
  icon: string,
  color: string
): HTMLElement {
  const wrapper = document.createElement("div");
  wrapper.className = "sa-marker";
<<<<<<< Updated upstream
  wrapper.style.cssText = "position:relative;width:18px;height:18px;cursor:pointer;";

  // Pin — centered in the wrapper, determines the anchor point
  const pin = document.createElement("div");
  pin.style.cssText = `
    position:absolute;top:2px;left:2px;
    width:14px;height:14px;border-radius:50%;
    background:${color};border:2px solid white;
    box-shadow:0 1px 4px rgba(0,0,0,0.4);
  `;
  wrapper.appendChild(pin);
=======
  wrapper.style.cssText =
    "display:flex;flex-direction:column;align-items:center;cursor:pointer;";

  // Shape icon
  const svg = createMarkerSVG(icon, color, 18);
  svg.style.filter = "drop-shadow(0 1px 3px rgba(0,0,0,0.4))";
  wrapper.appendChild(svg);
>>>>>>> Stashed changes

  // Label — absolutely positioned below the pin, outside the wrapper's layout
  const text = document.createElement("div");
  text.textContent = label;
  text.style.cssText = `
    position:absolute;top:20px;left:50%;transform:translateX(-50%);
    font-size:11px;font-weight:600;color:white;
    background:rgba(0,0,0,0.7);padding:1px 5px;
    border-radius:3px;white-space:nowrap;
    max-width:120px;overflow:hidden;text-overflow:ellipsis;
  `;
  wrapper.appendChild(text);

  return wrapper;
}

function buildPopupHTML(loc: UserLocation, color: string): string {
  const name = loc.display_name || loc.username;
  const parts = [
    `<div style="font-weight:600;color:${color};margin-bottom:4px">${escapeHtml(name)}</div>`,
    `<div style="font-size:12px;color:#aaa">@${escapeHtml(loc.username)}</div>`,
    `<div style="font-size:12px;margin-top:4px">`,
    `${loc.lat.toFixed(5)}, ${loc.lng.toFixed(5)}`,
    `</div>`,
  ];

  if (loc.speed != null) {
    parts.push(
      `<div style="font-size:12px">${(loc.speed * 3.6).toFixed(1)} km/h</div>`
    );
  }
  if (loc.heading != null) {
    parts.push(
      `<div style="font-size:12px">Heading: ${loc.heading.toFixed(0)}&deg;</div>`
    );
  }

  parts.push(
    `<div style="font-size:11px;color:#888;margin-top:4px">${formatAge(loc.timestamp)}</div>`
  );

  return parts.join("");
}

function escapeHtml(s: string): string {
  const div = document.createElement("div");
  div.textContent = s;
  return div.innerHTML;
}

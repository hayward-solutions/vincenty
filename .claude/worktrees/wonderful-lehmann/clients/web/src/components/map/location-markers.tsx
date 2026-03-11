"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import type { UserLocation } from "@/lib/hooks/use-locations";
import type { Group } from "@/types/api";
import { createMarkerSVG } from "./marker-shapes";

interface LocationMarkersProps {
  map: maplibregl.Map;
  /** Map keyed by device_id */
  locations: Map<string, UserLocation>;
  /** Current device ID — its marker is excluded (rendered by SelfMarker instead) */
  currentDeviceId?: string;
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
 * LocationMarkers manages MapLibre markers for all tracked device locations.
 * It creates/updates/removes markers as the locations map changes.
 *
 * The locations map is keyed by device_id, so each device gets its own marker.
 * When a `groups` map is provided, markers are rendered using each group's
 * configured icon shape and color. Otherwise, falls back to colored circles.
 */
export function LocationMarkers({
  map,
  locations,
  currentDeviceId,
  groups,
}: LocationMarkersProps) {
  const markersRef = useRef<Map<string, maplibregl.Marker>>(new Map());
  const colorIndexRef = useRef<Map<string, number>>(new Map());
  const nextColorRef = useRef(0);
  // Track per-marker style to detect when we need to recreate the element
  const styleRef = useRef<Map<string, string>>(new Map());

  useEffect(() => {
    const existingIds = new Set(markersRef.current.keys());

    for (const [deviceId, loc] of locations) {
      // Skip current device (handled by SelfMarker)
      if (deviceId === currentDeviceId) continue;

      existingIds.delete(deviceId);

      // Assign a stable fallback color index per user (not per device)
      // so all devices of the same user share a color
      if (!colorIndexRef.current.has(loc.user_id)) {
        colorIndexRef.current.set(loc.user_id, nextColorRef.current++);
      }

      const { icon, color } = resolveMarkerStyle(
        loc,
        groups,
        colorIndexRef.current.get(loc.user_id)!
      );
      const styleKey = `${icon}:${color}`;

      const existing = markersRef.current.get(deviceId);
      const prevStyle = styleRef.current.get(deviceId);

      if (existing && prevStyle === styleKey) {
        // Same style — just update position and popup
        existing.setLngLat([loc.lng, loc.lat]);

        const popup = existing.getPopup();
        if (popup) {
          popup.setHTML(buildPopupHTML(loc, color));
        }

        applyIconRotation(existing, loc.heading ?? null);
      } else {
        // Style changed or new marker — (re)create
        if (existing) {
          existing.remove();
          markersRef.current.delete(deviceId);
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

        const marker = new maplibregl.Marker({ element: el, anchor: "center", subpixelPositioning: true })
          .setLngLat([loc.lng, loc.lat])
          .setPopup(popup)
          .addTo(map);

        applyIconRotation(marker, loc.heading ?? null);

        markersRef.current.set(deviceId, marker);
        styleRef.current.set(deviceId, styleKey);
      }
    }

    // Remove markers for devices no longer in the locations map
    for (const deviceId of existingIds) {
      const marker = markersRef.current.get(deviceId);
      if (marker) {
        marker.remove();
        markersRef.current.delete(deviceId);
        styleRef.current.delete(deviceId);
      }
    }
  }, [map, locations, currentDeviceId, groups]);

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

/**
 * Rotate only the SVG icon to reflect the device's heading, leaving the
 * wrapper and label untouched so the label stays screen-upright and centred
 * directly below the icon at all headings.
 */
function applyIconRotation(marker: maplibregl.Marker, heading: number | null): void {
  const icon = marker.getElement().querySelector<SVGSVGElement>(".sa-marker-icon");
  if (!icon) return;
  icon.style.transform = heading != null ? `rotate(${heading}deg)` : "";
}

function createMarkerElement(
  label: string,
  icon: string,
  color: string
): HTMLElement {
  const wrapper = document.createElement("div");
  wrapper.className = "sa-marker";
  wrapper.style.cssText = "width:18px;height:18px;cursor:pointer;";

  // Shape icon — centered in the wrapper, determines the anchor point.
  // Rotation is applied to this element only (not the wrapper) so the label
  // position is unaffected by heading changes.
  const svg = createMarkerSVG(icon, color, 18);
  svg.setAttribute("class", "sa-marker-icon");
  svg.style.cssText = "position:absolute;top:0;left:0;transform-origin:center center;";
  svg.style.filter = "drop-shadow(0 1px 3px rgba(0,0,0,0.4))";
  wrapper.appendChild(svg);

  // Label — absolutely positioned below the pin, outside the wrapper's layout.
  // Because only the SVG icon rotates (not the wrapper), this stays horizontal
  // and centred below the icon regardless of heading.
  const text = document.createElement("div");
  text.className = "sa-marker-label";
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
  ];

  // Show device name in popup
  if (loc.device_name) {
    parts.push(
      `<div style="font-size:12px;color:#bbb;margin-top:2px">${escapeHtml(loc.device_name)}${loc.is_primary ? ' <span style="color:#22c55e;font-size:10px">(primary)</span>' : ""}</div>`
    );
  }

  parts.push(
    `<div style="font-size:12px;margin-top:4px">`,
    `${loc.lat.toFixed(5)}, ${loc.lng.toFixed(5)}`,
    `</div>`
  );

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

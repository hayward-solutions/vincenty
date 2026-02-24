import { render, act } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";
import type { UserLocation } from "@/lib/hooks/use-locations";
import type { Group } from "@/types/api";

const mocks = vi.hoisted(() => {
  const popup = { setHTML: vi.fn() };
  const marker = {
    setLngLat: vi.fn().mockReturnThis(),
    setPopup: vi.fn().mockReturnThis(),
    addTo: vi.fn().mockReturnThis(),
    remove: vi.fn(),
    getPopup: vi.fn(() => popup),
    setRotation: vi.fn(),
  };
  return {
    Marker: vi.fn(function () { return { ...marker }; }),
    Popup: vi.fn(function () { return { ...popup }; }),
  };
});

vi.mock("maplibre-gl", () => ({
  default: { Marker: mocks.Marker, Popup: mocks.Popup },
}));

import { LocationMarkers } from "./location-markers";

function makeLoc(overrides: Partial<UserLocation> = {}): UserLocation {
  return {
    user_id: "u1",
    username: "alice",
    display_name: "Alice",
    device_id: "dev1",
    device_name: "Phone",
    is_primary: true,
    group_id: "g1",
    lat: 51.5,
    lng: -0.1,
    timestamp: new Date().toISOString(),
    ...overrides,
  };
}

const mockMap = {} as any;

describe("LocationMarkers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns null (renders nothing to the DOM)", () => {
    const locations = new Map<string, UserLocation>();
    const { container } = render(
      <LocationMarkers map={mockMap} locations={locations} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("creates markers for each location excluding currentDeviceId", () => {
    const locations = new Map<string, UserLocation>([
      ["dev1", makeLoc({ device_id: "dev1", user_id: "u1" })],
      ["dev2", makeLoc({ device_id: "dev2", user_id: "u2", username: "bob", display_name: "Bob" })],
      ["dev3", makeLoc({ device_id: "dev3", user_id: "u3", username: "carol", display_name: "Carol" })],
    ]);

    render(
      <LocationMarkers
        map={mockMap}
        locations={locations}
        currentDeviceId="dev1"
      />,
    );

    // Should create markers for dev2 and dev3 only
    expect(mocks.Marker).toHaveBeenCalledTimes(2);
    expect(mocks.Popup).toHaveBeenCalledTimes(2);
  });

  it("does not create a marker for currentDeviceId", () => {
    const locations = new Map<string, UserLocation>([
      ["dev1", makeLoc({ device_id: "dev1" })],
    ]);

    render(
      <LocationMarkers
        map={mockMap}
        locations={locations}
        currentDeviceId="dev1"
      />,
    );

    expect(mocks.Marker).not.toHaveBeenCalled();
  });

  it("removes markers for devices no longer in locations map", () => {
    const loc1 = makeLoc({ device_id: "dev1", user_id: "u1" });
    const loc2 = makeLoc({ device_id: "dev2", user_id: "u2", username: "bob" });
    const locations1 = new Map<string, UserLocation>([
      ["dev1", loc1],
      ["dev2", loc2],
    ]);

    const { rerender } = render(
      <LocationMarkers map={mockMap} locations={locations1} />,
    );

    expect(mocks.Marker).toHaveBeenCalledTimes(2);

    // Grab a reference to the remove mock from one of the created markers
    const markerInstances = mocks.Marker.mock.results.map((r: any) => r.value);

    // Now rerender with only dev1
    const locations2 = new Map<string, UserLocation>([["dev1", loc1]]);
    rerender(<LocationMarkers map={mockMap} locations={locations2} />);

    // The marker for dev2 should have been removed
    const removeCallCount = markerInstances.reduce(
      (count: number, m: any) => count + m.remove.mock.calls.length,
      0,
    );
    expect(removeCallCount).toBeGreaterThanOrEqual(1);
  });

  it("updates position without recreating marker when style is unchanged", () => {
    const loc = makeLoc({ device_id: "dev1", user_id: "u1" });
    const locations1 = new Map<string, UserLocation>([["dev1", loc]]);

    const { rerender } = render(
      <LocationMarkers map={mockMap} locations={locations1} />,
    );

    expect(mocks.Marker).toHaveBeenCalledTimes(1);
    const createdMarker = mocks.Marker.mock.results[0].value;

    // Rerender with updated lat/lng but same group → same style
    const updatedLoc = makeLoc({ device_id: "dev1", user_id: "u1", lat: 52.0, lng: 0.5 });
    const locations2 = new Map<string, UserLocation>([["dev1", updatedLoc]]);
    rerender(<LocationMarkers map={mockMap} locations={locations2} />);

    // Should NOT have created a new marker — still just 1 call
    expect(mocks.Marker).toHaveBeenCalledTimes(1);

    // But should have updated position
    expect(createdMarker.setLngLat).toHaveBeenCalledWith([0.5, 52.0]);
  });

  it("cleans up all markers on unmount", () => {
    const locations = new Map<string, UserLocation>([
      ["dev1", makeLoc({ device_id: "dev1", user_id: "u1" })],
      ["dev2", makeLoc({ device_id: "dev2", user_id: "u2", username: "bob" })],
    ]);

    const { unmount } = render(
      <LocationMarkers map={mockMap} locations={locations} />,
    );

    const markerInstances = mocks.Marker.mock.results.map((r: any) => r.value);

    unmount();

    for (const m of markerInstances) {
      expect(m.remove).toHaveBeenCalled();
    }
  });
});

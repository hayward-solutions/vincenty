import { describe, it, expect, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { dispatchWSMessage, clearWSSubscribers } from "@/test/test-utils";

import { useLocations } from "@/lib/hooks/use-locations";
import type { WSLocationBroadcast, WSLocationSnapshot } from "@/types/api";

afterEach(() => {
  clearWSSubscribers();
});

describe("useLocations", () => {
  it("starts with empty locations map", () => {
    const { result } = renderHook(() => useLocations());

    expect(result.current.locations.size).toBe(0);
  });

  it("updates locations on location_broadcast", async () => {
    const { result } = renderHook(() => useLocations());

    const broadcast: WSLocationBroadcast = {
      user_id: "user-1",
      username: "testuser",
      display_name: "Test User",
      device_id: "device-1",
      device_name: "Web Browser",
      is_primary: true,
      group_id: "group-1",
      lat: -33.8688,
      lng: 151.2093,
      timestamp: "2025-01-01T12:00:00Z",
    };

    act(() => {
      dispatchWSMessage("location_broadcast", broadcast);
    });

    // Wait for requestAnimationFrame flush
    await waitFor(() => {
      expect(result.current.locations.size).toBe(1);
    });

    const loc = result.current.locations.get("device-1");
    expect(loc?.lat).toBe(-33.8688);
    expect(loc?.lng).toBe(151.2093);
    expect(loc?.group_id).toBe("group-1");
  });

  it("updates locations on location_snapshot", async () => {
    const { result } = renderHook(() => useLocations());

    const snapshot: WSLocationSnapshot = {
      group_id: "group-1",
      locations: [
        {
          user_id: "user-1",
          username: "testuser",
          display_name: "Test User",
          device_id: "device-1",
          device_name: "Web Browser",
          is_primary: true,
          group_id: "group-1",
          lat: -33.8688,
          lng: 151.2093,
          timestamp: "2025-01-01T12:00:00Z",
        },
        {
          user_id: "user-2",
          username: "otheruser",
          display_name: "Other User",
          device_id: "device-2",
          device_name: "Phone",
          is_primary: true,
          group_id: "group-1",
          lat: -33.87,
          lng: 151.21,
          timestamp: "2025-01-01T12:00:00Z",
        },
      ],
    };

    act(() => {
      dispatchWSMessage("location_snapshot", snapshot);
    });

    await waitFor(() => {
      expect(result.current.locations.size).toBe(2);
    });
  });

  it("getGroupLocations filters by group_id", async () => {
    const { result } = renderHook(() => useLocations());

    act(() => {
      dispatchWSMessage("location_broadcast", {
        user_id: "user-1",
        username: "testuser",
        display_name: "Test User",
        device_id: "device-1",
        device_name: "Web",
        is_primary: true,
        group_id: "group-1",
        lat: 0,
        lng: 0,
        timestamp: "2025-01-01T12:00:00Z",
      });
      dispatchWSMessage("location_broadcast", {
        user_id: "user-2",
        username: "otheruser",
        display_name: "Other User",
        device_id: "device-2",
        device_name: "Phone",
        is_primary: true,
        group_id: "group-2",
        lat: 1,
        lng: 1,
        timestamp: "2025-01-01T12:00:00Z",
      });
    });

    await waitFor(() => {
      expect(result.current.locations.size).toBe(2);
    });

    const group1Locs = result.current.getGroupLocations("group-1");
    expect(group1Locs).toHaveLength(1);
    expect(group1Locs[0].device_id).toBe("device-1");

    const group2Locs = result.current.getGroupLocations("group-2");
    expect(group2Locs).toHaveLength(1);
    expect(group2Locs[0].device_id).toBe("device-2");
  });

  it("overwrites existing device location on update", async () => {
    const { result } = renderHook(() => useLocations());

    act(() => {
      dispatchWSMessage("location_broadcast", {
        user_id: "user-1",
        username: "testuser",
        display_name: "Test User",
        device_id: "device-1",
        device_name: "Web",
        is_primary: true,
        group_id: "group-1",
        lat: 0,
        lng: 0,
        timestamp: "2025-01-01T12:00:00Z",
      });
    });

    await waitFor(() => {
      expect(result.current.locations.get("device-1")?.lat).toBe(0);
    });

    act(() => {
      dispatchWSMessage("location_broadcast", {
        user_id: "user-1",
        username: "testuser",
        display_name: "Test User",
        device_id: "device-1",
        device_name: "Web",
        is_primary: true,
        group_id: "group-1",
        lat: 10,
        lng: 20,
        timestamp: "2025-01-01T12:01:00Z",
      });
    });

    await waitFor(() => {
      expect(result.current.locations.get("device-1")?.lat).toBe(10);
    });

    // Still only one entry
    expect(result.current.locations.size).toBe(1);
  });
});

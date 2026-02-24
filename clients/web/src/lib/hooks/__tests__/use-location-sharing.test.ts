import { describe, it, expect } from "vitest";
import { renderHook } from "@testing-library/react";
import React from "react";

// Import test-utils FIRST to activate the module mocks for auth-context and websocket-context.
// LocationProvider imports useAuth and useWebSocket, so these mocks must be registered before
// the module under test is imported.
import "@/test/test-utils";

import { useLocationSharing, LocationProvider } from "@/lib/hooks/use-location-sharing";

describe("useLocationSharing", () => {
  it("throws when used outside LocationProvider", () => {
    // Suppress console.error for this test
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    expect(() => {
      renderHook(() => useLocationSharing());
    }).toThrow("useLocationSharing must be used within a LocationProvider");

    spy.mockRestore();
  });

  it("provides default state within LocationProvider", () => {
    // Mock navigator.geolocation to prevent actual geolocation access
    const mockGeolocation = {
      watchPosition: vi.fn().mockReturnValue(1),
      clearWatch: vi.fn(),
      getCurrentPosition: vi.fn(),
    };
    Object.defineProperty(navigator, "geolocation", {
      value: mockGeolocation,
      writable: true,
    });

    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(LocationProvider, null, children);

    const { result } = renderHook(() => useLocationSharing(), { wrapper });

    expect(result.current.lastPosition).toBeNull();
    expect(result.current.error).toBeNull();
  });
});

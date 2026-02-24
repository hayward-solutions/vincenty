import { render, act } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";

const mocks = vi.hoisted(() => {
  const popup = { setHTML: vi.fn() };
  const marker = {
    setLngLat: vi.fn().mockReturnThis(),
    setPopup: vi.fn().mockReturnThis(),
    addTo: vi.fn().mockReturnThis(),
    remove: vi.fn(),
    getPopup: vi.fn(() => popup),
  };
  return {
    mockMap: { flyTo: vi.fn(), getZoom: vi.fn(() => 10) },
    Marker: vi.fn(function () { return { ...marker }; }),
    Popup: vi.fn(function () { return { ...popup }; }),
  };
});

vi.mock("maplibre-gl", () => ({
  default: { Marker: mocks.Marker, Popup: mocks.Popup },
}));

// Dynamic import to allow resetting module-level `styleInjected` between tests
async function loadSelfMarker() {
  const mod = await import("./self-marker");
  return mod.SelfMarker;
}

describe("SelfMarker", () => {
  let SelfMarker: Awaited<ReturnType<typeof loadSelfMarker>>;

  beforeEach(async () => {
    vi.clearAllMocks();
    // Reset module registry so `styleInjected` resets to false
    vi.resetModules();
    // Re-apply the mock after resetModules (hoisted mocks survive)
    vi.mock("maplibre-gl", () => ({
      default: {
        Marker: mocks.Marker,
        Popup: mocks.Popup,
      },
    }));
    const mod = await import("./self-marker");
    SelfMarker = mod.SelfMarker;
  });

  it("returns null when no position is provided", () => {
    const { container } = render(
      <SelfMarker map={mocks.mockMap as any} position={null} />,
    );
    expect(container.innerHTML).toBe("");
    expect(mocks.Marker).not.toHaveBeenCalled();
  });

  it("creates a marker when position is provided", () => {
    const position = { lat: 51.5, lng: -0.1, heading: null };
    render(
      <SelfMarker map={mocks.mockMap as any} position={position} />,
    );
    expect(mocks.Marker).toHaveBeenCalledTimes(1);
    expect(mocks.Popup).toHaveBeenCalledTimes(1);
    const markerInstance = mocks.Marker.mock.results[0].value;
    expect(markerInstance.setLngLat).toHaveBeenCalledWith([-0.1, 51.5]);
    expect(markerInstance.addTo).toHaveBeenCalledWith(mocks.mockMap);
  });

  it("flies to position on first fix when autoCenter=true", () => {
    const position = { lat: 51.5, lng: -0.1, heading: null };
    render(
      <SelfMarker
        map={mocks.mockMap as any}
        position={position}
        autoCenter={true}
      />,
    );
    expect(mocks.mockMap.flyTo).toHaveBeenCalledTimes(1);
    expect(mocks.mockMap.flyTo).toHaveBeenCalledWith(
      expect.objectContaining({
        center: [-0.1, 51.5],
      }),
    );
  });

  it("does not fly when autoCenter=false", () => {
    const position = { lat: 51.5, lng: -0.1, heading: null };
    render(
      <SelfMarker
        map={mocks.mockMap as any}
        position={position}
        autoCenter={false}
      />,
    );
    expect(mocks.mockMap.flyTo).not.toHaveBeenCalled();
  });

  it("removes marker on unmount", () => {
    const position = { lat: 51.5, lng: -0.1, heading: null };
    const { unmount } = render(
      <SelfMarker map={mocks.mockMap as any} position={position} />,
    );
    const markerInstance = mocks.Marker.mock.results[0].value;

    unmount();

    expect(markerInstance.remove).toHaveBeenCalled();
  });
});

import { render } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";
import type { MessageResponse } from "@/types/api";

const mocks = vi.hoisted(() => {
  const bounds = { extend: vi.fn() };
  return {
    mockMap: {
      addSource: vi.fn(),
      addLayer: vi.fn(),
      removeSource: vi.fn(),
      removeLayer: vi.fn(),
      getSource: vi.fn(() => null),
      getLayer: vi.fn(() => null),
      fitBounds: vi.fn(),
    },
    LngLatBounds: vi.fn(function () { return bounds; }),
    bounds,
  };
});

vi.mock("maplibre-gl", () => ({
  default: { LngLatBounds: mocks.LngLatBounds },
}));

import { GpxOverlay } from "./gpx-overlay";

function makeMessage(metadata: unknown = null): MessageResponse {
  return {
    id: "m1",
    sender_id: "u1",
    username: "alice",
    display_name: "Alice",
    content: "GPX file",
    message_type: "gpx",
    attachments: [],
    created_at: "2025-01-01T00:00:00Z",
    metadata: metadata as any,
  };
}

const sampleGeoJSON = {
  type: "FeatureCollection" as const,
  features: [
    {
      type: "Feature" as const,
      geometry: {
        type: "LineString" as const,
        coordinates: [
          [-0.1, 51.5],
          [-0.2, 51.6],
        ],
      },
      properties: {},
    },
    {
      type: "Feature" as const,
      geometry: {
        type: "Point" as const,
        coordinates: [-0.15, 51.55],
      },
      properties: {},
    },
  ],
};

describe("GpxOverlay", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns null (renders nothing to the DOM)", () => {
    const { container } = render(
      <GpxOverlay map={mocks.mockMap as any} message={null} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("does not add layers when message is null", () => {
    render(<GpxOverlay map={mocks.mockMap as any} message={null} />);
    expect(mocks.mockMap.addSource).not.toHaveBeenCalled();
    expect(mocks.mockMap.addLayer).not.toHaveBeenCalled();
  });

  it("adds source + 2 layers when message has metadata", () => {
    const message = makeMessage(sampleGeoJSON);

    render(<GpxOverlay map={mocks.mockMap as any} message={message} />);

    expect(mocks.mockMap.addSource).toHaveBeenCalledTimes(1);
    expect(mocks.mockMap.addSource).toHaveBeenCalledWith(
      "gpx-geojson",
      expect.objectContaining({ type: "geojson", data: sampleGeoJSON }),
    );

    // Line layer + point layer
    expect(mocks.mockMap.addLayer).toHaveBeenCalledTimes(2);
    expect(mocks.mockMap.addLayer).toHaveBeenCalledWith(
      expect.objectContaining({ id: "gpx-lines", type: "line" }),
    );
    expect(mocks.mockMap.addLayer).toHaveBeenCalledWith(
      expect.objectContaining({ id: "gpx-points", type: "circle" }),
    );
  });

  it("calls fitBounds with the computed bounds", () => {
    const message = makeMessage(sampleGeoJSON);

    render(<GpxOverlay map={mocks.mockMap as any} message={message} />);

    // LngLatBounds should have been created and extended with coordinates
    expect(mocks.LngLatBounds).toHaveBeenCalledTimes(1);

    // extend called for: 2 points from LineString + 1 Point feature = 3 calls
    expect(mocks.bounds.extend).toHaveBeenCalledTimes(3);

    expect(mocks.mockMap.fitBounds).toHaveBeenCalledTimes(1);
    expect(mocks.mockMap.fitBounds).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ padding: 60, maxZoom: 16 }),
    );
  });

  it("cleans up layers on unmount", () => {
    const message = makeMessage(sampleGeoJSON);

    // Make getLayer/getSource return truthy so cleanup removal fires
    mocks.mockMap.getLayer.mockReturnValue({});
    mocks.mockMap.getSource.mockReturnValue({});

    const { unmount } = render(
      <GpxOverlay map={mocks.mockMap as any} message={message} />,
    );

    unmount();

    expect(mocks.mockMap.removeLayer).toHaveBeenCalledWith("gpx-lines");
    expect(mocks.mockMap.removeLayer).toHaveBeenCalledWith("gpx-points");
    expect(mocks.mockMap.removeSource).toHaveBeenCalledWith("gpx-geojson");
  });
});

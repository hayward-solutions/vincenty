import { render } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";
import type { LocationHistoryEntry } from "@/types/api";

vi.mock("maplibre-gl", () => ({ default: {} }));

import { HistoryTracks } from "./history-tracks";

function createMockMap() {
  return {
    addSource: vi.fn(),
    addLayer: vi.fn(),
    removeSource: vi.fn(),
    removeLayer: vi.fn(),
    getSource: vi.fn(() => null),
    getLayer: vi.fn(() => null),
  };
}

function makeEntry(overrides: Partial<LocationHistoryEntry> = {}): LocationHistoryEntry {
  return {
    user_id: "u1",
    device_id: "d1",
    device_name: "Phone",
    username: "alice",
    display_name: "Alice",
    lat: 51.5,
    lng: -0.1,
    recorded_at: "2025-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("HistoryTracks", () => {
  let mockMap: ReturnType<typeof createMockMap>;

  beforeEach(() => {
    vi.clearAllMocks();
    mockMap = createMockMap();
  });

  it("returns null (renders nothing to the DOM)", () => {
    const { container } = render(
      <HistoryTracks map={mockMap as any} history={[]} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("adds source and layer for each track with >= 2 points", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.5, lng: -0.1, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.6, lng: -0.2, recorded_at: "2025-01-01T00:01:00Z" }),
      makeEntry({ user_id: "u2", device_id: "d2", lat: 40.7, lng: -74.0, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ user_id: "u2", device_id: "d2", lat: 40.8, lng: -74.1, recorded_at: "2025-01-01T00:01:00Z" }),
    ];

    render(<HistoryTracks map={mockMap as any} history={history} />);

    // Two tracks (u1:d1 and u2:d2), each with 2 points
    expect(mockMap.addSource).toHaveBeenCalledTimes(2);
    expect(mockMap.addLayer).toHaveBeenCalledTimes(2);
  });

  it("skips tracks with fewer than 2 points", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.5, lng: -0.1 }),
    ];

    render(<HistoryTracks map={mockMap as any} history={history} />);

    expect(mockMap.addSource).not.toHaveBeenCalled();
    expect(mockMap.addLayer).not.toHaveBeenCalled();
  });

  it("filters entries by playbackTime when set", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.5, lng: -0.1, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.6, lng: -0.2, recorded_at: "2025-01-01T00:01:00Z" }),
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.7, lng: -0.3, recorded_at: "2025-01-01T00:02:00Z" }),
    ];

    // playbackTime cuts off the last entry — only 2 remain (still >= 2, so track is created)
    const playbackTime = new Date("2025-01-01T00:01:30Z");
    render(
      <HistoryTracks map={mockMap as any} history={history} playbackTime={playbackTime} />,
    );

    expect(mockMap.addSource).toHaveBeenCalledTimes(1);
    // Verify the source data only has 2 coordinates
    const sourceData = mockMap.addSource.mock.calls[0][1].data;
    expect(sourceData.geometry.coordinates).toHaveLength(2);
  });

  it("cleans up layers and sources on unmount", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.5, lng: -0.1, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.6, lng: -0.2, recorded_at: "2025-01-01T00:01:00Z" }),
    ];

    // getLayer/getSource return truthy so removeLayer/removeSource actually fire
    mockMap.getLayer.mockReturnValue({});
    mockMap.getSource.mockReturnValue({});

    const { unmount } = render(
      <HistoryTracks map={mockMap as any} history={history} />,
    );

    unmount();

    expect(mockMap.removeLayer).toHaveBeenCalled();
    expect(mockMap.removeSource).toHaveBeenCalled();
  });
});

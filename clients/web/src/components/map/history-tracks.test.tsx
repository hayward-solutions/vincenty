import { render } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";
import type { LocationHistoryEntry } from "@/types/api";

vi.mock("maplibre-gl", () => ({ default: {} }));

import { HistoryTracks } from "./history-tracks";

function createMockMap() {
  const setData = vi.fn();
  return {
    addSource: vi.fn(),
    addLayer: vi.fn(),
    removeSource: vi.fn(),
    removeLayer: vi.fn(),
    getSource: vi.fn(() => ({ setData })),
    getLayer: vi.fn(),
    _setData: setData,
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

  it("adds no sources or layers for empty history", () => {
    render(<HistoryTracks map={mockMap as any} history={[]} />);
    expect(mockMap.addSource).not.toHaveBeenCalled();
    expect(mockMap.addLayer).not.toHaveBeenCalled();
  });

  it("adds source and layer for each track with >= 2 points, plus head markers", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.5, lng: -0.1, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.6, lng: -0.2, recorded_at: "2025-01-01T00:01:00Z" }),
      makeEntry({ user_id: "u2", device_id: "d2", lat: 40.7, lng: -74.0, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ user_id: "u2", device_id: "d2", lat: 40.8, lng: -74.1, recorded_at: "2025-01-01T00:01:00Z" }),
    ];

    render(<HistoryTracks map={mockMap as any} history={history} />);

    // Two track line sources + one head markers source = 3 sources total
    // Two track line layers + one head markers layer = 3 layers total
    expect(mockMap.addSource).toHaveBeenCalledTimes(3);
    expect(mockMap.addLayer).toHaveBeenCalledTimes(3);

    // The head layer uses a circle type with data-driven color
    const layerCalls = mockMap.addLayer.mock.calls;
    const headLayerCall = layerCalls.find((call) => call[0].id === "track-heads-layer");
    expect(headLayerCall).toBeDefined();
    expect(headLayerCall![0].type).toBe("circle");
  });

  it("shows a head-marker dot for single-point tracks (no line is rendered)", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.5, lng: -0.1 }),
    ];

    render(<HistoryTracks map={mockMap as any} history={history} />);

    // 1 track source (line, empty coords) + 1 head markers source = 2 sources
    // 1 track layer (line) + 1 head markers layer = 2 layers
    expect(mockMap.addSource).toHaveBeenCalledTimes(2);
    expect(mockMap.addLayer).toHaveBeenCalledTimes(2);
  });

  it("calls setData on existing source instead of recreating layers", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ lat: 51.5, lng: -0.1, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ lat: 51.6, lng: -0.2, recorded_at: "2025-01-01T00:01:00Z" }),
      makeEntry({ lat: 51.7, lng: -0.3, recorded_at: "2025-01-01T00:02:00Z" }),
    ];

    const { rerender } = render(
      <HistoryTracks map={mockMap as any} history={history} />,
    );

    const addSourceCallCount = mockMap.addSource.mock.calls.length;
    const addLayerCallCount = mockMap.addLayer.mock.calls.length;

    // Changing playbackTime should NOT add new sources/layers
    rerender(
      <HistoryTracks
        map={mockMap as any}
        history={history}
        playbackTime={new Date("2025-01-01T00:01:30Z")}
      />,
    );

    expect(mockMap.addSource).toHaveBeenCalledTimes(addSourceCallCount);
    expect(mockMap.addLayer).toHaveBeenCalledTimes(addLayerCallCount);

    // setData should have been called for the track source and head source
    expect(mockMap._setData).toHaveBeenCalled();
  });

  it("filters entries by playbackTime when setData is called", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ lat: 51.5, lng: -0.1, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ lat: 51.6, lng: -0.2, recorded_at: "2025-01-01T00:01:00Z" }),
      makeEntry({ lat: 51.7, lng: -0.3, recorded_at: "2025-01-01T00:02:00Z" }),
    ];

    const playbackTime = new Date("2025-01-01T00:01:30Z");
    render(
      <HistoryTracks map={mockMap as any} history={history} playbackTime={playbackTime} />,
    );

    // Find the setData call for the track line source (not the head markers)
    const setDataCalls = mockMap._setData.mock.calls;
    const lineCall = setDataCalls.find(
      (call: unknown[]) => (call[0] as GeoJSON.Feature).geometry?.type === "LineString"
    );
    expect(lineCall).toBeDefined();
    // Only 2 of 3 points are at or before playbackTime
    expect((lineCall![0] as GeoJSON.Feature<GeoJSON.LineString>).geometry.coordinates).toHaveLength(2);
  });

  it("shows all entries when no playbackTime is set", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ lat: 51.5, lng: -0.1, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ lat: 51.6, lng: -0.2, recorded_at: "2025-01-01T00:01:00Z" }),
      makeEntry({ lat: 51.7, lng: -0.3, recorded_at: "2025-01-01T00:02:00Z" }),
    ];

    render(<HistoryTracks map={mockMap as any} history={history} />);

    const setDataCalls = mockMap._setData.mock.calls;
    const lineCall = setDataCalls.find(
      (call: unknown[]) => (call[0] as GeoJSON.Feature).geometry?.type === "LineString"
    );
    expect(lineCall).toBeDefined();
    expect((lineCall![0] as GeoJSON.Feature<GeoJSON.LineString>).geometry.coordinates).toHaveLength(3);
  });

  it("cleans up layers and sources on unmount", () => {
    const history: LocationHistoryEntry[] = [
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.5, lng: -0.1, recorded_at: "2025-01-01T00:00:00Z" }),
      makeEntry({ user_id: "u1", device_id: "d1", lat: 51.6, lng: -0.2, recorded_at: "2025-01-01T00:01:00Z" }),
    ];

    // getLayer returns truthy so removeLayer/removeSource actually fire
    mockMap.getLayer.mockReturnValue({});

    const { unmount } = render(
      <HistoryTracks map={mockMap as any} history={history} />,
    );

    unmount();

    expect(mockMap.removeLayer).toHaveBeenCalled();
    expect(mockMap.removeSource).toHaveBeenCalled();
  });
});

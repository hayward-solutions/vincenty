import { render } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";
import type { DrawingResponse } from "@/types/api";

vi.mock("maplibre-gl", () => ({ default: {} }));

import { DrawingOverlay } from "./drawing-overlay";

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

const mockDrawing: DrawingResponse = {
  id: "d1",
  owner_id: "u1",
  username: "user",
  display_name: "User",
  name: "Test Drawing",
  geojson: { type: "FeatureCollection", features: [] },
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

function makeDrawing(overrides: Partial<DrawingResponse> = {}): DrawingResponse {
  return { ...mockDrawing, ...overrides };
}

describe("DrawingOverlay", () => {
  let mockMap: ReturnType<typeof createMockMap>;

  beforeEach(() => {
    vi.clearAllMocks();
    mockMap = createMockMap();
  });

  it("returns null (renders nothing to the DOM)", () => {
    const { container } = render(
      <DrawingOverlay map={mockMap as any} drawings={[]} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("adds source + 4 layers per drawing", () => {
    const drawings = [
      makeDrawing({ id: "d1" }),
      makeDrawing({ id: "d2" }),
    ];

    render(<DrawingOverlay map={mockMap as any} drawings={drawings} />);

    // 1 source per drawing
    expect(mockMap.addSource).toHaveBeenCalledTimes(2);
    // 4 layers per drawing (fill, outline, line, point)
    expect(mockMap.addLayer).toHaveBeenCalledTimes(8);

    // Verify source IDs
    expect(mockMap.addSource).toHaveBeenCalledWith("drawing-d1", expect.any(Object));
    expect(mockMap.addSource).toHaveBeenCalledWith("drawing-d2", expect.any(Object));
  });

  it("removes layers when a drawing is removed from the list", () => {
    const d1 = makeDrawing({ id: "d1" });
    const d2 = makeDrawing({ id: "d2" });

    // First render with both drawings
    const { rerender } = render(
      <DrawingOverlay map={mockMap as any} drawings={[d1, d2]} />,
    );

    // Now make getLayer/getSource return truthy so removal works
    mockMap.getLayer.mockReturnValue({});
    mockMap.getSource.mockReturnValue({});

    // Rerender with only d1 — d2 should be removed
    rerender(<DrawingOverlay map={mockMap as any} drawings={[d1]} />);

    // Should remove 4 layers + 1 source for d2
    expect(mockMap.removeLayer).toHaveBeenCalledWith("drawing-d2-fill");
    expect(mockMap.removeLayer).toHaveBeenCalledWith("drawing-d2-outline");
    expect(mockMap.removeLayer).toHaveBeenCalledWith("drawing-d2-line");
    expect(mockMap.removeLayer).toHaveBeenCalledWith("drawing-d2-point");
    expect(mockMap.removeSource).toHaveBeenCalledWith("drawing-d2");
  });

  it("updates source data when drawing's updated_at changes", () => {
    const d1 = makeDrawing({ id: "d1", updated_at: "2025-01-01T00:00:00Z" });

    const mockSource = { setData: vi.fn() };
    mockMap.getSource.mockReturnValue(mockSource);

    const { rerender } = render(
      <DrawingOverlay map={mockMap as any} drawings={[d1]} />,
    );

    const initialAddSourceCalls = mockMap.addSource.mock.calls.length;

    // Rerender with updated_at changed
    const updatedD1 = makeDrawing({
      id: "d1",
      updated_at: "2025-01-02T00:00:00Z",
      geojson: {
        type: "FeatureCollection",
        features: [{ type: "Feature", geometry: { type: "Point", coordinates: [0, 0] }, properties: {} }],
      },
    });

    rerender(<DrawingOverlay map={mockMap as any} drawings={[updatedD1]} />);

    // Should NOT have added a new source (it already exists)
    expect(mockMap.addSource.mock.calls.length).toBe(initialAddSourceCalls);

    // Should have called setData on the existing source
    expect(mockSource.setData).toHaveBeenCalledWith(updatedD1.geojson);
  });

  it("cleans up all drawings on unmount", () => {
    const drawings = [
      makeDrawing({ id: "d1" }),
      makeDrawing({ id: "d2" }),
    ];

    mockMap.getLayer.mockReturnValue({});
    mockMap.getSource.mockReturnValue({});

    const { unmount } = render(
      <DrawingOverlay map={mockMap as any} drawings={drawings} />,
    );

    unmount();

    // Each drawing has 4 layers + 1 source = 8 removeLayer + 2 removeSource
    expect(mockMap.removeLayer).toHaveBeenCalledTimes(8);
    expect(mockMap.removeSource).toHaveBeenCalledTimes(2);
  });
});

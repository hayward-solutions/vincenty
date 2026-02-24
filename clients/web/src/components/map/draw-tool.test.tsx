import { render } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";
import { DrawTool } from "./draw-tool";
import type { DrawStyle, CompletedShape } from "./draw-tool";

// ---------------------------------------------------------------------------
// Mock map
// ---------------------------------------------------------------------------

const createMockMap = () => {
  const canvas = { style: { cursor: "" } };
  const source = { setData: vi.fn() };
  return {
    getCanvas: vi.fn(() => canvas),
    addSource: vi.fn(),
    addLayer: vi.fn(),
    removeSource: vi.fn(),
    removeLayer: vi.fn(),
    getSource: vi.fn((id: string) =>
      id === "draw-tool-geojson" ? source : null
    ),
    getLayer: vi.fn(() => null),
    on: vi.fn(),
    off: vi.fn(),
    canvas,
    source,
  };
};

// ---------------------------------------------------------------------------
// Default props
// ---------------------------------------------------------------------------

const defaultProps = () => ({
  active: true,
  mode: "line" as const,
  style: {
    stroke: "#ff0000",
    fill: "#ff000040",
    strokeWidth: 2,
  } as DrawStyle,
  resetKey: 0,
  completedFeatures: [] as GeoJSON.Feature[],
  onShapeComplete: vi.fn(),
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("DrawTool", () => {
  let mockMap: ReturnType<typeof createMockMap>;

  beforeEach(() => {
    mockMap = createMockMap();
  });

  it("returns null (renderless component)", () => {
    const props = defaultProps();
    const { container } = render(
      <DrawTool map={mockMap as any} {...props} />
    );
    expect(container.innerHTML).toBe("");
  });

  it("sets cursor to 'crosshair' when active", () => {
    const props = defaultProps();
    render(<DrawTool map={mockMap as any} {...props} />);
    expect(mockMap.canvas.style.cursor).toBe("crosshair");
  });

  it("adds source and 5 layers when active", () => {
    const props = defaultProps();
    render(<DrawTool map={mockMap as any} {...props} />);

    expect(mockMap.addSource).toHaveBeenCalledWith("draw-tool-geojson", {
      type: "geojson",
      data: { type: "FeatureCollection", features: [] },
    });
    expect(mockMap.addLayer).toHaveBeenCalledTimes(5);

    // Verify layer IDs
    const layerIds = mockMap.addLayer.mock.calls.map(
      (call: any[]) => call[0].id
    );
    expect(layerIds).toEqual([
      "draw-tool-fill",
      "draw-tool-outline",
      "draw-tool-line",
      "draw-tool-pending",
      "draw-tool-points",
    ]);
  });

  it("registers click, mousemove, dblclick event listeners", () => {
    const props = defaultProps();
    render(<DrawTool map={mockMap as any} {...props} />);

    const registeredEvents = mockMap.on.mock.calls.map(
      (call: any[]) => call[0]
    );
    expect(registeredEvents).toContain("click");
    expect(registeredEvents).toContain("mousemove");
    expect(registeredEvents).toContain("dblclick");
  });

  it("removes event listeners and cleans up on deactivation", () => {
    const props = defaultProps();
    const { unmount } = render(
      <DrawTool map={mockMap as any} {...props} />
    );

    unmount();

    const removedEvents = mockMap.off.mock.calls.map(
      (call: any[]) => call[0]
    );
    expect(removedEvents).toContain("click");
    expect(removedEvents).toContain("mousemove");
    expect(removedEvents).toContain("dblclick");
  });

  it("resets cursor to '' on cleanup", () => {
    const props = defaultProps();
    const { unmount } = render(
      <DrawTool map={mockMap as any} {...props} />
    );

    expect(mockMap.canvas.style.cursor).toBe("crosshair");
    unmount();
    expect(mockMap.canvas.style.cursor).toBe("");
  });

  it("does not add layers when not active", () => {
    const props = defaultProps();
    render(
      <DrawTool map={mockMap as any} {...props} active={false} />
    );

    expect(mockMap.addSource).not.toHaveBeenCalled();
    expect(mockMap.addLayer).not.toHaveBeenCalled();
    expect(mockMap.on).not.toHaveBeenCalled();
    expect(mockMap.canvas.style.cursor).toBe("");
  });

  it("calls onShapeComplete when a circle drawing is finalized via two clicks", () => {
    const props = defaultProps();
    props.mode = "circle";

    render(<DrawTool map={mockMap as any} {...props} />);

    const clickHandler = mockMap.on.mock.calls.find(
      ([e]: any[]) => e === "click"
    )?.[1];
    expect(clickHandler).toBeDefined();

    // First click — set center
    clickHandler({
      lngLat: { lng: 10, lat: 20 },
    });

    // Second click — set edge, triggers completion
    clickHandler({
      lngLat: { lng: 10.01, lat: 20 },
    });

    expect(props.onShapeComplete).toHaveBeenCalledTimes(1);
    expect(props.onShapeComplete).toHaveBeenCalledWith(
      expect.objectContaining({
        feature: expect.objectContaining({
          type: "Feature",
          properties: expect.objectContaining({ shapeType: "circle" }),
          geometry: expect.objectContaining({ type: "Polygon" }),
        }),
      })
    );
  });
});

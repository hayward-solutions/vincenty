import { render } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";
import { MeasureTool, formatDistance, formatArea } from "./measure-tool";

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
      id === "measure-geojson" ? source : null
    ),
    getLayer: vi.fn(() => null),
    on: vi.fn(),
    off: vi.fn(),
    canvas,
    source,
  };
};

// ---------------------------------------------------------------------------
// Pure function tests
// ---------------------------------------------------------------------------

describe("formatDistance", () => {
  it('returns "500m" for 500', () => {
    expect(formatDistance(500)).toBe("500m");
  });

  it('returns "1.50km" for 1500', () => {
    expect(formatDistance(1500)).toBe("1.50km");
  });
});

describe("formatArea", () => {
  it("returns formatted m² for 5000", () => {
    expect(formatArea(5000)).toBe(`${(5000).toLocaleString()}m²`);
  });

  it('returns "1.50km²" for 1500000', () => {
    expect(formatArea(1_500_000)).toBe("1.50km²");
  });
});

// ---------------------------------------------------------------------------
// Component tests
// ---------------------------------------------------------------------------

describe("MeasureTool", () => {
  let mockMap: ReturnType<typeof createMockMap>;

  const defaultProps = () => ({
    active: true,
    mode: "line" as const,
    resetKey: 0,
    onMeasurementsChange: vi.fn(),
  });

  beforeEach(() => {
    mockMap = createMockMap();
  });

  it("returns null (renderless component)", () => {
    const props = defaultProps();
    const { container } = render(
      <MeasureTool map={mockMap as any} {...props} />
    );
    expect(container.innerHTML).toBe("");
  });

  it("sets cursor to 'crosshair' when active", () => {
    const props = defaultProps();
    render(<MeasureTool map={mockMap as any} {...props} />);
    expect(mockMap.canvas.style.cursor).toBe("crosshair");
  });

  it("adds source and 4 layers when active in line mode", () => {
    const props = defaultProps();
    render(<MeasureTool map={mockMap as any} {...props} />);

    expect(mockMap.addSource).toHaveBeenCalledWith("measure-geojson", {
      type: "geojson",
      data: { type: "FeatureCollection", features: [] },
    });
    expect(mockMap.addLayer).toHaveBeenCalledTimes(4);

    const layerIds = mockMap.addLayer.mock.calls.map(
      (call: any[]) => call[0].id
    );
    expect(layerIds).toEqual([
      "measure-lines",
      "measure-pending",
      "measure-points",
      "measure-labels",
    ]);
  });

  it("adds 5 layers when active in circle mode", () => {
    const props = defaultProps();
    props.mode = "circle";
    render(<MeasureTool map={mockMap as any} {...props} />);

    expect(mockMap.addLayer).toHaveBeenCalledTimes(5);

    const layerIds = mockMap.addLayer.mock.calls.map(
      (call: any[]) => call[0].id
    );
    expect(layerIds).toEqual([
      "measure-fill",
      "measure-outline",
      "measure-radius",
      "measure-center",
      "measure-labels",
    ]);
  });

  it("registers click, mousemove, dblclick listeners", () => {
    const props = defaultProps();
    render(<MeasureTool map={mockMap as any} {...props} />);

    const registeredEvents = mockMap.on.mock.calls.map(
      (call: any[]) => call[0]
    );
    expect(registeredEvents).toContain("click");
    expect(registeredEvents).toContain("mousemove");
    expect(registeredEvents).toContain("dblclick");
  });

  it("calls onMeasurementsChange with initial empty result on activation", () => {
    const props = defaultProps();
    render(<MeasureTool map={mockMap as any} {...props} />);

    expect(props.onMeasurementsChange).toHaveBeenCalledWith({
      segments: [],
      total: 0,
      radius: undefined,
      area: undefined,
    });
  });

  it("cleans up on deactivation", () => {
    const props = defaultProps();
    const { unmount } = render(
      <MeasureTool map={mockMap as any} {...props} />
    );

    expect(mockMap.canvas.style.cursor).toBe("crosshair");

    unmount();

    const removedEvents = mockMap.off.mock.calls.map(
      (call: any[]) => call[0]
    );
    expect(removedEvents).toContain("click");
    expect(removedEvents).toContain("mousemove");
    expect(removedEvents).toContain("dblclick");
    expect(mockMap.canvas.style.cursor).toBe("");
  });

  it("does not add layers when not active", () => {
    const props = defaultProps();
    render(
      <MeasureTool map={mockMap as any} {...props} active={false} />
    );

    expect(mockMap.addSource).not.toHaveBeenCalled();
    expect(mockMap.addLayer).not.toHaveBeenCalled();
    expect(mockMap.on).not.toHaveBeenCalled();
    expect(mockMap.canvas.style.cursor).toBe("");
  });
});

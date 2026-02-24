import { render, screen, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MapControls } from "./map-controls";

// Radix UI Tooltip uses ResizeObserver which jsdom doesn't provide
beforeAll(() => {
  global.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
});

const createMockMap = () => ({
  getBearing: vi.fn(() => 0),
  getPitch: vi.fn(() => 0),
  getProjection: vi.fn(() => ({ type: "globe" })),
  getZoom: vi.fn(() => 10),
  zoomIn: vi.fn(),
  zoomOut: vi.fn(),
  easeTo: vi.fn(),
  flyTo: vi.fn(),
  setTerrain: vi.fn(),
  setProjection: vi.fn(),
  on: vi.fn(),
  off: vi.fn(),
});

type MockMap = ReturnType<typeof createMockMap>;

function renderControls(
  overrides: { map?: MockMap; terrainAvailable?: boolean; position?: { lat: number; lng: number; heading: number | null } | null } = {}
) {
  const map = overrides.map ?? createMockMap();
  const props = {
    map: map as unknown as Parameters<typeof MapControls>[0]["map"],
    terrainAvailable: overrides.terrainAvailable,
    position: overrides.position ?? null,
  };
  const result = render(<MapControls {...props} />);
  return { ...result, map, props };
}

describe("MapControls", () => {
  describe("rendering", () => {
    it("renders Zoom in button", () => {
      renderControls();
      expect(screen.getByRole("button", { name: "Zoom in" })).toBeInTheDocument();
    });

    it("renders Zoom out button", () => {
      renderControls();
      expect(screen.getByRole("button", { name: "Zoom out" })).toBeInTheDocument();
    });

    it("renders Reset north button", () => {
      renderControls();
      expect(screen.getByRole("button", { name: "Reset north" })).toBeInTheDocument();
    });

    it("renders Toggle globe button", () => {
      renderControls();
      expect(screen.getByRole("button", { name: "Toggle globe" })).toBeInTheDocument();
    });
  });

  describe("click handlers", () => {
    it("clicking Zoom in calls map.zoomIn()", async () => {
      const user = userEvent.setup();
      const { map } = renderControls();
      await user.click(screen.getByRole("button", { name: "Zoom in" }));
      expect(map.zoomIn).toHaveBeenCalledTimes(1);
    });

    it("clicking Zoom out calls map.zoomOut()", async () => {
      const user = userEvent.setup();
      const { map } = renderControls();
      await user.click(screen.getByRole("button", { name: "Zoom out" }));
      expect(map.zoomOut).toHaveBeenCalledTimes(1);
    });

    it("clicking Reset north calls map.easeTo with bearing 0 and pitch 0", async () => {
      const user = userEvent.setup();
      const { map } = renderControls();
      await user.click(screen.getByRole("button", { name: "Reset north" }));
      expect(map.easeTo).toHaveBeenCalledWith({ bearing: 0, pitch: 0 });
    });
  });

  describe("terrain button", () => {
    it("is not shown when terrainAvailable is false", () => {
      renderControls({ terrainAvailable: false });
      expect(screen.queryByRole("button", { name: "Toggle terrain" })).not.toBeInTheDocument();
    });

    it("is not shown when terrainAvailable is undefined", () => {
      renderControls();
      expect(screen.queryByRole("button", { name: "Toggle terrain" })).not.toBeInTheDocument();
    });

    it("is shown when terrainAvailable is true", () => {
      renderControls({ terrainAvailable: true });
      expect(screen.getByRole("button", { name: "Toggle terrain" })).toBeInTheDocument();
    });
  });

  describe("track button", () => {
    it("is disabled when no position prop", () => {
      renderControls({ position: null });
      expect(screen.getByRole("button", { name: "Track my location" })).toBeDisabled();
    });

    it("is enabled when position is provided", () => {
      renderControls({ position: { lat: 40, lng: -74, heading: null } });
      expect(screen.getByRole("button", { name: "Track my location" })).toBeEnabled();
    });
  });

  describe("map event listeners", () => {
    it("registers dragstart, rotate, and pitch event listeners", () => {
      const map = createMockMap();
      renderControls({ map });

      const registeredEvents = map.on.mock.calls.map(
        (call: [string, ...unknown[]]) => call[0]
      );
      expect(registeredEvents).toContain("dragstart");
      expect(registeredEvents).toContain("rotate");
      expect(registeredEvents).toContain("pitch");
    });
  });
});

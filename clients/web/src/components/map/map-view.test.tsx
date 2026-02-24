import { render, screen, act } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach } from "vitest";
import type { MapSettings } from "@/types/api";

const mocks = vi.hoisted(() => {
  const mapInstance = {
    on: vi.fn(),
    off: vi.fn(),
    remove: vi.fn(),
    addSource: vi.fn(),
  };
  return {
    mapInstance,
    MapConstructor: vi.fn(function () {
      return mapInstance;
    }),
  };
});

vi.mock("maplibre-gl", () => ({
  default: {
    Map: mocks.MapConstructor,
  },
}));

import { MapView } from "./map-view";

const defaultSettings: MapSettings = {
  tile_url: "https://tile.osm.org/{z}/{x}/{y}.png",
  center_lat: 0,
  center_lng: 0,
  zoom: 2,
  min_zoom: 0,
  max_zoom: 19,
  terrain_url: "",
  terrain_encoding: "terrarium",
  configs: [],
};

function triggerLoadEvent() {
  const loadCall = mocks.mapInstance.on.mock.calls.find(
    ([event]: [string]) => event === "load",
  );
  expect(loadCall).toBeDefined();
  const loadHandler = loadCall![1] as () => void;
  act(() => {
    loadHandler();
  });
}

describe("MapView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders a container div", () => {
    const { container } = render(<MapView settings={defaultSettings} />);
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper).toBeInstanceOf(HTMLDivElement);
    expect(wrapper.querySelector("div")).toBeTruthy();
  });

  it("creates a maplibregl.Map with correct options", () => {
    render(<MapView settings={defaultSettings} />);

    expect(mocks.MapConstructor).toHaveBeenCalledTimes(1);
    const opts = mocks.MapConstructor.mock.calls[0][0];
    expect(opts.container).toBeInstanceOf(HTMLDivElement);
    expect(opts.center).toEqual([0, 0]);
    expect(opts.zoom).toBe(2);
    expect(opts.minZoom).toBe(0);
    expect(opts.maxZoom).toBe(19);
  });

  it("calls onMapReady with map instance after load event fires", () => {
    const onMapReady = vi.fn();
    render(<MapView settings={defaultSettings} onMapReady={onMapReady} />);

    expect(onMapReady).not.toHaveBeenCalled();

    triggerLoadEvent();

    expect(onMapReady).toHaveBeenCalledTimes(1);
    expect(onMapReady).toHaveBeenCalledWith(mocks.mapInstance);
  });

  it("does not render children before load event", () => {
    render(
      <MapView settings={defaultSettings}>
        <span data-testid="child">Hello</span>
      </MapView>,
    );

    expect(screen.queryByTestId("child")).not.toBeInTheDocument();
  });

  it("renders children after load event fires", () => {
    render(
      <MapView settings={defaultSettings}>
        <span data-testid="child">Hello</span>
      </MapView>,
    );

    triggerLoadEvent();

    expect(screen.getByTestId("child")).toBeInTheDocument();
    expect(screen.getByTestId("child")).toHaveTextContent("Hello");
  });

  it("calls map.remove() on unmount", () => {
    const { unmount } = render(<MapView settings={defaultSettings} />);

    triggerLoadEvent();

    expect(mocks.mapInstance.remove).not.toHaveBeenCalled();

    unmount();

    expect(mocks.mapInstance.remove).toHaveBeenCalledTimes(1);
  });

  it("adds terrain-dem source when terrain_url is provided", () => {
    const settingsWithTerrain: MapSettings = {
      ...defaultSettings,
      terrain_url: "https://example.com/terrain/{z}/{x}/{y}.png",
      terrain_encoding: "terrarium",
    };

    render(<MapView settings={settingsWithTerrain} />);

    triggerLoadEvent();

    expect(mocks.mapInstance.addSource).toHaveBeenCalledTimes(1);
    expect(mocks.mapInstance.addSource).toHaveBeenCalledWith("terrain-dem", {
      type: "raster-dem",
      tiles: ["https://example.com/terrain/{z}/{x}/{y}.png"],
      encoding: "terrarium",
      tileSize: 256,
    });
  });

  it("does not add terrain-dem source when terrain_url is empty", () => {
    render(<MapView settings={defaultSettings} />);

    triggerLoadEvent();

    expect(mocks.mapInstance.addSource).not.toHaveBeenCalled();
  });
});

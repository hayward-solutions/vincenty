import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useMapSettings,
  useMapConfigs,
  useCreateMapConfig,
  useUpdateMapConfig,
  useDeleteMapConfig,
  useTerrainConfigs,
  useCreateTerrainConfig,
  useUpdateTerrainConfig,
  useDeleteTerrainConfig,
} from "@/lib/hooks/use-map-settings";

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

describe("useMapSettings", () => {
  it("fetches map settings on mount", async () => {
    const { result } = renderHook(() => useMapSettings());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.settings).toMatchObject({
      center_lat: 0,
      center_lng: 0,
    });
    expect(result.current.settings?.configs).toHaveLength(1);
    expect(result.current.error).toBeNull();
  });

  it("handles error", async () => {
    server.use(
      http.get("/api/v1/map/settings", () => {
        return HttpResponse.json({ error: { message: "unauthorized" } }, { status: 401 });
      })
    );

    const { result } = renderHook(() => useMapSettings());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.error).toBe("unauthorized");
  });
});

describe("useMapConfigs", () => {
  it("fetches map configs on mount", async () => {
    const { result } = renderHook(() => useMapConfigs());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.configs).toHaveLength(1);
    expect(result.current.configs[0].name).toBe("OpenStreetMap");
    expect(result.current.error).toBeNull();
  });
});

describe("useCreateMapConfig", () => {
  it("creates a map config", async () => {
    const { result } = renderHook(() => useCreateMapConfig());

    let config;
    await act(async () => {
      config = await result.current.createMapConfig({
        name: "Custom Map",
        tile_url: "https://example.com/{z}/{x}/{y}.png",
      });
    });

    expect(config).toMatchObject({ name: "OpenStreetMap" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUpdateMapConfig", () => {
  it("updates a map config", async () => {
    const { result } = renderHook(() => useUpdateMapConfig());

    let config;
    await act(async () => {
      config = await result.current.updateMapConfig("mapconfig-1", { name: "Updated Map" });
    });

    expect(config).toMatchObject({ name: "Updated Map" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteMapConfig", () => {
  it("deletes a map config", async () => {
    const { result } = renderHook(() => useDeleteMapConfig());

    await act(async () => {
      await result.current.deleteMapConfig("mapconfig-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

describe("useTerrainConfigs", () => {
  it("fetches terrain configs on mount", async () => {
    const { result } = renderHook(() => useTerrainConfigs());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.configs).toHaveLength(1);
    expect(result.current.configs[0].name).toBe("Default Terrain");
    expect(result.current.error).toBeNull();
  });
});

describe("useCreateTerrainConfig", () => {
  it("creates a terrain config", async () => {
    const { result } = renderHook(() => useCreateTerrainConfig());

    let config;
    await act(async () => {
      config = await result.current.createTerrainConfig({
        name: "Custom Terrain",
        terrain_url: "https://example.com/terrain/{z}/{x}/{y}.png",
      });
    });

    expect(config).toBeDefined();
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUpdateTerrainConfig", () => {
  it("updates a terrain config", async () => {
    const { result } = renderHook(() => useUpdateTerrainConfig());

    let config;
    await act(async () => {
      config = await result.current.updateTerrainConfig("terrain-1", { name: "Updated Terrain" });
    });

    expect(config).toMatchObject({ name: "Updated Terrain" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteTerrainConfig", () => {
  it("deletes a terrain config", async () => {
    const { result } = renderHook(() => useDeleteTerrainConfig());

    await act(async () => {
      await result.current.deleteTerrainConfig("terrain-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

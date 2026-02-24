import { describe, it, expect } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useMyDevices,
  useResolveDevice,
  useClaimDevice,
  useCreateDevice,
  useUpdateDevice,
  useSetPrimaryDevice,
  useDeleteDevice,
} from "@/lib/hooks/use-devices";

describe("useMyDevices", () => {
  it("fetches devices when fetch() is called", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useMyDevices());

    // Initial state — not yet fetched
    expect(result.current.devices).toEqual([]);
    expect(result.current.isLoading).toBe(false);

    await act(async () => {
      await result.current.fetch();
    });

    expect(result.current.devices).toHaveLength(1);
    expect(result.current.devices[0].name).toBe("Web Browser");
    expect(result.current.isLoading).toBe(false);
  });

  it("handles error", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.get("/api/v1/users/me/devices", () => {
        return HttpResponse.json({ error: { message: "unauthorized" } }, { status: 401 });
      })
    );

    const { result } = renderHook(() => useMyDevices());

    await act(async () => {
      await result.current.fetch();
    });

    expect(result.current.error).toBe("unauthorized");
  });
});

describe("useResolveDevice", () => {
  it("resolves device via server", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useResolveDevice());

    let response;
    await act(async () => {
      response = await result.current.resolve();
    });

    expect(response).toMatchObject({ matched: true });
    expect(response?.device?.id).toBe("device-1");
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useClaimDevice", () => {
  it("claims an existing device", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useClaimDevice());

    let device;
    await act(async () => {
      device = await result.current.claimDevice("device-1");
    });

    expect(device).toMatchObject({ id: "device-1" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useCreateDevice", () => {
  it("creates a new device", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useCreateDevice());

    let device;
    await act(async () => {
      device = await result.current.createDevice("My Tablet");
    });

    expect(device).toMatchObject({ id: "device-1" });
    expect(result.current.isLoading).toBe(false);
  });

  it("defaults name to 'Web Browser'", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.post("/api/v1/users/me/devices", async ({ request }) => {
        const body = (await request.json()) as { name: string; device_type: string };
        expect(body.name).toBe("Web Browser");
        expect(body.device_type).toBe("web");
        return HttpResponse.json({ id: "device-new", ...body, user_id: "user-1", device_uid: "uid", is_primary: false, created_at: "", updated_at: "" });
      })
    );

    const { result } = renderHook(() => useCreateDevice());

    await act(async () => {
      await result.current.createDevice();
    });
  });
});

describe("useUpdateDevice", () => {
  it("updates a device", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUpdateDevice());

    let device;
    await act(async () => {
      device = await result.current.updateDevice("device-1", { name: "Renamed" });
    });

    expect(device).toMatchObject({ name: "Renamed" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useSetPrimaryDevice", () => {
  it("sets device as primary", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useSetPrimaryDevice());

    let device;
    await act(async () => {
      device = await result.current.setPrimary("device-1");
    });

    expect(device).toMatchObject({ is_primary: true });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteDevice", () => {
  it("deletes a device", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useDeleteDevice());

    await act(async () => {
      await result.current.deleteDevice("device-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

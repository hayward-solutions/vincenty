import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useLocationHistory,
  useMyLocationHistory,
  useVisibleHistory,
  useUserLocationHistory,
  useMyGroups,
  useAllLocations,
} from "@/lib/hooks/use-location-history";

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

describe("useLocationHistory", () => {
  it("fetches location history for a group", async () => {
    const { result } = renderHook(() => useLocationHistory());

    const from = new Date("2025-01-01T00:00:00Z");
    const to = new Date("2025-01-01T23:59:59Z");

    await act(async () => {
      await result.current.fetchHistory("group-1", from, to);
    });

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data[0].lat).toBe(-33.8688);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("handles error", async () => {
    server.use(
      http.get("/api/v1/groups/:groupId/locations/history", () => {
        return HttpResponse.json({ error: { message: "not found" } }, { status: 404 });
      })
    );

    const { result } = renderHook(() => useLocationHistory());

    await act(async () => {
      await result.current.fetchHistory(
        "group-unknown",
        new Date(),
        new Date()
      );
    });

    expect(result.current.error).toBe("not found");
    expect(result.current.data).toEqual([]);
  });

  it("clears data", async () => {
    const { result } = renderHook(() => useLocationHistory());

    await act(async () => {
      await result.current.fetchHistory(
        "group-1",
        new Date("2025-01-01"),
        new Date("2025-01-02")
      );
    });

    expect(result.current.data).toHaveLength(1);

    act(() => {
      result.current.clear();
    });

    expect(result.current.data).toEqual([]);
  });
});

describe("useMyLocationHistory", () => {
  it("fetches own location history", async () => {
    const { result } = renderHook(() => useMyLocationHistory());

    await act(async () => {
      await result.current.fetchHistory(
        new Date("2025-01-01"),
        new Date("2025-01-02")
      );
    });

    expect(result.current.data).toHaveLength(1);
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useVisibleHistory", () => {
  it("fetches visible history", async () => {
    const { result } = renderHook(() => useVisibleHistory());

    await act(async () => {
      await result.current.fetchHistory(
        new Date("2025-01-01"),
        new Date("2025-01-02")
      );
    });

    expect(result.current.data).toHaveLength(1);
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUserLocationHistory", () => {
  it("fetches a specific user's location history", async () => {
    const { result } = renderHook(() => useUserLocationHistory());

    await act(async () => {
      await result.current.fetchHistory(
        "user-1",
        new Date("2025-01-01"),
        new Date("2025-01-02")
      );
    });

    expect(result.current.data).toHaveLength(1);
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useMyGroups", () => {
  it("fetches user groups on mount", async () => {
    const { result } = renderHook(() => useMyGroups());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.groups).toHaveLength(1);
    expect(result.current.groups[0].name).toBe("Test Group");
  });

  it("returns empty array on error", async () => {
    server.use(
      http.get("/api/v1/users/me/groups", () => {
        return HttpResponse.json({ error: { message: "error" } }, { status: 500 });
      })
    );

    const { result } = renderHook(() => useMyGroups());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.groups).toEqual([]);
  });
});

describe("useAllLocations", () => {
  it("fetches all latest locations", async () => {
    const { result } = renderHook(() => useAllLocations());

    await act(async () => {
      await result.current.fetchAll();
    });

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data[0].is_primary).toBe(true);
    expect(result.current.isLoading).toBe(false);
  });

  it("handles error", async () => {
    server.use(
      http.get("/api/v1/locations", () => {
        return HttpResponse.json({ error: { message: "forbidden" } }, { status: 403 });
      })
    );

    const { result } = renderHook(() => useAllLocations());

    await act(async () => {
      await result.current.fetchAll();
    });

    expect(result.current.error).toBe("forbidden");
  });
});

import { describe, it, expect } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useOwnDrawings,
  useSharedDrawings,
  useCreateDrawing,
  useUpdateDrawing,
  useDeleteDrawing,
  useShareDrawing,
  useDrawingShares,
  useUnshareDrawing,
} from "@/lib/hooks/use-drawings";

describe("useOwnDrawings", () => {
  it("fetches own drawings on mount", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useOwnDrawings());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.drawings).toHaveLength(1);
    expect(result.current.drawings[0].name).toBe("Test Drawing");
    expect(result.current.error).toBeNull();
  });

  it("handles error", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.get("/api/v1/drawings", () => {
        return HttpResponse.json({ error: { message: "server error" } }, { status: 500 });
      })
    );

    const { result } = renderHook(() => useOwnDrawings());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.error).toBe("server error");
  });
});

describe("useSharedDrawings", () => {
  it("fetches shared drawings on mount", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useSharedDrawings());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.drawings).toHaveLength(1);
    expect(result.current.drawings[0].owner_id).toBe("user-2");
  });
});

describe("useCreateDrawing", () => {
  it("creates a drawing", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useCreateDrawing());

    let drawing;
    await act(async () => {
      drawing = await result.current.createDrawing({
        name: "New Drawing",
        geojson: { type: "FeatureCollection", features: [] },
      });
    });

    expect(drawing).toMatchObject({ name: "Test Drawing" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUpdateDrawing", () => {
  it("updates a drawing", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUpdateDrawing());

    let drawing;
    await act(async () => {
      drawing = await result.current.updateDrawing("drawing-1", { name: "Updated" });
    });

    expect(drawing).toMatchObject({ name: "Updated Drawing" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteDrawing", () => {
  it("deletes a drawing", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useDeleteDrawing());

    await act(async () => {
      await result.current.deleteDrawing("drawing-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

describe("useShareDrawing", () => {
  it("shares a drawing to a group", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useShareDrawing());

    let msg;
    await act(async () => {
      msg = await result.current.shareDrawing("drawing-1", { group_id: "group-1" });
    });

    expect(msg).toMatchObject({ id: "msg-1" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDrawingShares", () => {
  it("fetches shares for a drawing", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useDrawingShares("drawing-1"));

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.shares).toHaveLength(1);
    expect(result.current.shares[0]).toMatchObject({ type: "group", name: "Test Group" });
  });

  it("returns empty when drawingId is null", async () => {
    const { result } = renderHook(() => useDrawingShares(null));

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.shares).toEqual([]);
  });
});

describe("useUnshareDrawing", () => {
  it("unshares a drawing", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUnshareDrawing());

    await act(async () => {
      await result.current.unshareDrawing("drawing-1", "msg-share-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

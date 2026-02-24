import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useMyAuditLogs,
  useGroupAuditLogs,
  useAllAuditLogs,
  exportMyAuditLogs,
  exportAllAuditLogs,
  exportMyLocationGPX,
} from "@/lib/hooks/use-audit-logs";

// Capture the REAL createElement before any test spies
const realCreateElement = document.createElement.bind(document);

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useMyAuditLogs", () => {
  it("fetches my audit logs", async () => {
    const { result } = renderHook(() => useMyAuditLogs());

    await act(async () => {
      await result.current.fetch();
    });

    expect(result.current.data).toHaveLength(1);
    expect(result.current.total).toBe(1);
    expect(result.current.data[0].action).toBe("login");
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("passes filter params", async () => {
    server.use(
      http.get("/api/v1/audit-logs/me", ({ request }) => {
        const url = new URL(request.url);
        return HttpResponse.json({
          data: [],
          total: 0,
          page: Number(url.searchParams.get("page") ?? 1),
          page_size: Number(url.searchParams.get("page_size") ?? 20),
        });
      })
    );

    const { result } = renderHook(() => useMyAuditLogs());

    await act(async () => {
      await result.current.fetch({
        action: "login",
        page: 2,
        page_size: 10,
      });
    });

    expect(result.current.data).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });

  it("handles error", async () => {
    server.use(
      http.get("/api/v1/audit-logs/me", () => {
        return HttpResponse.json({ error: { message: "forbidden" } }, { status: 403 });
      })
    );

    const { result } = renderHook(() => useMyAuditLogs());

    await act(async () => {
      await result.current.fetch();
    });

    expect(result.current.error).toBe("forbidden");
  });
});

describe("useGroupAuditLogs", () => {
  it("fetches group audit logs", async () => {
    const { result } = renderHook(() => useGroupAuditLogs());

    await act(async () => {
      await result.current.fetch("group-1");
    });

    expect(result.current.data).toHaveLength(1);
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useAllAuditLogs", () => {
  it("fetches all audit logs (admin)", async () => {
    const { result } = renderHook(() => useAllAuditLogs());

    await act(async () => {
      await result.current.fetch();
    });

    expect(result.current.data).toHaveLength(1);
    expect(result.current.total).toBe(1);
    expect(result.current.isLoading).toBe(false);
  });
});

describe("exportMyAuditLogs", () => {
  it("triggers a CSV download", async () => {
    const mockClick = vi.fn();
    const mockRemove = vi.fn();
    const createObjectURL = vi.fn(() => "blob:url");
    const revokeObjectURL = vi.fn();
    globalThis.URL.createObjectURL = createObjectURL;
    globalThis.URL.revokeObjectURL = revokeObjectURL;

    vi.spyOn(document, "createElement").mockImplementation((tag: string) => {
      if (tag === "a") {
        const el = realCreateElement("a");
        el.click = mockClick;
        el.remove = mockRemove;
        return el;
      }
      return realCreateElement(tag);
    });

    await exportMyAuditLogs("csv");

    expect(createObjectURL).toHaveBeenCalled();
    expect(mockClick).toHaveBeenCalled();
    expect(revokeObjectURL).toHaveBeenCalled();
  });
});

describe("exportAllAuditLogs", () => {
  it("exports admin audit logs", async () => {
    const createObjectURL = vi.fn(() => "blob:url");
    const revokeObjectURL = vi.fn();
    globalThis.URL.createObjectURL = createObjectURL;
    globalThis.URL.revokeObjectURL = revokeObjectURL;

    const mockClick = vi.fn();
    const mockRemove = vi.fn();
    vi.spyOn(document, "createElement").mockImplementation((tag: string) => {
      if (tag === "a") {
        const el = realCreateElement("a");
        el.click = mockClick;
        el.remove = mockRemove;
        return el;
      }
      return realCreateElement(tag);
    });

    await exportAllAuditLogs("json");

    expect(createObjectURL).toHaveBeenCalled();
    expect(mockClick).toHaveBeenCalled();
  });
});

describe("exportMyLocationGPX", () => {
  it("exports location history as GPX", async () => {
    const createObjectURL = vi.fn(() => "blob:url");
    const revokeObjectURL = vi.fn();
    globalThis.URL.createObjectURL = createObjectURL;
    globalThis.URL.revokeObjectURL = revokeObjectURL;

    const mockClick = vi.fn();
    const mockRemove = vi.fn();
    vi.spyOn(document, "createElement").mockImplementation((tag: string) => {
      if (tag === "a") {
        const el = realCreateElement("a");
        el.click = mockClick;
        el.remove = mockRemove;
        return el;
      }
      return realCreateElement(tag);
    });

    const from = new Date("2025-01-01T00:00:00Z");
    const to = new Date("2025-01-01T23:59:59Z");
    await exportMyLocationGPX(from, to);

    expect(createObjectURL).toHaveBeenCalled();
    expect(mockClick).toHaveBeenCalled();
  });
});

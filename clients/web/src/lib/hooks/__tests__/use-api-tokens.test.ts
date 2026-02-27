import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useApiTokens,
  useCreateApiToken,
  useDeleteApiToken,
} from "@/lib/hooks/use-api-tokens";

describe("useApiTokens", () => {
  it("fetches tokens when fetch() is called", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useApiTokens());

    // Initial state — not yet fetched
    expect(result.current.tokens).toEqual([]);
    expect(result.current.isLoading).toBe(false);

    await act(async () => {
      await result.current.fetch();
    });

    expect(result.current.tokens).toHaveLength(1);
    expect(result.current.tokens[0].name).toBe("CI Pipeline");
    expect(result.current.isLoading).toBe(false);
  });

  it("handles error", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.get("/api/v1/users/me/api-tokens", () => {
        return HttpResponse.json(
          { error: { message: "unauthorized" } },
          { status: 401 }
        );
      })
    );

    const { result } = renderHook(() => useApiTokens());

    await act(async () => {
      await result.current.fetch();
    });

    expect(result.current.error).toBe("unauthorized");
  });
});

describe("useCreateApiToken", () => {
  it("creates a new token", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useCreateApiToken());

    let response;
    await act(async () => {
      response = await result.current.createToken({ name: "CI Pipeline" });
    });

    expect(response).toMatchObject({ id: "token-1", name: "CI Pipeline" });
    expect(response?.token).toMatch(/^sat_/);
    expect(result.current.isLoading).toBe(false);
  });

  it("sends expiry when provided", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.post("/api/v1/users/me/api-tokens", async ({ request }) => {
        const body = (await request.json()) as {
          name: string;
          expires_at?: string;
        };
        expect(body.name).toBe("Temp Token");
        expect(body.expires_at).toBe("2026-12-31T00:00:00Z");
        return HttpResponse.json({
          id: "token-2",
          name: body.name,
          expires_at: body.expires_at,
          created_at: "2025-06-01T00:00:00Z",
          token: "sat_abcdef",
        });
      })
    );

    const { result } = renderHook(() => useCreateApiToken());

    await act(async () => {
      await result.current.createToken({
        name: "Temp Token",
        expires_at: "2026-12-31T00:00:00Z",
      });
    });
  });
});

describe("useDeleteApiToken", () => {
  it("deletes a token", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useDeleteApiToken());

    await act(async () => {
      await result.current.deleteToken("token-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

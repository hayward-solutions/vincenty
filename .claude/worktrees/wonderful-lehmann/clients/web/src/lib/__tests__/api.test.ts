import { describe, it, expect, beforeEach } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import { api, ApiError } from "@/lib/api";

// The api.ts module does NOT import auth-context or websocket-context,
// so we do NOT need the mocks from test-utils here.

describe("ApiClient", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  // -----------------------------------------------------------------------
  // GET
  // -----------------------------------------------------------------------

  describe("get()", () => {
    it("performs a GET request and returns JSON", async () => {
      server.use(
        http.get("/api/v1/ping", () => {
          return HttpResponse.json({ status: "ok" });
        })
      );

      const result = await api.get<{ status: string }>("/api/v1/ping");
      expect(result).toEqual({ status: "ok" });
    });

    it("appends query params", async () => {
      server.use(
        http.get("/api/v1/items", ({ request }) => {
          const url = new URL(request.url);
          return HttpResponse.json({
            page: url.searchParams.get("page"),
            size: url.searchParams.get("page_size"),
          });
        })
      );

      const result = await api.get<{ page: string; size: string }>(
        "/api/v1/items",
        { params: { page: "2", page_size: "10" } }
      );
      expect(result.page).toBe("2");
      expect(result.size).toBe("10");
    });
  });

  // -----------------------------------------------------------------------
  // POST
  // -----------------------------------------------------------------------

  describe("post()", () => {
    it("sends JSON body", async () => {
      server.use(
        http.post("/api/v1/items", async ({ request }) => {
          const body = (await request.json()) as { name: string };
          return HttpResponse.json({ id: "1", name: body.name });
        })
      );

      const result = await api.post<{ id: string; name: string }>(
        "/api/v1/items",
        { name: "test" }
      );
      expect(result.name).toBe("test");
    });

    it("sends POST without body", async () => {
      server.use(
        http.post("/api/v1/trigger", () => {
          return HttpResponse.json({ triggered: true });
        })
      );

      const result = await api.post<{ triggered: boolean }>("/api/v1/trigger");
      expect(result.triggered).toBe(true);
    });
  });

  // -----------------------------------------------------------------------
  // PUT
  // -----------------------------------------------------------------------

  describe("put()", () => {
    it("sends a PUT request with JSON body", async () => {
      server.use(
        http.put("/api/v1/items/1", async ({ request }) => {
          const body = (await request.json()) as { name: string };
          return HttpResponse.json({ id: "1", name: body.name });
        })
      );

      const result = await api.put<{ id: string; name: string }>(
        "/api/v1/items/1",
        { name: "updated" }
      );
      expect(result.name).toBe("updated");
    });
  });

  // -----------------------------------------------------------------------
  // DELETE
  // -----------------------------------------------------------------------

  describe("delete()", () => {
    it("returns undefined for 204 responses", async () => {
      server.use(
        http.delete("/api/v1/items/1", () => {
          return new HttpResponse(null, { status: 204 });
        })
      );

      const result = await api.delete("/api/v1/items/1");
      expect(result).toBeUndefined();
    });
  });

  // -----------------------------------------------------------------------
  // Authorization header
  // -----------------------------------------------------------------------

  describe("auth header", () => {
    it("includes Bearer token when access_token is in localStorage", async () => {
      localStorage.setItem("access_token", "my-token");

      server.use(
        http.get("/api/v1/check-auth", ({ request }) => {
          const auth = request.headers.get("Authorization");
          return HttpResponse.json({ auth });
        })
      );

      const result = await api.get<{ auth: string | null }>("/api/v1/check-auth");
      expect(result.auth).toBe("Bearer my-token");
    });

    it("omits Authorization header when no token", async () => {
      server.use(
        http.get("/api/v1/check-auth", ({ request }) => {
          const auth = request.headers.get("Authorization");
          return HttpResponse.json({ auth });
        })
      );

      const result = await api.get<{ auth: string | null }>("/api/v1/check-auth");
      expect(result.auth).toBeNull();
    });
  });

  // -----------------------------------------------------------------------
  // Error handling
  // -----------------------------------------------------------------------

  describe("error handling", () => {
    it("throws ApiError with status and message for non-ok responses", async () => {
      server.use(
        http.get("/api/v1/fail", () => {
          return HttpResponse.json(
            { error: { message: "not found" } },
            { status: 404 }
          );
        })
      );

      await expect(api.get("/api/v1/fail")).rejects.toThrow(ApiError);

      try {
        await api.get("/api/v1/fail");
      } catch (err) {
        expect(err).toBeInstanceOf(ApiError);
        expect((err as ApiError).status).toBe(404);
        expect((err as ApiError).message).toBe("not found");
      }
    });

    it("falls back to statusText when response body has no error.message", async () => {
      server.use(
        http.get("/api/v1/fail-no-body", () => {
          return new HttpResponse("not json", {
            status: 500,
            statusText: "Internal Server Error",
          });
        })
      );

      try {
        await api.get("/api/v1/fail-no-body");
      } catch (err) {
        expect(err).toBeInstanceOf(ApiError);
        expect((err as ApiError).status).toBe(500);
        expect((err as ApiError).message).toBe("Internal Server Error");
      }
    });
  });

  // -----------------------------------------------------------------------
  // Auto-refresh on 401
  // -----------------------------------------------------------------------

  describe("auto-refresh on 401", () => {
    it("refreshes token and retries on 401", async () => {
      localStorage.setItem("access_token", "expired-token");
      localStorage.setItem("refresh_token", "valid-refresh");

      let callCount = 0;

      server.use(
        http.get("/api/v1/protected", ({ request }) => {
          callCount++;
          const auth = request.headers.get("Authorization");
          if (auth === "Bearer expired-token") {
            return HttpResponse.json(
              { error: { message: "unauthorized" } },
              { status: 401 }
            );
          }
          return HttpResponse.json({ data: "success" });
        }),
        http.post("/api/v1/auth/refresh", async ({ request }) => {
          const body = (await request.json()) as { refresh_token: string };
          expect(body.refresh_token).toBe("valid-refresh");
          return HttpResponse.json({
            access_token: "new-access-token",
            refresh_token: "new-refresh-token",
          });
        })
      );

      const result = await api.get<{ data: string }>("/api/v1/protected");
      expect(result.data).toBe("success");
      expect(callCount).toBe(2); // first call 401, retry succeeds
      expect(localStorage.getItem("access_token")).toBe("new-access-token");
      expect(localStorage.getItem("refresh_token")).toBe("new-refresh-token");
    });

    it("does not retry more than once", async () => {
      localStorage.setItem("access_token", "expired-token");
      localStorage.setItem("refresh_token", "valid-refresh");

      server.use(
        http.get("/api/v1/always-401", () => {
          return HttpResponse.json(
            { error: { message: "unauthorized" } },
            { status: 401 }
          );
        }),
        http.post("/api/v1/auth/refresh", () => {
          return HttpResponse.json({
            access_token: "still-bad-token",
            refresh_token: "new-refresh",
          });
        })
      );

      await expect(api.get("/api/v1/always-401")).rejects.toThrow(ApiError);
    });

    it("clears tokens when refresh fails", async () => {
      localStorage.setItem("access_token", "expired-token");
      localStorage.setItem("refresh_token", "bad-refresh");

      server.use(
        http.get("/api/v1/protected", () => {
          return HttpResponse.json(
            { error: { message: "unauthorized" } },
            { status: 401 }
          );
        }),
        http.post("/api/v1/auth/refresh", () => {
          return HttpResponse.json(
            { error: { message: "invalid refresh token" } },
            { status: 401 }
          );
        })
      );

      await expect(api.get("/api/v1/protected")).rejects.toThrow(ApiError);
      expect(localStorage.getItem("access_token")).toBeNull();
      expect(localStorage.getItem("refresh_token")).toBeNull();
    });
  });

  // -----------------------------------------------------------------------
  // upload()
  // -----------------------------------------------------------------------

  describe("upload()", () => {
    it("sends multipart form data without Content-Type header", async () => {
      localStorage.setItem("access_token", "my-token");

      server.use(
        http.put("/api/v1/users/me/avatar", ({ request }) => {
          // Verify no explicit Content-Type (browser sets it with boundary)
          const auth = request.headers.get("Authorization");
          expect(auth).toBe("Bearer my-token");
          return HttpResponse.json({ avatar_url: "/avatars/test.jpg" });
        })
      );

      const formData = new FormData();
      formData.append("avatar", new Blob(["fake-image"]), "test.jpg");

      const result = await api.upload<{ avatar_url: string }>(
        "/api/v1/users/me/avatar",
        formData
      );
      expect(result.avatar_url).toBe("/avatars/test.jpg");
    });

    it("returns undefined for 204 upload response", async () => {
      server.use(
        http.put("/api/v1/upload-no-content", () => {
          return new HttpResponse(null, { status: 204 });
        })
      );

      const result = await api.upload(
        "/api/v1/upload-no-content",
        new FormData()
      );
      expect(result).toBeUndefined();
    });

    it("throws ApiError for failed uploads", async () => {
      server.use(
        http.put("/api/v1/upload-fail", () => {
          return HttpResponse.json(
            { error: { message: "file too large" } },
            { status: 413 }
          );
        })
      );

      await expect(
        api.upload("/api/v1/upload-fail", new FormData())
      ).rejects.toThrow(ApiError);
    });
  });

  // -----------------------------------------------------------------------
  // setTokens / clearTokens
  // -----------------------------------------------------------------------

  describe("token management", () => {
    it("setTokens stores both tokens in localStorage", () => {
      api.setTokens("access-123", "refresh-456");
      expect(localStorage.getItem("access_token")).toBe("access-123");
      expect(localStorage.getItem("refresh_token")).toBe("refresh-456");
    });

    it("clearTokens removes both tokens from localStorage", () => {
      localStorage.setItem("access_token", "abc");
      localStorage.setItem("refresh_token", "def");
      api.clearTokens();
      expect(localStorage.getItem("access_token")).toBeNull();
      expect(localStorage.getItem("refresh_token")).toBeNull();
    });
  });
});

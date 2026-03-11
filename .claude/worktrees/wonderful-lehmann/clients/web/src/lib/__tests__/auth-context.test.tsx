/**
 * Tests for AuthProvider and useAuth.
 *
 * IMPORTANT: This file does NOT import test-utils.tsx because that file
 * mocks @/lib/auth-context at module level. We need the REAL AuthProvider
 * here, with MSW intercepting the API calls.
 */
import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import { AuthProvider, useAuth } from "@/lib/auth-context";
import { mockUser, mockAuthResponse, mockMFAChallenge } from "@/test/fixtures";

function wrapper({ children }: { children: React.ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>;
}

beforeEach(() => {
  localStorage.clear();
});

// -------------------------------------------------------------------------
// useAuth outside provider
// -------------------------------------------------------------------------

describe("useAuth", () => {
  it("throws when used outside AuthProvider", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    expect(() => {
      renderHook(() => useAuth());
    }).toThrow("useAuth must be used within an AuthProvider");

    spy.mockRestore();
  });
});

// -------------------------------------------------------------------------
// Session restore on mount
// -------------------------------------------------------------------------

describe("AuthProvider — session restore", () => {
  it("is unauthenticated when no token in localStorage", async () => {
    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.isAdmin).toBe(false);
  });

  it("restores session from access_token in localStorage", async () => {
    localStorage.setItem("access_token", "valid-token");

    const { result } = renderHook(() => useAuth(), { wrapper });

    // Initially loading
    expect(result.current.isLoading).toBe(true);

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.user).toMatchObject({ username: "testuser" });
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.isAdmin).toBe(false);
  });

  it("clears tokens and stays unauthenticated when /users/me fails", async () => {
    localStorage.setItem("access_token", "expired-token");

    server.use(
      http.get("/api/v1/users/me", () => {
        return HttpResponse.json(
          { error: { message: "unauthorized" } },
          { status: 401 }
        );
      }),
      // Also handle the refresh attempt that the api client makes on 401
      http.post("/api/v1/auth/refresh", () => {
        return HttpResponse.json(
          { error: { message: "invalid" } },
          { status: 401 }
        );
      })
    );

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(localStorage.getItem("access_token")).toBeNull();
  });

  it("sets isAdmin when restored user is admin", async () => {
    localStorage.setItem("access_token", "admin-token");

    server.use(
      http.get("/api/v1/users/me", () => {
        return HttpResponse.json({ ...mockUser, is_admin: true });
      })
    );

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.isAdmin).toBe(true);
  });
});

// -------------------------------------------------------------------------
// Login
// -------------------------------------------------------------------------

describe("AuthProvider — login", () => {
  it("logs in successfully and stores tokens", async () => {
    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    let loginResult;
    await act(async () => {
      loginResult = await result.current.login("testuser", "password123");
    });

    expect(loginResult).toMatchObject({
      access_token: "test-access-token",
      user: { username: "testuser" },
    });
    expect(result.current.user).toMatchObject({ username: "testuser" });
    expect(result.current.isAuthenticated).toBe(true);
    expect(localStorage.getItem("access_token")).toBe("test-access-token");
    expect(localStorage.getItem("refresh_token")).toBe("test-refresh-token");
  });

  it("returns MFA challenge when MFA is required", async () => {
    server.use(
      http.post("/api/v1/auth/login", () => {
        return HttpResponse.json(mockMFAChallenge);
      })
    );

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    let loginResult;
    await act(async () => {
      loginResult = await result.current.login("testuser", "password123");
    });

    expect(loginResult).toMatchObject({
      mfa_required: true,
      mfa_token: "mfa-token-123",
      methods: ["totp", "recovery"],
    });
    // User should NOT be set yet
    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });
});

// -------------------------------------------------------------------------
// completeMFALogin
// -------------------------------------------------------------------------

describe("AuthProvider — completeMFALogin", () => {
  it("completes MFA login and sets user + tokens", async () => {
    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => {
      result.current.completeMFALogin(mockAuthResponse);
    });

    expect(result.current.user).toMatchObject({ username: "testuser" });
    expect(result.current.isAuthenticated).toBe(true);
    expect(localStorage.getItem("access_token")).toBe("test-access-token");
    expect(localStorage.getItem("refresh_token")).toBe("test-refresh-token");
  });
});

// -------------------------------------------------------------------------
// Logout
// -------------------------------------------------------------------------

describe("AuthProvider — logout", () => {
  it("logs out, clears tokens, and resets user", async () => {
    localStorage.setItem("access_token", "valid-token");
    localStorage.setItem("refresh_token", "valid-refresh");

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    await act(async () => {
      await result.current.logout();
    });

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(localStorage.getItem("access_token")).toBeNull();
    expect(localStorage.getItem("refresh_token")).toBeNull();
  });

  it("clears user even if logout API call fails", async () => {
    localStorage.setItem("access_token", "valid-token");
    localStorage.setItem("refresh_token", "valid-refresh");

    server.use(
      http.post("/api/v1/auth/logout", () => {
        return HttpResponse.json(
          { error: { message: "server error" } },
          { status: 500 }
        );
      })
    );

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    await act(async () => {
      await result.current.logout();
    });

    // User is cleared regardless of API error
    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(localStorage.getItem("access_token")).toBeNull();
  });

  it("skips API call when no refresh token", async () => {
    localStorage.setItem("access_token", "valid-token");
    // No refresh_token

    let logoutCalled = false;
    server.use(
      http.post("/api/v1/auth/logout", () => {
        logoutCalled = true;
        return new HttpResponse(null, { status: 204 });
      })
    );

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    await act(async () => {
      await result.current.logout();
    });

    expect(logoutCalled).toBe(false);
    expect(result.current.user).toBeNull();
  });
});

// -------------------------------------------------------------------------
// refreshUser
// -------------------------------------------------------------------------

describe("AuthProvider — refreshUser", () => {
  it("refreshes the current user", async () => {
    localStorage.setItem("access_token", "valid-token");

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    // Override /users/me to return updated display name
    server.use(
      http.get("/api/v1/users/me", () => {
        return HttpResponse.json({
          ...mockUser,
          display_name: "Refreshed Name",
        });
      })
    );

    await act(async () => {
      await result.current.refreshUser();
    });

    expect(result.current.user?.display_name).toBe("Refreshed Name");
  });

  it("silently ignores errors during refresh", async () => {
    localStorage.setItem("access_token", "valid-token");

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    server.use(
      http.get("/api/v1/users/me", () => {
        return HttpResponse.json(
          { error: { message: "error" } },
          { status: 500 }
        );
      }),
      http.post("/api/v1/auth/refresh", () => {
        return HttpResponse.json(
          { error: { message: "error" } },
          { status: 500 }
        );
      })
    );

    await act(async () => {
      await result.current.refreshUser();
    });

    // User remains from previous state
    expect(result.current.user).toMatchObject({ username: "testuser" });
  });
});

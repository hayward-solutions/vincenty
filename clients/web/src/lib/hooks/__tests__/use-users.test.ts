import { describe, it, expect } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils"; // activate mocks

import { useUsers, useCreateUser, useUpdateUser, useDeleteUser } from "@/lib/hooks/use-users";

describe("useUsers", () => {
  it("fetches paginated user list on mount", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUsers());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).not.toBeNull();
    expect(result.current.data?.data).toHaveLength(2);
    expect(result.current.data?.total).toBe(2);
    expect(result.current.error).toBeNull();
  });

  it("passes pagination params", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.get("/api/v1/users", ({ request }) => {
        const url = new URL(request.url);
        return HttpResponse.json({
          data: [],
          total: 0,
          page: Number(url.searchParams.get("page")),
          page_size: Number(url.searchParams.get("page_size")),
        });
      })
    );

    const { result } = renderHook(() => useUsers(3, 5));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data?.page).toBe(3);
    expect(result.current.data?.page_size).toBe(5);
  });

  it("handles API errors", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.get("/api/v1/users", () => {
        return HttpResponse.json(
          { error: { message: "forbidden" } },
          { status: 403 }
        );
      })
    );

    const { result } = renderHook(() => useUsers());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.error).toBe("forbidden");
  });

  it("exposes refetch function", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUsers());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    // Refetch should work
    await act(async () => {
      await result.current.refetch();
    });

    expect(result.current.data?.data).toHaveLength(2);
  });
});

describe("useCreateUser", () => {
  it("creates user and manages loading", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useCreateUser());

    let user;
    await act(async () => {
      user = await result.current.createUser({
        username: "newuser",
        email: "new@test.com",
        password: "password123",
      });
    });

    expect(user).toMatchObject({ id: "user-new" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUpdateUser", () => {
  it("updates a user by ID", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUpdateUser());

    let user;
    await act(async () => {
      user = await result.current.updateUser("user-1", { display_name: "Updated" });
    });

    expect(user).toBeDefined();
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteUser", () => {
  it("deletes a user by ID", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useDeleteUser());

    await act(async () => {
      await result.current.deleteUser("user-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

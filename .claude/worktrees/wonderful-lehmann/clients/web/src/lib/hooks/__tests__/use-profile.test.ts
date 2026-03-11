import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils"; // activate mocks

import { useUpdateMe, useChangePassword, useUploadAvatar, useDeleteAvatar } from "@/lib/hooks/use-profile";

describe("useUpdateMe", () => {
  it("updates user profile and manages loading state", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUpdateMe());

    expect(result.current.isLoading).toBe(false);

    let updated: unknown;
    await act(async () => {
      updated = await result.current.updateMe({ display_name: "New Name" });
    });

    expect(updated).toMatchObject({ display_name: "Updated Name" });
    expect(result.current.isLoading).toBe(false);
  });

  it("resets loading on error", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.put("/api/v1/users/me", () => {
        return HttpResponse.json(
          { error: { message: "bad request" } },
          { status: 400 }
        );
      })
    );

    const { result } = renderHook(() => useUpdateMe());

    let error: unknown;
    await act(async () => {
      try {
        await result.current.updateMe({ display_name: "" });
      } catch (e) {
        error = e;
      }
    });

    expect(error).toBeDefined();
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useChangePassword", () => {
  it("changes password successfully", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useChangePassword());

    await act(async () => {
      await result.current.changePassword({
        current_password: "old",
        new_password: "new",
      });
    });

    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUploadAvatar", () => {
  it("uploads avatar via FormData", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUploadAvatar());

    const file = new File(["fake-image"], "avatar.jpg", { type: "image/jpeg" });

    let user: unknown;
    await act(async () => {
      user = await result.current.uploadAvatar(file);
    });

    expect(user).toMatchObject({ avatar_url: "/avatars/test.jpg" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteAvatar", () => {
  it("deletes avatar", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useDeleteAvatar());

    let user: unknown;
    await act(async () => {
      user = await result.current.deleteAvatar();
    });

    expect(user).toMatchObject({ avatar_url: "" });
    expect(result.current.isLoading).toBe(false);
  });
});

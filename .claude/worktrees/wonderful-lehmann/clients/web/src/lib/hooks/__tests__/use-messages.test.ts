import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useSendMessage,
  useGroupMessages,
  useDirectMessages,
  useDeleteMessage,
} from "@/lib/hooks/use-messages";

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

describe("useSendMessage", () => {
  it("sends a message via multipart form data", async () => {
    const { result } = renderHook(() => useSendMessage());

    let msg;
    await act(async () => {
      msg = await result.current.sendMessage({
        content: "Hello",
        groupId: "group-1",
      });
    });

    expect(msg).toMatchObject({ id: "msg-1", content: "Hello, world!" });
    expect(result.current.isLoading).toBe(false);
  });

  it("includes device_id from localStorage", async () => {
    localStorage.setItem("device_id", "device-1");

    server.use(
      http.post("/api/v1/messages", async ({ request }) => {
        const formData = await request.formData();
        const deviceId = formData.get("device_id");
        return HttpResponse.json({
          id: "msg-2",
          sender_id: "user-1",
          username: "testuser",
          display_name: "Test User",
          content: "test",
          message_type: "text",
          attachments: [],
          created_at: "2025-01-01T00:00:00Z",
          device_id: deviceId,
        });
      })
    );

    const { result } = renderHook(() => useSendMessage());

    await act(async () => {
      await result.current.sendMessage({ content: "test" });
    });
  });

  it("throws ApiError on failure", async () => {
    server.use(
      http.post("/api/v1/messages", () => {
        return HttpResponse.json(
          { error: { message: "content required" } },
          { status: 400 }
        );
      })
    );

    const { result } = renderHook(() => useSendMessage());

    await expect(
      act(async () => {
        await result.current.sendMessage({});
      })
    ).rejects.toThrow("content required");
  });
});

describe("useGroupMessages", () => {
  it("fetches group messages on mount", async () => {
    const { result } = renderHook(() => useGroupMessages("group-1"));

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.messages).toHaveLength(1);
    expect(result.current.messages[0].content).toBe("Hello, world!");
    expect(result.current.error).toBeNull();
  });

  it("does not fetch when groupId is null", async () => {
    const { result } = renderHook(() => useGroupMessages(null));

    // Should not trigger loading at all
    expect(result.current.messages).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });

  it("handles load error", async () => {
    server.use(
      http.get("/api/v1/groups/:groupId/messages", () => {
        return HttpResponse.json(
          { error: { message: "not found" } },
          { status: 404 }
        );
      })
    );

    const { result } = renderHook(() => useGroupMessages("group-unknown"));

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.error).toBe("not found");
  });

  it("supports addOptimistic", async () => {
    const { result } = renderHook(() => useGroupMessages("group-1"));

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => {
      result.current.addOptimistic({
        id: "msg-optimistic",
        sender_id: "user-1",
        username: "testuser",
        display_name: "Test User",
        group_id: "group-1",
        content: "Optimistic message",
        message_type: "text",
        attachments: [],
        created_at: new Date().toISOString(),
      });
    });

    expect(result.current.messages).toHaveLength(2);
    expect(result.current.messages[0].content).toBe("Optimistic message");
  });
});

describe("useDirectMessages", () => {
  it("fetches direct messages on mount", async () => {
    const { result } = renderHook(() => useDirectMessages("user-2"));

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.messages).toHaveLength(1);
    expect(result.current.error).toBeNull();
  });

  it("does not fetch when otherUserId is null", async () => {
    const { result } = renderHook(() => useDirectMessages(null));

    expect(result.current.messages).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteMessage", () => {
  it("deletes a message", async () => {
    const { result } = renderHook(() => useDeleteMessage());

    await act(async () => {
      await result.current.deleteMessage("msg-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

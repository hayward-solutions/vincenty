import { describe, it, expect } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import "@/test/test-utils";

import { useConversations } from "@/lib/hooks/use-conversations";

describe("useConversations", () => {
  it("fetches groups and DM partners on mount", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useConversations());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    // Should have one group + one DM partner
    expect(result.current.conversations).toHaveLength(2);
    expect(result.current.conversations[0]).toMatchObject({
      type: "group",
      name: "Test Group",
    });
    expect(result.current.conversations[1]).toMatchObject({
      type: "direct",
      name: "Other User",
    });
    expect(result.current.error).toBeNull();
  });

  it("addDmConversation adds a new DM if not already present", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useConversations());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    let conv;
    act(() => {
      conv = result.current.addDmConversation("user-3", "Third User");
    });

    expect(conv).toMatchObject({ id: "user-3", type: "direct", name: "Third User" });
    expect(result.current.conversations).toHaveLength(3);
  });

  it("addDmConversation returns existing conversation if already present", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useConversations());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    let conv;
    act(() => {
      conv = result.current.addDmConversation("user-2", "Other User");
    });

    expect(conv).toMatchObject({ id: "user-2", type: "direct" });
    // Should not add a duplicate
    expect(result.current.conversations).toHaveLength(2);
  });

  it("exposes refetch", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useConversations());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      await result.current.refetch();
    });

    expect(result.current.conversations).toHaveLength(2);
  });
});

import { describe, it, expect } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useGroups,
  useGroup,
  useCreateGroup,
  useUpdateGroup,
  useDeleteGroup,
  useUpdateGroupMarker,
  useGroupMembers,
  useAddGroupMember,
  useUpdateGroupMember,
  useRemoveGroupMember,
} from "@/lib/hooks/use-groups";

describe("useGroups", () => {
  it("fetches paginated group list on mount", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useGroups());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.data?.data).toHaveLength(1);
    expect(result.current.data?.data[0].name).toBe("Test Group");
    expect(result.current.error).toBeNull();
  });

  it("handles fetch error", async () => {
    localStorage.setItem("access_token", "test-token");

    server.use(
      http.get("/api/v1/groups", () => {
        return HttpResponse.json({ error: { message: "server error" } }, { status: 500 });
      })
    );

    const { result } = renderHook(() => useGroups());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.error).toBe("server error");
  });
});

describe("useGroup", () => {
  it("fetches a single group by ID", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useGroup("group-1"));

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.group?.name).toBe("Test Group");
    expect(result.current.error).toBeNull();
  });
});

describe("useCreateGroup", () => {
  it("creates a group", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useCreateGroup());

    let group;
    await act(async () => {
      group = await result.current.createGroup({ name: "New Group" });
    });

    expect(group).toMatchObject({ name: "Test Group" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUpdateGroup", () => {
  it("updates a group", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUpdateGroup());

    let group;
    await act(async () => {
      group = await result.current.updateGroup("group-1", { name: "Updated" });
    });

    expect(group).toMatchObject({ name: "Updated Group" });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteGroup", () => {
  it("deletes a group", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useDeleteGroup());

    await act(async () => {
      await result.current.deleteGroup("group-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUpdateGroupMarker", () => {
  it("updates group marker", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUpdateGroupMarker());

    let group;
    await act(async () => {
      group = await result.current.updateMarker("group-1", { marker_color: "#ff0000" });
    });

    expect(group).toBeDefined();
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useGroupMembers", () => {
  it("fetches group members on mount", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useGroupMembers("group-1"));

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.members).toHaveLength(1);
    expect(result.current.members[0].username).toBe("testuser");
    expect(result.current.error).toBeNull();
  });
});

describe("useAddGroupMember", () => {
  it("adds a member to a group", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useAddGroupMember());

    let member;
    await act(async () => {
      member = await result.current.addMember("group-1", {
        user_id: "user-2",
        can_read: true,
        can_write: true,
      });
    });

    expect(member).toBeDefined();
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useUpdateGroupMember", () => {
  it("updates a group member", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useUpdateGroupMember());

    let member;
    await act(async () => {
      member = await result.current.updateMember("group-1", "user-1", {
        is_group_admin: true,
      });
    });

    expect(member).toMatchObject({ is_group_admin: true });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useRemoveGroupMember", () => {
  it("removes a member from a group", async () => {
    localStorage.setItem("access_token", "test-token");

    const { result } = renderHook(() => useRemoveGroupMember());

    await act(async () => {
      await result.current.removeMember("group-1", "user-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

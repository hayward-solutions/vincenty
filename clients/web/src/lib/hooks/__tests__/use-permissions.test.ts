import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import { usePermissionPolicy } from "@/lib/hooks/use-permissions";

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

describe("usePermissionPolicy", () => {
  it("fetches permission policy on mount", async () => {
    const { result } = renderHook(() => usePermissionPolicy());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.policy).not.toBeNull();
    expect(result.current.policy!.group_communication.send_messages).toEqual([
      "server_admin",
      "group_admin",
      "writer",
    ]);
    expect(result.current.policy!.group_management.add_members).toEqual([
      "group_admin",
    ]);
  });

  it("returns null policy on error", async () => {
    server.use(
      http.get("/api/v1/server/permissions", () => {
        return HttpResponse.json(
          { error: { message: "forbidden" } },
          { status: 403 }
        );
      })
    );

    const { result } = renderHook(() => usePermissionPolicy());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.policy).toBeNull();
  });

  it("updates permission policy", async () => {
    const { result } = renderHook(() => usePermissionPolicy());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    const updated = {
      group_communication: {
        ...result.current.policy!.group_communication,
        send_messages: ["server_admin", "group_admin", "writer", "reader"],
      },
      group_management: {
        ...result.current.policy!.group_management,
      },
    };

    let response;
    await act(async () => {
      response = await result.current.update(updated);
    });

    expect(response).toMatchObject({
      group_communication: {
        send_messages: ["server_admin", "group_admin", "writer", "reader"],
      },
    });
  });

  it("refetch reloads the policy", async () => {
    const { result } = renderHook(() => usePermissionPolicy());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    // Override with a different response
    server.use(
      http.get("/api/v1/server/permissions", () => {
        return HttpResponse.json({
          group_communication: {
            send_messages: ["server_admin"],
            read_messages: ["server_admin"],
            send_attachments: ["server_admin"],
            share_drawings: ["server_admin"],
            share_location: ["server_admin"],
            view_locations: ["server_admin"],
          },
          group_management: {
            add_members: ["group_admin"],
            remove_members: ["group_admin"],
            update_members: ["group_admin"],
            update_marker: ["group_admin"],
            view_audit_logs: ["group_admin"],
          },
        });
      })
    );

    await act(async () => {
      await result.current.refetch();
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.policy!.group_communication.send_messages).toEqual([
      "server_admin",
    ]);
  });
});

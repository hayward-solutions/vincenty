import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render, mockAuth } from "@/test/test-utils";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import { NewDmDialog } from "./new-dm-dialog";

// ResizeObserver polyfill for Radix ScrollArea
if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof globalThis.ResizeObserver;
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

function renderDialog(
  overrides: Partial<{
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSelect: (userId: string, displayName: string) => void;
  }> = {}
) {
  const props = {
    open: true,
    onOpenChange: vi.fn(),
    onSelect: vi.fn(),
    ...overrides,
  };
  return { ...render(<NewDmDialog {...props} />), props };
}

beforeEach(() => {
  vi.clearAllMocks();
  // Reset to non-admin by default
  mockAuth.isAdmin = false;
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("NewDmDialog", () => {
  it("does not render content when closed", () => {
    renderDialog({ open: false });
    expect(
      screen.queryByText("New Direct Message")
    ).not.toBeInTheDocument();
  });

  it("renders title and search input when open", () => {
    renderDialog();
    expect(screen.getByText("New Direct Message")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("Search users...")
    ).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Non-admin: fetches group members
  // -----------------------------------------------------------------------

  describe("non-admin user", () => {
    beforeEach(() => {
      // Mock groups → returns one group
      server.use(
        http.get("/api/v1/users/me/groups", () => {
          return HttpResponse.json([
            { id: "g1", name: "Group 1" },
          ]);
        }),
        http.get("/api/v1/groups/g1/members", () => {
          return HttpResponse.json([
            {
              user_id: "user-1", // self — should be filtered out
              username: "testuser",
              display_name: "Test User",
            },
            {
              user_id: "user-2",
              username: "alice",
              display_name: "Alice",
            },
            {
              user_id: "user-3",
              username: "bob",
              display_name: "Bob",
            },
          ]);
        })
      );
    });

    it("fetches and displays group members (excluding self)", async () => {
      renderDialog();

      await waitFor(() => {
        expect(screen.getByText("Alice")).toBeInTheDocument();
        expect(screen.getByText("Bob")).toBeInTheDocument();
      });

      expect(screen.queryByText("Test User")).not.toBeInTheDocument();
    });

    it("filters users by search query", async () => {
      const user = userEvent.setup();
      renderDialog();

      await waitFor(() => {
        expect(screen.getByText("Alice")).toBeInTheDocument();
      });

      await user.type(screen.getByPlaceholderText("Search users..."), "ali");

      expect(screen.getByText("Alice")).toBeInTheDocument();
      expect(screen.queryByText("Bob")).not.toBeInTheDocument();
    });

    it("shows 'No matching users' when search has no results", async () => {
      const user = userEvent.setup();
      renderDialog();

      await waitFor(() => {
        expect(screen.getByText("Alice")).toBeInTheDocument();
      });

      await user.type(
        screen.getByPlaceholderText("Search users..."),
        "zzzzz"
      );

      expect(screen.getByText("No matching users")).toBeInTheDocument();
    });

    it("calls onSelect and onOpenChange(false) when user is clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderDialog();

      await waitFor(() => {
        expect(screen.getByText("Alice")).toBeInTheDocument();
      });

      await user.click(screen.getByText("Alice"));

      expect(props.onSelect).toHaveBeenCalledWith("user-2", "Alice");
      expect(props.onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  // -----------------------------------------------------------------------
  // Admin: fetches all users
  // -----------------------------------------------------------------------

  describe("admin user", () => {
    beforeEach(() => {
      mockAuth.isAdmin = true;
      server.use(
        http.get("/api/v1/users", () => {
          return HttpResponse.json({
            data: [
              {
                id: "user-1",
                username: "testuser",
                display_name: "Test User",
                is_active: true,
              },
              {
                id: "user-4",
                username: "charlie",
                display_name: "Charlie",
                is_active: true,
              },
              {
                id: "user-5",
                username: "inactive",
                display_name: "Inactive",
                is_active: false,
              },
            ],
            total: 3,
            page: 1,
            page_size: 200,
          });
        })
      );
    });

    it("fetches all active users excluding self", async () => {
      renderDialog();

      await waitFor(() => {
        expect(screen.getByText("Charlie")).toBeInTheDocument();
      });

      // Self and inactive should be filtered
      expect(screen.queryByText("Test User")).not.toBeInTheDocument();
      expect(screen.queryByText("Inactive")).not.toBeInTheDocument();
    });
  });

  // -----------------------------------------------------------------------
  // Empty state
  // -----------------------------------------------------------------------

  it("shows 'No users available' when no users returned", async () => {
    server.use(
      http.get("/api/v1/users/me/groups", () => {
        return HttpResponse.json([]);
      })
    );

    renderDialog();

    await waitFor(() => {
      expect(
        screen.getByText("No users available to message")
      ).toBeInTheDocument();
    });
  });

  // -----------------------------------------------------------------------
  // Shows username when different from displayName
  // -----------------------------------------------------------------------

  it("shows @username when different from display name", async () => {
    server.use(
      http.get("/api/v1/users/me/groups", () => {
        return HttpResponse.json([{ id: "g1", name: "G1" }]);
      }),
      http.get("/api/v1/groups/g1/members", () => {
        return HttpResponse.json([
          {
            user_id: "user-10",
            username: "jdoe",
            display_name: "John Doe",
          },
        ]);
      })
    );

    renderDialog();

    await waitFor(() => {
      expect(screen.getByText("John Doe")).toBeInTheDocument();
      expect(screen.getByText("@jdoe")).toBeInTheDocument();
    });
  });
});

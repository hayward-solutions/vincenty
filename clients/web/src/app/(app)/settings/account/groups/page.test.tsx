import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import AccountGroupsPage from "./page";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  apiGet: vi.fn(),
  updateMarker: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@/lib/api", () => ({
  api: { get: mocks.apiGet },
  ApiError: class ApiError extends Error {},
}));

vi.mock("@/lib/hooks/use-groups", () => ({
  useUpdateGroupMarker: () => ({
    updateMarker: mocks.updateMarker,
    isLoading: false,
  }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const mockGroups = [
  {
    id: "group-1",
    name: "Alpha Team",
    description: "First team",
    marker_icon: "circle",
    marker_color: "#3b82f6",
    member_count: 5,
    created_by: "user-1",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
];

const mockMembers = [
  {
    id: "m1",
    group_id: "group-1",
    user_id: "user-1",
    username: "testuser",
    display_name: "Test User",
    can_read: true,
    can_write: true,
    is_group_admin: true,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
];

const mockMembersNonAdmin = [
  {
    id: "m1",
    group_id: "group-1",
    user_id: "user-1",
    username: "testuser",
    display_name: "Test User",
    can_read: true,
    can_write: true,
    is_group_admin: false,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
];

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.clearAllMocks();
  mocks.apiGet.mockImplementation((url: string) => {
    if (url === "/api/v1/users/me/groups") return Promise.resolve(mockGroups);
    if (url.includes("/members")) return Promise.resolve(mockMembers);
    return Promise.reject(new Error("unhandled"));
  });
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("AccountGroupsPage", () => {
  it("renders heading 'My Groups'", async () => {
    render(<AccountGroupsPage />);
    // The heading appears in both loading and loaded states
    expect(screen.getByText("My Groups")).toBeInTheDocument();
  });

  it("shows loading skeletons initially", () => {
    // Make apiGet return a promise that never resolves so the page stays in loading state
    mocks.apiGet.mockImplementation(() => new Promise(() => {}));
    render(<AccountGroupsPage />);

    expect(screen.getByText("My Groups")).toBeInTheDocument();
    // The loading state renders two Skeleton elements (h-12 w-full)
    // Skeletons render as divs with the skeleton class
    const skeletons = document.querySelectorAll('[class*="h-12"]');
    expect(skeletons.length).toBe(2);
  });

  it("shows group name and description after loading", async () => {
    render(<AccountGroupsPage />);

    await waitFor(() => {
      expect(screen.getByText("Alpha Team")).toBeInTheDocument();
    });
    expect(screen.getByText("First team")).toBeInTheDocument();
  });

  it("shows 'Edit Marker' button when user is group admin", async () => {
    render(<AccountGroupsPage />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /edit marker/i })
      ).toBeInTheDocument();
    });
  });

  it("shows 'Admin only' text when user is not group admin", async () => {
    mocks.apiGet.mockImplementation((url: string) => {
      if (url === "/api/v1/users/me/groups") return Promise.resolve(mockGroups);
      if (url.includes("/members"))
        return Promise.resolve(mockMembersNonAdmin);
      return Promise.reject(new Error("unhandled"));
    });

    render(<AccountGroupsPage />);

    await waitFor(() => {
      expect(screen.getByText("Admin only")).toBeInTheDocument();
    });
    expect(
      screen.queryByRole("button", { name: /edit marker/i })
    ).not.toBeInTheDocument();
  });

  it("shows empty state when user has no groups", async () => {
    mocks.apiGet.mockImplementation((url: string) => {
      if (url === "/api/v1/users/me/groups") return Promise.resolve([]);
      return Promise.reject(new Error("unhandled"));
    });

    render(<AccountGroupsPage />);

    await waitFor(() => {
      expect(
        screen.getByText("You are not a member of any groups.")
      ).toBeInTheDocument();
    });
  });

  it("shows marker shape label badge", async () => {
    render(<AccountGroupsPage />);

    await waitFor(() => {
      expect(screen.getByText("Circle")).toBeInTheDocument();
    });
  });

  it("opens the marker editor dialog when 'Edit Marker' is clicked", async () => {
    const user = userEvent.setup();
    render(<AccountGroupsPage />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /edit marker/i })
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /edit marker/i }));

    await waitFor(() => {
      expect(
        screen.getByText("Edit Marker - Alpha Team")
      ).toBeInTheDocument();
    });
    // Dialog should contain Save and Cancel buttons
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /cancel/i })
    ).toBeInTheDocument();
    // Dialog should contain shape and color labels
    expect(screen.getByText("Shape")).toBeInTheDocument();
    expect(screen.getByText("Color")).toBeInTheDocument();
  });
});

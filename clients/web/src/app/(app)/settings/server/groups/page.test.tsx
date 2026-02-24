import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import GroupsSettingsPage from "./page";
import type { Group, ListResponse } from "@/types/api";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({ toast: mockToast }));

const mockGroupsHook = vi.hoisted(() => ({
  data: null as ListResponse<Group> | null,
  isLoading: false,
  refetch: vi.fn(),
}));

const mockCreateGroupHook = vi.hoisted(() => ({
  createGroup: vi.fn().mockResolvedValue({}),
  isLoading: false,
}));

const mockDeleteGroupHook = vi.hoisted(() => ({
  deleteGroup: vi.fn().mockResolvedValue(undefined),
}));

const mockUpdateGroupHook = vi.hoisted(() => ({
  updateGroup: vi.fn().mockResolvedValue({}),
  isLoading: false,
}));

vi.mock("@/lib/hooks/use-groups", () => ({
  useGroups: () => mockGroupsHook,
  useCreateGroup: () => mockCreateGroupHook,
  useDeleteGroup: () => mockDeleteGroupHook,
  useUpdateGroup: () => mockUpdateGroupHook,
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

vi.mock("@/components/map/marker-shapes", () => ({
  markerSVGString: () => '<svg data-testid="marker-icon"></svg>',
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => "/settings/server/groups",
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
  }: {
    children: React.ReactNode;
    href: string;
  }) => <a href={href}>{children}</a>,
}));

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const mockGroupsData: ListResponse<Group> = {
  data: [
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
    {
      id: "group-2",
      name: "Bravo Team",
      description: "",
      marker_icon: "diamond",
      marker_color: "#10b981",
      member_count: 3,
      created_by: "user-1",
      created_at: "2025-02-01T00:00:00Z",
      updated_at: "2025-02-01T00:00:00Z",
    },
  ],
  total: 2,
  page: 1,
  page_size: 20,
};

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.clearAllMocks();
  mockGroupsHook.data = null;
  mockGroupsHook.isLoading = false;
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("GroupsSettingsPage", () => {
  it("renders the Groups heading and Create Group button", () => {
    render(<GroupsSettingsPage />);
    expect(screen.getByText("Groups")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /create group/i })
    ).toBeInTheDocument();
  });

  it("renders group names in the table", () => {
    mockGroupsHook.data = mockGroupsData;
    render(<GroupsSettingsPage />);

    expect(screen.getByText("Alpha Team")).toBeInTheDocument();
    expect(screen.getByText("Bravo Team")).toBeInTheDocument();
  });

  it("shows member count for each group", () => {
    mockGroupsHook.data = mockGroupsData;
    render(<GroupsSettingsPage />);

    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it('shows "-" for empty description', () => {
    mockGroupsHook.data = mockGroupsData;
    render(<GroupsSettingsPage />);

    // Alpha Team has "First team", Bravo Team has empty string -> "-"
    expect(screen.getByText("First team")).toBeInTheDocument();
    expect(screen.getByText("-")).toBeInTheDocument();
  });

  it("links group names to the detail page", () => {
    mockGroupsHook.data = mockGroupsData;
    render(<GroupsSettingsPage />);

    const alphaLink = screen.getByText("Alpha Team").closest("a");
    expect(alphaLink).toHaveAttribute(
      "href",
      "/settings/server/groups/group-1"
    );

    const bravoLink = screen.getByText("Bravo Team").closest("a");
    expect(bravoLink).toHaveAttribute(
      "href",
      "/settings/server/groups/group-2"
    );
  });

  it("shows 'No groups found' when data is empty", () => {
    mockGroupsHook.data = { data: [], total: 0, page: 1, page_size: 20 };
    render(<GroupsSettingsPage />);
    expect(screen.getByText("No groups found")).toBeInTheDocument();
  });

  it("shows table column headers", () => {
    mockGroupsHook.data = mockGroupsData;
    render(<GroupsSettingsPage />);

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Description")).toBeInTheDocument();
    expect(screen.getByText("Members")).toBeInTheDocument();
    expect(screen.getByText("Created")).toBeInTheDocument();
  });

  it("has action menu trigger buttons for each row", () => {
    mockGroupsHook.data = mockGroupsData;
    render(<GroupsSettingsPage />);

    // Each row has a "..." trigger button
    const triggers = screen.getAllByRole("button", { name: "..." });
    expect(triggers).toHaveLength(2);
  });

  it("calls deleteGroup when delete is confirmed", async () => {
    mockGroupsHook.data = mockGroupsData;
    const user = userEvent.setup();
    const confirmSpy = vi
      .spyOn(window, "confirm")
      .mockReturnValue(true);

    render(<GroupsSettingsPage />);

    // Click the first row's "..." trigger
    const triggers = screen.getAllByRole("button", { name: "..." });
    await user.click(triggers[0]);

    // Wait for the dropdown menu to appear and click Delete
    await waitFor(() => {
      expect(screen.getByText("Delete")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Delete"));

    expect(confirmSpy).toHaveBeenCalledWith(
      'Delete group "Alpha Team"? This will remove all members.'
    );
    await waitFor(() => {
      expect(mockDeleteGroupHook.deleteGroup).toHaveBeenCalledWith("group-1");
    });
    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith(
        'Group "Alpha Team" deleted'
      );
    });
    expect(mockGroupsHook.refetch).toHaveBeenCalled();

    confirmSpy.mockRestore();
  });

  it("shows pagination when total exceeds page size", () => {
    mockGroupsHook.data = {
      data: Array(20).fill(mockGroupsData.data[0]),
      total: 25,
      page: 1,
      page_size: 20,
    };
    render(<GroupsSettingsPage />);

    expect(screen.getByText(/showing 1-20 of 25/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /next/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled();
  });

  it("does not show pagination when total fits in one page", () => {
    mockGroupsHook.data = mockGroupsData; // total: 2, page_size: 20
    render(<GroupsSettingsPage />);

    expect(
      screen.queryByRole("button", { name: /next/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /previous/i })
    ).not.toBeInTheDocument();
  });
});

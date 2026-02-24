import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import GroupDetailPage from "./page";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  routerPush: vi.fn(),
  group: {
    id: "group-1",
    name: "Alpha Team",
    description: "First team",
    marker_icon: "circle",
    marker_color: "#3b82f6",
    member_count: 2,
    created_by: "user-1",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  } as {
    id: string;
    name: string;
    description: string;
    marker_icon: string;
    marker_color: string;
    member_count: number;
    created_by: string;
    created_at: string;
    updated_at: string;
  } | null,
  groupLoading: false,
  membersLoading: false,
  members: [
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
    {
      id: "m2",
      group_id: "group-1",
      user_id: "user-2",
      username: "otheruser",
      display_name: "Other User",
      can_read: true,
      can_write: false,
      is_group_admin: false,
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
  ],
  refetchGroup: vi.fn(),
  refetchMembers: vi.fn(),
  updateMarker: vi.fn().mockResolvedValue(undefined),
  addMember: vi.fn().mockResolvedValue(undefined),
  updateMember: vi.fn().mockResolvedValue(undefined),
  removeMember: vi.fn().mockResolvedValue(undefined),
  auditFetch: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useParams: () => ({ id: "group-1" }),
  useRouter: () => ({ push: mocks.routerPush }),
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...props
  }: {
    children: React.ReactNode;
    href: string;
    [key: string]: unknown;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

vi.mock("@/lib/hooks/use-groups", () => ({
  useGroup: () => ({
    group: mocks.group,
    isLoading: mocks.groupLoading,
    refetch: mocks.refetchGroup,
  }),
  useUpdateGroupMarker: () => ({
    updateMarker: mocks.updateMarker,
    isLoading: false,
  }),
  useGroupMembers: () => ({
    members: mocks.members,
    isLoading: mocks.membersLoading,
    refetch: mocks.refetchMembers,
  }),
  useAddGroupMember: () => ({
    addMember: mocks.addMember,
    isLoading: false,
  }),
  useUpdateGroupMember: () => ({
    updateMember: mocks.updateMember,
    isLoading: false,
  }),
  useRemoveGroupMember: () => ({
    removeMember: mocks.removeMember,
  }),
}));

vi.mock("@/lib/hooks/use-users", () => ({
  useUsers: () => ({
    data: {
      data: [
        {
          id: "user-3",
          username: "newuser",
          display_name: "New User",
          is_active: true,
        },
      ],
      total: 1,
      page: 1,
      page_size: 100,
    },
  }),
}));

vi.mock("@/lib/hooks/use-audit-logs", () => ({
  useGroupAuditLogs: () => ({
    data: [],
    total: 0,
    isLoading: false,
    error: null,
    fetch: mocks.auditFetch,
  }),
}));

vi.mock("@/components/audit/audit-log-table", () => ({
  AuditLogTable: () => <div data-testid="audit-log-table" />,
}));

vi.mock("@/components/map/marker-shapes", () => ({
  AVAILABLE_SHAPES: ["circle", "square"],
  MARKER_SHAPES: {
    circle: { label: "Circle", path: "M0,0" },
    square: { label: "Square", path: "M0,0" },
  },
  PRESET_COLORS: ["#3b82f6", "#ef4444"],
  markerSVGString: () => '<svg data-testid="marker-svg"></svg>',
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function resetMocks() {
  mocks.group = {
    id: "group-1",
    name: "Alpha Team",
    description: "First team",
    marker_icon: "circle",
    marker_color: "#3b82f6",
    member_count: 2,
    created_by: "user-1",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  };
  mocks.groupLoading = false;
  mocks.membersLoading = false;
  mocks.members = [
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
    {
      id: "m2",
      group_id: "group-1",
      user_id: "user-2",
      username: "otheruser",
      display_name: "Other User",
      can_read: true,
      can_write: false,
      is_group_admin: false,
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
  ];
  vi.clearAllMocks();
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("GroupDetailPage", () => {
  beforeEach(resetMocks);

  it("renders the group name as a heading", () => {
    render(<GroupDetailPage />);
    expect(
      screen.getByRole("heading", { name: "Alpha Team" })
    ).toBeInTheDocument();
  });

  it("renders the group description", () => {
    render(<GroupDetailPage />);
    expect(screen.getByText("First team")).toBeInTheDocument();
  });

  it('shows "Group not found" when group is null', () => {
    mocks.group = null;
    render(<GroupDetailPage />);
    expect(screen.getByText("Group not found")).toBeInTheDocument();
  });

  it('shows back button "← Groups"', () => {
    render(<GroupDetailPage />);
    expect(
      screen.getByRole("button", { name: /Groups/ })
    ).toBeInTheDocument();
  });

  it("renders Members section with member count", () => {
    render(<GroupDetailPage />);
    expect(
      screen.getByRole("heading", { name: "Members (2)" })
    ).toBeInTheDocument();
  });

  it("shows member usernames in the table", () => {
    render(<GroupDetailPage />);
    expect(screen.getByText("testuser")).toBeInTheDocument();
    expect(screen.getByText("otheruser")).toBeInTheDocument();
  });

  it("shows Read/Write Yes/No badges for members", () => {
    render(<GroupDetailPage />);

    // There are multiple "Yes" and "No" badges across both members.
    // testuser: Read=Yes, Write=Yes
    // otheruser: Read=Yes, Write=No
    // Total: 3 "Yes" badges, 1 "No" badge in Read/Write columns
    const yesBadges = screen.getAllByText("Yes");
    const noBadges = screen.getAllByText("No");

    // testuser: Read Yes + Write Yes = 2, otheruser: Read Yes = 1 → 3 Yes total
    expect(yesBadges.length).toBe(3);
    // otheruser: Write No = 1 No total
    expect(noBadges.length).toBe(1);
  });

  it("shows Admin badge for group admin and Member badge for non-admin", () => {
    render(<GroupDetailPage />);
    expect(screen.getByText("Admin")).toBeInTheDocument();
    expect(screen.getByText("Member")).toBeInTheDocument();
  });

  it('renders "Add Member" button', () => {
    render(<GroupDetailPage />);
    expect(
      screen.getByRole("button", { name: "Add Member" })
    ).toBeInTheDocument();
  });

  it('renders "Audit Log" heading and audit table component', () => {
    render(<GroupDetailPage />);
    expect(
      screen.getByRole("heading", { name: "Audit Log" })
    ).toBeInTheDocument();
    expect(screen.getByTestId("audit-log-table")).toBeInTheDocument();
  });
});

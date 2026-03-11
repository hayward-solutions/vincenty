import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import UsersSettingsPage from "./page";
import type { User, ListResponse } from "@/types/api";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({ toast: mockToast }));

const mockUserData: ListResponse<User> = {
  data: [
    {
      id: "user-1",
      username: "alice",
      email: "alice@example.com",
      display_name: "Alice",
      avatar_url: "",
      marker_icon: "default",
      marker_color: "#000",
      is_admin: true,
      is_active: true,
      mfa_enabled: true,
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
    {
      id: "user-2",
      username: "bob",
      email: "bob@example.com",
      display_name: "Bob",
      avatar_url: "",
      marker_icon: "default",
      marker_color: "#000",
      is_admin: false,
      is_active: false,
      mfa_enabled: false,
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
  ],
  total: 2,
  page: 1,
  page_size: 20,
};

const mockUsersHook = vi.hoisted(() => ({
  data: null as ListResponse<User> | null,
  isLoading: false,
  refetch: vi.fn(),
}));

const mockCreateUserHook = vi.hoisted(() => ({
  createUser: vi.fn(),
  isLoading: false,
}));

const mockDeleteUserHook = vi.hoisted(() => ({
  deleteUser: vi.fn(),
  isLoading: false,
}));

const mockUpdateUserHook = vi.hoisted(() => ({
  updateUser: vi.fn(),
  isLoading: false,
}));

const mockAdminResetMFAHook = vi.hoisted(() => ({
  resetMFA: vi.fn(),
  isLoading: false,
}));

vi.mock("@/lib/hooks/use-users", () => ({
  useUsers: () => mockUsersHook,
  useCreateUser: () => mockCreateUserHook,
  useDeleteUser: () => mockDeleteUserHook,
  useUpdateUser: () => mockUpdateUserHook,
}));

vi.mock("@/lib/hooks/use-mfa", () => ({
  useAdminResetMFA: () => mockAdminResetMFAHook,
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => "/settings/server/users",
}));

vi.mock("next/link", () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.clearAllMocks();
  mockUsersHook.data = null;
  mockUsersHook.isLoading = false;
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("UsersSettingsPage", () => {
  it("renders the Users heading", () => {
    render(<UsersSettingsPage />);
    expect(screen.getByText("Users")).toBeInTheDocument();
  });

  it("renders Create User button", () => {
    render(<UsersSettingsPage />);
    expect(
      screen.getByRole("button", { name: /create user/i })
    ).toBeInTheDocument();
  });

  it("renders user table with data", () => {
    mockUsersHook.data = mockUserData;
    render(<UsersSettingsPage />);

    // Table headers
    expect(screen.getByText("Username")).toBeInTheDocument();
    expect(screen.getByText("Email")).toBeInTheDocument();

    // User data
    expect(screen.getByText("alice")).toBeInTheDocument();
    expect(screen.getByText("alice@example.com")).toBeInTheDocument();
    expect(screen.getByText("bob")).toBeInTheDocument();
    expect(screen.getByText("bob@example.com")).toBeInTheDocument();
  });

  it("shows Admin badge for admin users and User badge for non-admins", () => {
    mockUsersHook.data = mockUserData;
    render(<UsersSettingsPage />);
    expect(screen.getByText("Admin")).toBeInTheDocument();
    expect(screen.getByText("User")).toBeInTheDocument();
  });

  it("shows MFA status badges", () => {
    mockUsersHook.data = mockUserData;
    render(<UsersSettingsPage />);
    expect(screen.getByText("Enabled")).toBeInTheDocument();
    expect(screen.getByText("Off")).toBeInTheDocument();
  });

  it("shows Active/Inactive status", () => {
    mockUsersHook.data = mockUserData;
    render(<UsersSettingsPage />);
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Inactive")).toBeInTheDocument();
  });

  it("shows 'No users found' when data is empty", () => {
    mockUsersHook.data = { data: [], total: 0, page: 1, page_size: 20 };
    render(<UsersSettingsPage />);
    expect(screen.getByText("No users found")).toBeInTheDocument();
  });

  it("shows pagination when total exceeds page size", () => {
    mockUsersHook.data = {
      data: Array(20).fill(mockUserData.data[0]),
      total: 25,
      page: 1,
      page_size: 20,
    };
    render(<UsersSettingsPage />);
    expect(screen.getByText(/showing 1-20 of 25/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /next/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled();
  });
});

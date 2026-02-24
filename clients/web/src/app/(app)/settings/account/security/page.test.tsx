import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import SecuritySettingsPage from "./page";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
  toast: mockToast,
}));

const mockChangePasswordHook = vi.hoisted(() => ({
  changePassword: vi.fn(),
  isLoading: false,
}));

vi.mock("@/lib/hooks/use-profile", () => ({
  useChangePassword: () => mockChangePasswordHook,
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

// Mock MFAMethodList — tested separately
vi.mock("@/components/mfa/mfa-method-list", () => ({
  MFAMethodList: () => <div data-testid="mfa-method-list">MFA Methods</div>,
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => "/settings/account/security",
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
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("SecuritySettingsPage", () => {
  it("renders the Security heading", () => {
    render(<SecuritySettingsPage />);
    expect(screen.getByText("Security")).toBeInTheDocument();
  });

  it("renders password change form", () => {
    render(<SecuritySettingsPage />);
    expect(screen.getByLabelText("Current Password")).toBeInTheDocument();
    expect(screen.getByLabelText("New Password")).toBeInTheDocument();
    expect(screen.getByLabelText("Confirm New Password")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /change password/i })
    ).toBeInTheDocument();
  });

  it("renders MFAMethodList component", () => {
    render(<SecuritySettingsPage />);
    expect(screen.getByTestId("mfa-method-list")).toBeInTheDocument();
  });

  it("renders API Tokens section", () => {
    render(<SecuritySettingsPage />);
    expect(screen.getByText("API Tokens")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Validation
  // -----------------------------------------------------------------------

  it("shows validation error when current password is empty", async () => {
    const user = userEvent.setup();
    render(<SecuritySettingsPage />);

    await user.type(screen.getByLabelText("New Password"), "newpass123");
    await user.type(
      screen.getByLabelText("Confirm New Password"),
      "newpass123"
    );
    await user.click(
      screen.getByRole("button", { name: /change password/i })
    );

    expect(
      screen.getByText("Current password is required")
    ).toBeInTheDocument();
    expect(mockChangePasswordHook.changePassword).not.toHaveBeenCalled();
  });

  it("shows validation error when new password is too short", async () => {
    const user = userEvent.setup();
    render(<SecuritySettingsPage />);

    await user.type(screen.getByLabelText("Current Password"), "oldpass");
    await user.type(screen.getByLabelText("New Password"), "short");
    await user.type(screen.getByLabelText("Confirm New Password"), "short");
    await user.click(
      screen.getByRole("button", { name: /change password/i })
    );

    expect(
      screen.getByText("New password must be at least 8 characters")
    ).toBeInTheDocument();
  });

  it("shows validation error when passwords do not match", async () => {
    const user = userEvent.setup();
    render(<SecuritySettingsPage />);

    await user.type(screen.getByLabelText("Current Password"), "oldpass");
    await user.type(screen.getByLabelText("New Password"), "newpass123");
    await user.type(
      screen.getByLabelText("Confirm New Password"),
      "different1"
    );
    await user.click(
      screen.getByRole("button", { name: /change password/i })
    );

    expect(
      screen.getByText("New passwords do not match")
    ).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Successful password change
  // -----------------------------------------------------------------------

  it("calls changePassword and shows success toast", async () => {
    mockChangePasswordHook.changePassword.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<SecuritySettingsPage />);

    await user.type(screen.getByLabelText("Current Password"), "oldpass");
    await user.type(screen.getByLabelText("New Password"), "newpass123");
    await user.type(
      screen.getByLabelText("Confirm New Password"),
      "newpass123"
    );
    await user.click(
      screen.getByRole("button", { name: /change password/i })
    );

    await waitFor(() => {
      expect(mockChangePasswordHook.changePassword).toHaveBeenCalledWith({
        current_password: "oldpass",
        new_password: "newpass123",
      });
      expect(mockToast.success).toHaveBeenCalledWith(
        "Password changed successfully"
      );
    });
  });

  // -----------------------------------------------------------------------
  // Error handling
  // -----------------------------------------------------------------------

  it("shows error toast on changePassword failure", async () => {
    mockChangePasswordHook.changePassword.mockRejectedValue(
      new Error("fail")
    );
    const user = userEvent.setup();
    render(<SecuritySettingsPage />);

    await user.type(screen.getByLabelText("Current Password"), "oldpass");
    await user.type(screen.getByLabelText("New Password"), "newpass123");
    await user.type(
      screen.getByLabelText("Confirm New Password"),
      "newpass123"
    );
    await user.click(
      screen.getByRole("button", { name: /change password/i })
    );

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith(
        "Failed to change password"
      );
    });
  });
});

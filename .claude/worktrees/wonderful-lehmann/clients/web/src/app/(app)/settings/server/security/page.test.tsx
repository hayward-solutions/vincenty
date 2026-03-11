import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import ServerSecuritySettingsPage from "./page";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
  toast: mockToast,
}));

const mockServerSettingsHook = vi.hoisted(() => ({
  settings: { mfa_required: false, mapbox_access_token: "", google_maps_api_key: "" },
  isLoading: false,
  update: vi.fn(),
  refetch: vi.fn(),
}));

vi.mock("@/lib/hooks/use-mfa", () => ({
  useServerSettings: () => mockServerSettingsHook,
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
  usePathname: () => "/settings/server/security",
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
  mockServerSettingsHook.settings = {
    mfa_required: false,
    mapbox_access_token: "",
    google_maps_api_key: "",
  };
  mockServerSettingsHook.isLoading = false;
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("ServerSecuritySettingsPage", () => {
  it("renders the Server Security heading", () => {
    render(<ServerSecuritySettingsPage />);
    expect(screen.getByText("Server Security")).toBeInTheDocument();
  });

  it("renders MFA policy card", () => {
    render(<ServerSecuritySettingsPage />);
    expect(
      screen.getByText("Multi-Factor Authentication Policy")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/require all users to configure mfa/i)
    ).toBeInTheDocument();
  });

  it("shows Enable button when MFA is not required", () => {
    render(<ServerSecuritySettingsPage />);
    expect(
      screen.getByRole("button", { name: /enable/i })
    ).toBeInTheDocument();
  });

  it("shows Disable button and warning when MFA is required", () => {
    mockServerSettingsHook.settings = {
      ...mockServerSettingsHook.settings,
      mfa_required: true,
    };
    render(<ServerSecuritySettingsPage />);
    expect(
      screen.getByRole("button", { name: /disable/i })
    ).toBeInTheDocument();
    expect(
      screen.getByText(/mfa is currently required for all users/i)
    ).toBeInTheDocument();
  });

  it("calls update and shows success toast when enabling MFA", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    mockServerSettingsHook.update.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<ServerSecuritySettingsPage />);

    await user.click(screen.getByRole("button", { name: /enable/i }));

    await waitFor(() => {
      expect(mockServerSettingsHook.update).toHaveBeenCalledWith({
        mfa_required: true,
      });
      expect(mockToast.success).toHaveBeenCalledWith(
        "MFA is now required for all users"
      );
    });
  });

  it("does not enable MFA when user cancels confirm dialog", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    const user = userEvent.setup();
    render(<ServerSecuritySettingsPage />);

    await user.click(screen.getByRole("button", { name: /enable/i }));

    expect(mockServerSettingsHook.update).not.toHaveBeenCalled();
  });

  it("calls update and shows success toast when disabling MFA", async () => {
    mockServerSettingsHook.settings = {
      ...mockServerSettingsHook.settings,
      mfa_required: true,
    };
    mockServerSettingsHook.update.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<ServerSecuritySettingsPage />);

    // No confirm dialog shown for disable
    await user.click(screen.getByRole("button", { name: /disable/i }));

    await waitFor(() => {
      expect(mockServerSettingsHook.update).toHaveBeenCalledWith({
        mfa_required: false,
      });
      expect(mockToast.success).toHaveBeenCalledWith(
        "MFA requirement removed"
      );
    });
  });

  it("shows error toast on update failure", async () => {
    mockServerSettingsHook.settings = {
      ...mockServerSettingsHook.settings,
      mfa_required: true,
    };
    mockServerSettingsHook.update.mockRejectedValue(new Error("fail"));
    const user = userEvent.setup();
    render(<ServerSecuritySettingsPage />);

    await user.click(screen.getByRole("button", { name: /disable/i }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith(
        "Failed to update settings"
      );
    });
  });
});

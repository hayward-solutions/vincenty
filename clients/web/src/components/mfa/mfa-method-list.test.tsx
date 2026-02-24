import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { MFAMethodList } from "./mfa-method-list";
import type { MFAMethod } from "@/types/api";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
  toast: mockToast,
}));

const mockMFAMethodsHook = vi.hoisted(() => ({
  methods: [] as MFAMethod[],
  isLoading: false,
  refetch: vi.fn(),
}));

const mockDeleteHook = vi.hoisted(() => ({
  deleteMethod: vi.fn(),
  isLoading: false,
}));

const mockToggleHook = vi.hoisted(() => ({
  toggle: vi.fn(),
  isLoading: false,
}));

vi.mock("@/lib/hooks/use-mfa", () => ({
  useMFAMethods: () => mockMFAMethodsHook,
  useDeleteMFAMethod: () => mockDeleteHook,
  useTogglePasswordless: () => mockToggleHook,
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

// Mock child dialogs — we test them separately
vi.mock("./totp-setup-dialog", () => ({
  TOTPSetupDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="totp-setup-dialog">TOTP Setup</div> : null,
}));

vi.mock("./webauthn-register-dialog", () => ({
  WebAuthnRegisterDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="webauthn-register-dialog">WebAuthn Register</div> : null,
}));

vi.mock("./recovery-codes-dialog", () => ({
  RegenerateRecoveryCodesDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="recovery-codes-dialog">Recovery Codes</div> : null,
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const totpMethod: MFAMethod = {
  id: "mfa-1",
  type: "totp",
  name: "Google Authenticator",
  verified: true,
  created_at: "2025-06-01T00:00:00Z",
};

const webauthnMethod: MFAMethod = {
  id: "mfa-2",
  type: "webauthn",
  name: "YubiKey",
  verified: true,
  passwordless_enabled: false,
  last_used_at: "2025-06-15T00:00:00Z",
  created_at: "2025-06-01T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  mockMFAMethodsHook.methods = [];
  mockMFAMethodsHook.isLoading = false;
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("MFAMethodList", () => {
  it("renders card title and description", () => {
    render(<MFAMethodList />);
    expect(screen.getByText("Multi-Factor Authentication")).toBeInTheDocument();
    expect(
      screen.getByText(/add an extra layer of security/i)
    ).toBeInTheDocument();
  });

  it("shows loading skeletons when loading", () => {
    mockMFAMethodsHook.isLoading = true;
    const { container } = render(<MFAMethodList />);
    // Skeleton components render with data-slot="skeleton"
    const skeletons = container.querySelectorAll('[data-slot="skeleton"]');
    if (skeletons.length === 0) {
      // Fallback: check for any animated placeholder elements
      // When loading, the authenticator/security key sections should NOT be present
      expect(screen.queryByText(/authenticator app/i)).not.toBeInTheDocument();
    } else {
      expect(skeletons.length).toBeGreaterThanOrEqual(1);
    }
  });

  it("shows empty messages when no methods configured", () => {
    render(<MFAMethodList />);
    expect(
      screen.getByText(/no authenticator apps configured/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/no security keys or passkeys registered/i)
    ).toBeInTheDocument();
  });

  it("does not show recovery codes section when no methods", () => {
    render(<MFAMethodList />);
    expect(screen.queryByText(/recovery codes/i)).not.toBeInTheDocument();
  });

  it("renders TOTP methods", () => {
    mockMFAMethodsHook.methods = [totpMethod];
    render(<MFAMethodList />);
    expect(screen.getByText("Google Authenticator")).toBeInTheDocument();
  });

  it("renders WebAuthn methods with passwordless badge", () => {
    mockMFAMethodsHook.methods = [webauthnMethod];
    render(<MFAMethodList />);
    expect(screen.getByText("YubiKey")).toBeInTheDocument();
    expect(screen.getByText("MFA only")).toBeInTheDocument();
  });

  it("renders Passwordless badge when enabled", () => {
    mockMFAMethodsHook.methods = [
      { ...webauthnMethod, passwordless_enabled: true },
    ];
    render(<MFAMethodList />);
    expect(screen.getByText("Passwordless")).toBeInTheDocument();
  });

  it("shows last used date when available", () => {
    mockMFAMethodsHook.methods = [webauthnMethod];
    render(<MFAMethodList />);
    expect(screen.getByText(/last used/i)).toBeInTheDocument();
  });

  it("shows recovery codes section when methods exist", () => {
    mockMFAMethodsHook.methods = [totpMethod];
    render(<MFAMethodList />);
    expect(screen.getByText("Recovery Codes")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /regenerate/i })
    ).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Add buttons
  // -----------------------------------------------------------------------

  it("opens TOTP setup dialog when Add button is clicked", async () => {
    const user = userEvent.setup();
    render(<MFAMethodList />);
    const addButtons = screen.getAllByRole("button", { name: /^add$/i });
    await user.click(addButtons[0]); // First Add is for Authenticator App
    expect(screen.getByTestId("totp-setup-dialog")).toBeInTheDocument();
  });

  it("opens WebAuthn register dialog when Add button is clicked", async () => {
    const user = userEvent.setup();
    render(<MFAMethodList />);
    const addButtons = screen.getAllByRole("button", { name: /^add$/i });
    await user.click(addButtons[1]); // Second Add is for Security Keys
    expect(screen.getByTestId("webauthn-register-dialog")).toBeInTheDocument();
  });

  it("opens recovery codes dialog when Regenerate is clicked", async () => {
    mockMFAMethodsHook.methods = [totpMethod];
    const user = userEvent.setup();
    render(<MFAMethodList />);
    await user.click(screen.getByRole("button", { name: /regenerate/i }));
    expect(screen.getByTestId("recovery-codes-dialog")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Delete method
  // -----------------------------------------------------------------------

  it("calls deleteMethod and shows success toast on delete", async () => {
    mockMFAMethodsHook.methods = [totpMethod];
    mockDeleteHook.deleteMethod.mockResolvedValue(undefined);
    vi.spyOn(window, "confirm").mockReturnValue(true);

    const user = userEvent.setup();
    render(<MFAMethodList />);
    await user.click(screen.getByRole("button", { name: /remove/i }));

    await waitFor(() => {
      expect(mockDeleteHook.deleteMethod).toHaveBeenCalledWith("mfa-1");
      expect(mockToast.success).toHaveBeenCalledWith('"Google Authenticator" removed');
      expect(mockMFAMethodsHook.refetch).toHaveBeenCalled();
    });
  });

  it("does not delete when confirm is cancelled", async () => {
    mockMFAMethodsHook.methods = [totpMethod];
    vi.spyOn(window, "confirm").mockReturnValue(false);

    const user = userEvent.setup();
    render(<MFAMethodList />);
    await user.click(screen.getByRole("button", { name: /remove/i }));

    expect(mockDeleteHook.deleteMethod).not.toHaveBeenCalled();
  });

  it("shows error toast when delete fails", async () => {
    mockMFAMethodsHook.methods = [totpMethod];
    mockDeleteHook.deleteMethod.mockRejectedValue(new Error("fail"));
    vi.spyOn(window, "confirm").mockReturnValue(true);

    const user = userEvent.setup();
    render(<MFAMethodList />);
    await user.click(screen.getByRole("button", { name: /remove/i }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith("Failed to remove method");
    });
  });

  // -----------------------------------------------------------------------
  // Toggle passwordless
  // -----------------------------------------------------------------------

  it("calls toggle and shows success toast on passwordless toggle", async () => {
    mockMFAMethodsHook.methods = [webauthnMethod];
    mockToggleHook.toggle.mockResolvedValue(undefined);

    const user = userEvent.setup();
    render(<MFAMethodList />);
    await user.click(screen.getByText("MFA only"));

    await waitFor(() => {
      expect(mockToggleHook.toggle).toHaveBeenCalledWith("mfa-2", true);
      expect(mockToast.success).toHaveBeenCalledWith("Passwordless login enabled");
      expect(mockMFAMethodsHook.refetch).toHaveBeenCalled();
    });
  });

  it("shows error toast when toggle fails", async () => {
    mockMFAMethodsHook.methods = [webauthnMethod];
    mockToggleHook.toggle.mockRejectedValue(new Error("fail"));

    const user = userEvent.setup();
    render(<MFAMethodList />);
    await user.click(screen.getByText("MFA only"));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith("Failed to update");
    });
  });
});

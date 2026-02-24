import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { TOTPSetupDialog } from "./totp-setup-dialog";
import type { TOTPSetupResponse, TOTPVerifyResponse } from "@/types/api";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
  toast: mockToast,
}));

const mockTOTPSetupHook = vi.hoisted(() => ({
  beginSetup: vi.fn(),
  verifySetup: vi.fn(),
  isLoading: false,
}));

vi.mock("@/lib/hooks/use-mfa", () => ({
  useTOTPSetup: () => mockTOTPSetupHook,
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

// Mock QRCodeSVG — renders a simple element
vi.mock("qrcode.react", () => ({
  QRCodeSVG: ({ value }: { value: string }) => (
    <div data-testid="qrcode" data-value={value} />
  ),
}));

// Mock RecoveryCodesDisplay
vi.mock("./recovery-codes-dialog", () => ({
  RecoveryCodesDisplay: ({
    codes,
    onDone,
  }: {
    codes: string[];
    onDone: () => void;
  }) => (
    <div data-testid="recovery-codes">
      <span>{codes.join(",")}</span>
      <button onClick={onDone}>Done</button>
    </div>
  ),
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const mockSetup: TOTPSetupResponse = {
  method_id: "mfa-1",
  secret: "JBSWY3DPEHPK3PXP",
  uri: "otpauth://totp/SitAware:testuser?secret=JBSWY3DPEHPK3PXP&issuer=SitAware",
  issuer: "SitAware",
  account: "testuser",
};

function renderDialog(
  overrides: Partial<{
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onComplete: () => void;
  }> = {}
) {
  const props = {
    open: true,
    onOpenChange: vi.fn(),
    onComplete: vi.fn(),
    ...overrides,
  };
  return { ...render(<TOTPSetupDialog {...props} />), props };
}

beforeEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("TOTPSetupDialog", () => {
  it("does not render content when closed", () => {
    renderDialog({ open: false });
    expect(
      screen.queryByText("Set Up Authenticator App")
    ).not.toBeInTheDocument();
  });

  it("shows the name step with default name", () => {
    renderDialog();
    expect(screen.getByText("Set Up Authenticator App")).toBeInTheDocument();
    expect(screen.getByLabelText(/device name/i)).toHaveValue(
      "Authenticator App"
    );
  });

  it("has Continue and Cancel buttons on name step", () => {
    renderDialog();
    expect(
      screen.getByRole("button", { name: /continue/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /cancel/i })
    ).toBeInTheDocument();
  });

  it("Cancel button closes the dialog", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByRole("button", { name: /cancel/i }));
    expect(props.onOpenChange).toHaveBeenCalledWith(false);
  });

  // -----------------------------------------------------------------------
  // Name → Scan step
  // -----------------------------------------------------------------------

  it("transitions to scan step after beginSetup succeeds", async () => {
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));

    await waitFor(() => {
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument();
    });

    expect(screen.getByTestId("qrcode")).toHaveAttribute(
      "data-value",
      mockSetup.uri
    );
    expect(screen.getByText(mockSetup.secret)).toBeInTheDocument();
  });

  it("shows error toast when beginSetup fails", async () => {
    mockTOTPSetupHook.beginSetup.mockRejectedValue(new Error("fail"));
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith(
        "Failed to begin TOTP setup"
      );
    });
  });

  // -----------------------------------------------------------------------
  // Scan → Verify step
  // -----------------------------------------------------------------------

  it("transitions to verify step", async () => {
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument()
    );

    await user.click(
      screen.getByRole("button", { name: /i've scanned it/i })
    );
    expect(screen.getByText("Verify Code")).toBeInTheDocument();
    expect(screen.getByLabelText(/verification code/i)).toBeInTheDocument();
  });

  it("has Back button on scan step", async () => {
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument()
    );

    await user.click(screen.getByRole("button", { name: /back/i }));
    expect(screen.getByText("Set Up Authenticator App")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Verify step
  // -----------------------------------------------------------------------

  it("strips non-digit characters from verify input", async () => {
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument()
    );
    await user.click(
      screen.getByRole("button", { name: /i've scanned it/i })
    );

    await user.type(screen.getByLabelText(/verification code/i), "12ab34");
    expect(screen.getByLabelText(/verification code/i)).toHaveValue("1234");
  });

  it("disables Verify button when code is less than 6 digits", async () => {
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument()
    );
    await user.click(
      screen.getByRole("button", { name: /i've scanned it/i })
    );

    expect(screen.getByRole("button", { name: /verify/i })).toBeDisabled();
  });

  it("calls verifySetup and shows recovery codes when returned", async () => {
    const verifyResp: TOTPVerifyResponse = {
      verified: true,
      recovery_codes: ["code-1", "code-2"],
    };
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    mockTOTPSetupHook.verifySetup.mockResolvedValue(verifyResp);
    const user = userEvent.setup();
    renderDialog();

    // Go through name → scan → verify
    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument()
    );
    await user.click(
      screen.getByRole("button", { name: /i've scanned it/i })
    );
    await user.type(screen.getByLabelText(/verification code/i), "123456");
    await user.click(screen.getByRole("button", { name: /verify/i }));

    await waitFor(() => {
      expect(mockTOTPSetupHook.verifySetup).toHaveBeenCalledWith(
        "mfa-1",
        "123456"
      );
      expect(screen.getByTestId("recovery-codes")).toBeInTheDocument();
      expect(screen.getByText("Save Recovery Codes")).toBeInTheDocument();
    });
  });

  it("completes without recovery codes step when none returned", async () => {
    const verifyResp: TOTPVerifyResponse = {
      verified: true,
    };
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    mockTOTPSetupHook.verifySetup.mockResolvedValue(verifyResp);
    const user = userEvent.setup();
    const { props } = renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument()
    );
    await user.click(
      screen.getByRole("button", { name: /i've scanned it/i })
    );
    await user.type(screen.getByLabelText(/verification code/i), "123456");
    await user.click(screen.getByRole("button", { name: /verify/i }));

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith(
        "Authenticator app configured"
      );
      expect(props.onOpenChange).toHaveBeenCalledWith(false);
      expect(props.onComplete).toHaveBeenCalled();
    });
  });

  it("shows error toast when verifySetup fails", async () => {
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    mockTOTPSetupHook.verifySetup.mockRejectedValue(new Error("bad code"));
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument()
    );
    await user.click(
      screen.getByRole("button", { name: /i've scanned it/i })
    );
    await user.type(screen.getByLabelText(/verification code/i), "123456");
    await user.click(screen.getByRole("button", { name: /verify/i }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith("Invalid code");
    });
  });

  // -----------------------------------------------------------------------
  // Recovery done
  // -----------------------------------------------------------------------

  it("closes dialog and calls onComplete when recovery codes Done is clicked", async () => {
    const verifyResp: TOTPVerifyResponse = {
      verified: true,
      recovery_codes: ["code-1"],
    };
    mockTOTPSetupHook.beginSetup.mockResolvedValue(mockSetup);
    mockTOTPSetupHook.verifySetup.mockResolvedValue(verifyResp);
    const user = userEvent.setup();
    const { props } = renderDialog();

    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(screen.getByText("Scan QR Code")).toBeInTheDocument()
    );
    await user.click(
      screen.getByRole("button", { name: /i've scanned it/i })
    );
    await user.type(screen.getByLabelText(/verification code/i), "123456");
    await user.click(screen.getByRole("button", { name: /verify/i }));

    await waitFor(() =>
      expect(screen.getByTestId("recovery-codes")).toBeInTheDocument()
    );

    // Click the mocked Done button inside RecoveryCodesDisplay
    await user.click(screen.getByRole("button", { name: /done/i }));

    expect(mockToast.success).toHaveBeenCalledWith(
      "Authenticator app configured"
    );
    expect(props.onOpenChange).toHaveBeenCalledWith(false);
    expect(props.onComplete).toHaveBeenCalled();
  });
});

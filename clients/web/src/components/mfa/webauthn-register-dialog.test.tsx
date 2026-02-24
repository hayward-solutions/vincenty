import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { WebAuthnRegisterDialog } from "./webauthn-register-dialog";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
  toast: mockToast,
}));

const mockWebAuthnHook = vi.hoisted(() => ({
  beginRegister: vi.fn(),
  finishRegister: vi.fn(),
  isLoading: false,
}));

vi.mock("@/lib/hooks/use-mfa", () => ({
  useWebAuthnRegister: () => mockWebAuthnHook,
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
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
  return { ...render(<WebAuthnRegisterDialog {...props} />), props };
}

function setupCredentialsMock(
  credential: PublicKeyCredential | null,
  shouldReject?: Error
) {
  const create = shouldReject
    ? vi.fn().mockRejectedValue(shouldReject)
    : vi.fn().mockResolvedValue(credential);
  Object.defineProperty(navigator, "credentials", {
    value: { create },
    configurable: true,
  });
  return create;
}

function makeServerResponse() {
  return {
    publicKey: {
      challenge: "dGVzdA", // base64url of "test"
      rp: { name: "SitAware" },
      user: { id: "dXNlcg", name: "testuser", displayName: "Test User" },
      pubKeyCredParams: [],
      excludeCredentials: [],
    },
  };
}

function makeCredential(): PublicKeyCredential {
  const attestationResponse = {
    attestationObject: new ArrayBuffer(1),
    clientDataJSON: new ArrayBuffer(1),
  };
  return {
    id: "cred-id",
    rawId: new ArrayBuffer(1),
    type: "public-key",
    response: attestationResponse,
    getClientExtensionResults: () => ({}),
    authenticatorAttachment: null,
  } as unknown as PublicKeyCredential;
}

beforeEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("WebAuthnRegisterDialog", () => {
  it("does not render content when closed", () => {
    renderDialog({ open: false });
    expect(
      screen.queryByText("Register Security Key")
    ).not.toBeInTheDocument();
  });

  it("shows name step with default name", () => {
    renderDialog();
    expect(screen.getByText("Register Security Key")).toBeInTheDocument();
    expect(screen.getByLabelText(/credential name/i)).toHaveValue(
      "Security Key"
    );
  });

  it("has Register and Cancel buttons", () => {
    renderDialog();
    expect(
      screen.getByRole("button", { name: /register/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /cancel/i })
    ).toBeInTheDocument();
  });

  it("Cancel closes the dialog", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog();
    await user.click(screen.getByRole("button", { name: /cancel/i }));
    expect(props.onOpenChange).toHaveBeenCalledWith(false);
  });

  it("allows changing credential name", async () => {
    const user = userEvent.setup();
    renderDialog();
    const input = screen.getByLabelText(/credential name/i);
    await user.clear(input);
    await user.type(input, "YubiKey 5");
    expect(input).toHaveValue("YubiKey 5");
  });

  // -----------------------------------------------------------------------
  // Successful registration without recovery codes
  // -----------------------------------------------------------------------

  it("registers successfully and closes when no recovery codes", async () => {
    mockWebAuthnHook.beginRegister.mockResolvedValue(makeServerResponse());
    setupCredentialsMock(makeCredential());
    mockWebAuthnHook.finishRegister.mockResolvedValue({ registered: true });

    const user = userEvent.setup();
    const { props } = renderDialog();

    await user.click(screen.getByRole("button", { name: /register/i }));

    // Should show waiting state briefly
    await waitFor(() => {
      expect(mockWebAuthnHook.beginRegister).toHaveBeenCalledWith(
        "Security Key"
      );
    });

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith(
        "Security key registered"
      );
      expect(props.onOpenChange).toHaveBeenCalledWith(false);
      expect(props.onComplete).toHaveBeenCalled();
    });
  });

  // -----------------------------------------------------------------------
  // Registration with recovery codes
  // -----------------------------------------------------------------------

  it("shows recovery codes when returned", async () => {
    mockWebAuthnHook.beginRegister.mockResolvedValue(makeServerResponse());
    setupCredentialsMock(makeCredential());
    mockWebAuthnHook.finishRegister.mockResolvedValue({
      registered: true,
      recovery_codes: ["rc-1", "rc-2"],
    });

    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /register/i }));

    await waitFor(() => {
      expect(screen.getByTestId("recovery-codes")).toBeInTheDocument();
      expect(screen.getByText("rc-1,rc-2")).toBeInTheDocument();
      expect(screen.getByText("Save Recovery Codes")).toBeInTheDocument();
    });
  });

  it("completes after recovery codes Done is clicked", async () => {
    mockWebAuthnHook.beginRegister.mockResolvedValue(makeServerResponse());
    setupCredentialsMock(makeCredential());
    mockWebAuthnHook.finishRegister.mockResolvedValue({
      registered: true,
      recovery_codes: ["rc-1"],
    });

    const user = userEvent.setup();
    const { props } = renderDialog();

    await user.click(screen.getByRole("button", { name: /register/i }));

    await waitFor(() =>
      expect(screen.getByTestId("recovery-codes")).toBeInTheDocument()
    );

    await user.click(screen.getByRole("button", { name: /done/i }));

    expect(mockToast.success).toHaveBeenCalledWith(
      "Security key registered"
    );
    expect(props.onOpenChange).toHaveBeenCalledWith(false);
    expect(props.onComplete).toHaveBeenCalled();
  });

  // -----------------------------------------------------------------------
  // Error handling
  // -----------------------------------------------------------------------

  it("shows error toast when credentials.create returns null", async () => {
    mockWebAuthnHook.beginRegister.mockResolvedValue(makeServerResponse());
    setupCredentialsMock(null);

    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /register/i }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith(
        "Failed to register security key"
      );
    });
    // Should return to name step
    expect(screen.getByText("Register Security Key")).toBeInTheDocument();
  });

  it("shows cancellation toast on NotAllowedError", async () => {
    mockWebAuthnHook.beginRegister.mockResolvedValue(makeServerResponse());
    const domErr = new DOMException("User cancelled", "NotAllowedError");
    setupCredentialsMock(null, domErr);

    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /register/i }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith(
        "Registration was cancelled"
      );
    });
  });

  it("shows ApiError message on API failure", async () => {
    const { ApiError } = await import("@/lib/api");
    mockWebAuthnHook.beginRegister.mockRejectedValue(
      new ApiError("Server error")
    );

    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /register/i }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith("Server error");
    });
  });

  it("shows generic error for unexpected errors", async () => {
    mockWebAuthnHook.beginRegister.mockRejectedValue(
      new Error("network failure")
    );

    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /register/i }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith(
        "Failed to register security key"
      );
    });
  });
});

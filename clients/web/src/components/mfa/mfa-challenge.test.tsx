import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { MFAChallenge } from "./mfa-challenge";
import type { MFAChallengeResponse, AuthResponse } from "@/types/api";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const mockMFAChallenge = {
  verifyTOTP: vi.fn(),
  verifyRecovery: vi.fn(),
  beginWebAuthn: vi.fn(),
  finishWebAuthn: vi.fn(),
  isLoading: false,
};

vi.mock("@/lib/hooks/use-mfa", () => ({
  useMFAChallenge: () => mockMFAChallenge,
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const authResp: AuthResponse = {
  access_token: "tok",
  refresh_token: "ref",
  user: {
    id: "user-1",
    username: "u",
    email: "u@e.com",
    display_name: "U",
    avatar_url: "",
    marker_icon: "default",
    marker_color: "#000",
    is_admin: false,
    is_active: true,
    mfa_enabled: true,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
};

function makeChallenge(
  methods: string[] = ["totp", "webauthn", "recovery"]
): MFAChallengeResponse {
  return { mfa_required: true, mfa_token: "mfa-token-123", methods };
}

function renderChallenge(
  overrides: Partial<{
    challenge: MFAChallengeResponse;
    onSuccess: (r: AuthResponse) => void;
    onCancel: () => void;
  }> = {}
) {
  const props = {
    challenge: makeChallenge(),
    onSuccess: vi.fn(),
    onCancel: vi.fn(),
    ...overrides,
  };
  return { ...render(<MFAChallenge {...props} />), props };
}

beforeEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("MFAChallenge", () => {
  it("renders the title", () => {
    renderChallenge();
    expect(screen.getByText("Two-Factor Authentication")).toBeInTheDocument();
  });

  it("renders method selector buttons when multiple methods", () => {
    renderChallenge();
    expect(screen.getByRole("button", { name: /authenticator/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /security key/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /recovery/i })).toBeInTheDocument();
  });

  it("does not render method selector for single method", () => {
    renderChallenge({ challenge: makeChallenge(["totp"]) });
    // Should have the TOTP form but no selector buttons (Authenticator in the selector)
    expect(screen.getByLabelText(/authentication code/i)).toBeInTheDocument();
    // The method selector has a container div with flex gap-1 — just check there's no Security Key button
    expect(screen.queryByRole("button", { name: /security key/i })).not.toBeInTheDocument();
  });

  it("shows Cancel and go back button", async () => {
    const user = userEvent.setup();
    const { props } = renderChallenge();
    await user.click(screen.getByRole("button", { name: /cancel and go back/i }));
    expect(props.onCancel).toHaveBeenCalledTimes(1);
  });

  // -----------------------------------------------------------------------
  // TOTP
  // -----------------------------------------------------------------------

  describe("TOTP flow", () => {
    it("shows TOTP form by default when first method is totp", () => {
      renderChallenge();
      expect(screen.getByLabelText(/authentication code/i)).toBeInTheDocument();
      expect(screen.getByPlaceholderText("000000")).toBeInTheDocument();
    });

    it("strips non-digit characters from TOTP input", async () => {
      const user = userEvent.setup();
      renderChallenge();
      const input = screen.getByLabelText(/authentication code/i);
      await user.type(input, "12ab34");
      expect(input).toHaveValue("1234");
    });

    it("disables Verify button when code is less than 6 digits", () => {
      renderChallenge();
      const verifyBtn = screen.getByRole("button", { name: /verify/i });
      expect(verifyBtn).toBeDisabled();
    });

    it("calls verifyTOTP on submit with correct args", async () => {
      mockMFAChallenge.verifyTOTP.mockResolvedValue(authResp);
      const user = userEvent.setup();
      const { props } = renderChallenge();

      await user.type(screen.getByLabelText(/authentication code/i), "123456");
      await user.click(screen.getByRole("button", { name: /verify/i }));

      await waitFor(() => {
        expect(mockMFAChallenge.verifyTOTP).toHaveBeenCalledWith("mfa-token-123", "123456");
        expect(props.onSuccess).toHaveBeenCalledWith(authResp);
      });
    });

    it("shows error on TOTP verification failure", async () => {
      mockMFAChallenge.verifyTOTP.mockRejectedValue(new Error("bad"));
      const user = userEvent.setup();
      renderChallenge();

      await user.type(screen.getByLabelText(/authentication code/i), "123456");
      await user.click(screen.getByRole("button", { name: /verify/i }));

      await waitFor(() => {
        expect(screen.getByText("Verification failed")).toBeInTheDocument();
      });
    });

    it("shows ApiError message on TOTP failure", async () => {
      const { ApiError } = await import("@/lib/api");
      mockMFAChallenge.verifyTOTP.mockRejectedValue(new ApiError("Invalid TOTP"));
      const user = userEvent.setup();
      renderChallenge();

      await user.type(screen.getByLabelText(/authentication code/i), "123456");
      await user.click(screen.getByRole("button", { name: /verify/i }));

      await waitFor(() => {
        expect(screen.getByText("Invalid TOTP")).toBeInTheDocument();
      });
    });
  });

  // -----------------------------------------------------------------------
  // Recovery
  // -----------------------------------------------------------------------

  describe("Recovery flow", () => {
    it("shows recovery form when Recovery tab is clicked", async () => {
      const user = userEvent.setup();
      renderChallenge();
      await user.click(screen.getByRole("button", { name: /recovery/i }));
      expect(screen.getByLabelText(/recovery code/i)).toBeInTheDocument();
      expect(screen.getByPlaceholderText("xxxx-xxxx")).toBeInTheDocument();
    });

    it("calls verifyRecovery on submit", async () => {
      mockMFAChallenge.verifyRecovery.mockResolvedValue(authResp);
      const user = userEvent.setup();
      const { props } = renderChallenge();

      await user.click(screen.getByRole("button", { name: /recovery/i }));
      await user.type(screen.getByLabelText(/recovery code/i), "aaaa-bbbb");
      await user.click(screen.getByRole("button", { name: /verify/i }));

      await waitFor(() => {
        expect(mockMFAChallenge.verifyRecovery).toHaveBeenCalledWith("mfa-token-123", "aaaa-bbbb");
        expect(props.onSuccess).toHaveBeenCalledWith(authResp);
      });
    });

    it("shows error on recovery failure", async () => {
      mockMFAChallenge.verifyRecovery.mockRejectedValue(new Error("bad"));
      const user = userEvent.setup();
      renderChallenge();

      await user.click(screen.getByRole("button", { name: /recovery/i }));
      await user.type(screen.getByLabelText(/recovery code/i), "aaaa-bbbb");
      await user.click(screen.getByRole("button", { name: /verify/i }));

      await waitFor(() => {
        expect(screen.getByText("Invalid recovery code")).toBeInTheDocument();
      });
    });
  });

  // -----------------------------------------------------------------------
  // WebAuthn
  // -----------------------------------------------------------------------

  describe("WebAuthn flow", () => {
    it("shows WebAuthn UI when Security Key tab is clicked", async () => {
      const user = userEvent.setup();
      renderChallenge();
      await user.click(screen.getByRole("button", { name: /security key/i }));
      expect(
        screen.getByRole("button", { name: /use security key/i })
      ).toBeInTheDocument();
    });

    it("calls beginWebAuthn, navigator.credentials.get, and finishWebAuthn", async () => {
      // Set up the mock chain
      const mockAssertionResponse = {
        authenticatorData: new ArrayBuffer(1),
        clientDataJSON: new ArrayBuffer(1),
        signature: new ArrayBuffer(1),
        userHandle: new ArrayBuffer(1),
      };
      const mockCredential = {
        id: "cred-id",
        rawId: new ArrayBuffer(1),
        type: "public-key",
        response: mockAssertionResponse,
      };

      mockMFAChallenge.beginWebAuthn.mockResolvedValue({
        options: {
          publicKey: {
            challenge: "dGVzdA", // base64url of "test"
            allowCredentials: [],
          },
        },
        mfa_token: "mfa-token-123",
      });

      Object.defineProperty(navigator, "credentials", {
        value: { get: vi.fn().mockResolvedValue(mockCredential) },
        configurable: true,
      });

      mockMFAChallenge.finishWebAuthn.mockResolvedValue(authResp);

      const user = userEvent.setup();
      const { props } = renderChallenge();
      await user.click(screen.getByRole("button", { name: /security key/i }));
      await user.click(screen.getByRole("button", { name: /use security key/i }));

      await waitFor(() => {
        expect(mockMFAChallenge.beginWebAuthn).toHaveBeenCalledWith("mfa-token-123");
        expect(navigator.credentials.get).toHaveBeenCalled();
        expect(mockMFAChallenge.finishWebAuthn).toHaveBeenCalled();
        expect(props.onSuccess).toHaveBeenCalledWith(authResp);
      });
    });

    it("shows error when navigator.credentials.get returns null", async () => {
      mockMFAChallenge.beginWebAuthn.mockResolvedValue({
        options: {
          publicKey: {
            challenge: "dGVzdA",
            allowCredentials: [],
          },
        },
        mfa_token: "mfa-token-123",
      });

      Object.defineProperty(navigator, "credentials", {
        value: { get: vi.fn().mockResolvedValue(null) },
        configurable: true,
      });

      const user = userEvent.setup();
      renderChallenge();
      await user.click(screen.getByRole("button", { name: /security key/i }));
      await user.click(screen.getByRole("button", { name: /use security key/i }));

      await waitFor(() => {
        expect(screen.getByText("No assertion returned")).toBeInTheDocument();
      });
    });

    it("shows cancellation error on NotAllowedError", async () => {
      mockMFAChallenge.beginWebAuthn.mockResolvedValue({
        options: {
          publicKey: { challenge: "dGVzdA", allowCredentials: [] },
        },
        mfa_token: "mfa-token-123",
      });

      const domErr = new DOMException("User cancelled", "NotAllowedError");
      Object.defineProperty(navigator, "credentials", {
        value: { get: vi.fn().mockRejectedValue(domErr) },
        configurable: true,
      });

      const user = userEvent.setup();
      renderChallenge();
      await user.click(screen.getByRole("button", { name: /security key/i }));
      await user.click(screen.getByRole("button", { name: /use security key/i }));

      await waitFor(() => {
        expect(screen.getByText("Authentication was cancelled")).toBeInTheDocument();
      });
    });
  });

  // -----------------------------------------------------------------------
  // Method switching
  // -----------------------------------------------------------------------

  describe("method switching", () => {
    it("clears code and error when switching methods", async () => {
      mockMFAChallenge.verifyTOTP.mockRejectedValue(new Error("bad"));
      const user = userEvent.setup();
      renderChallenge();

      // Type code and trigger error
      await user.type(screen.getByLabelText(/authentication code/i), "123456");
      await user.click(screen.getByRole("button", { name: /verify/i }));
      await waitFor(() => {
        expect(screen.getByText("Verification failed")).toBeInTheDocument();
      });

      // Switch to recovery
      await user.click(screen.getByRole("button", { name: /recovery/i }));

      // Error should be gone and input should be empty
      expect(screen.queryByText("Verification failed")).not.toBeInTheDocument();
      expect(screen.getByLabelText(/recovery code/i)).toHaveValue("");
    });
  });
});

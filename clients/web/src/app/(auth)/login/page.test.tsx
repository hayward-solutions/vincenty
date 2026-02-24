import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render, mockAuth } from "@/test/test-utils";
import LoginPage from "./page";
import type { MFAChallengeResponse } from "@/types/api";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockRouterPush = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockRouterPush }),
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

// Mock MFAChallenge component — we test it separately
vi.mock("@/components/mfa/mfa-challenge", () => ({
  MFAChallenge: ({
    onSuccess,
    onCancel,
  }: {
    onSuccess: (r: unknown) => void;
    onCancel: () => void;
  }) => (
    <div data-testid="mfa-challenge">
      <button onClick={() => onSuccess({ access_token: "t", refresh_token: "r", user: {} })}>
        MFA Success
      </button>
      <button onClick={onCancel}>MFA Cancel</button>
    </div>
  ),
}));

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.clearAllMocks();
  mockAuth.login.mockReset();
  mockAuth.completeMFALogin.mockReset();
  mockAuth.passkeyLogin.mockReset();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("LoginPage", () => {
  it("renders SitAware title and sign in subtitle", () => {
    render(<LoginPage />);
    expect(screen.getByText("SitAware")).toBeInTheDocument();
    expect(screen.getByText("Sign in to continue")).toBeInTheDocument();
  });

  it("renders username and password fields", () => {
    render(<LoginPage />);
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
  });

  it("renders Sign in and Passkey buttons", () => {
    render(<LoginPage />);
    expect(
      screen.getByRole("button", { name: /sign in$/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /sign in with passkey/i })
    ).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Normal login
  // -----------------------------------------------------------------------

  it("calls login and redirects to dashboard on success", async () => {
    mockAuth.login.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.type(screen.getByLabelText(/username/i), "testuser");
    await user.type(screen.getByLabelText(/password/i), "password123");
    await user.click(screen.getByRole("button", { name: /sign in$/i }));

    await waitFor(() => {
      expect(mockAuth.login).toHaveBeenCalledWith("testuser", "password123");
      expect(mockRouterPush).toHaveBeenCalledWith("/dashboard");
    });
  });

  it("shows error on login failure with ApiError", async () => {
    const { ApiError } = await import("@/lib/api");
    mockAuth.login.mockRejectedValue(new ApiError("Invalid credentials"));
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.type(screen.getByLabelText(/username/i), "bad");
    await user.type(screen.getByLabelText(/password/i), "wrong");
    await user.click(screen.getByRole("button", { name: /sign in$/i }));

    await waitFor(() => {
      expect(screen.getByText("Invalid credentials")).toBeInTheDocument();
    });
  });

  it("shows generic error on unexpected failure", async () => {
    mockAuth.login.mockRejectedValue(new Error("network"));
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.type(screen.getByLabelText(/username/i), "test");
    await user.type(screen.getByLabelText(/password/i), "pass");
    await user.click(screen.getByRole("button", { name: /sign in$/i }));

    await waitFor(() => {
      expect(
        screen.getByText("An unexpected error occurred")
      ).toBeInTheDocument();
    });
  });

  // -----------------------------------------------------------------------
  // MFA flow
  // -----------------------------------------------------------------------

  it("shows MFA challenge when login returns MFA response", async () => {
    const mfaResp: MFAChallengeResponse = {
      mfa_required: true,
      mfa_token: "mfa-token",
      methods: ["totp"],
    };
    mockAuth.login.mockResolvedValue(mfaResp);
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.type(screen.getByLabelText(/username/i), "test");
    await user.type(screen.getByLabelText(/password/i), "pass");
    await user.click(screen.getByRole("button", { name: /sign in$/i }));

    await waitFor(() => {
      expect(screen.getByTestId("mfa-challenge")).toBeInTheDocument();
    });
  });

  it("calls completeMFALogin and redirects on MFA success", async () => {
    const mfaResp: MFAChallengeResponse = {
      mfa_required: true,
      mfa_token: "mfa-token",
      methods: ["totp"],
    };
    mockAuth.login.mockResolvedValue(mfaResp);
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.type(screen.getByLabelText(/username/i), "test");
    await user.type(screen.getByLabelText(/password/i), "pass");
    await user.click(screen.getByRole("button", { name: /sign in$/i }));

    await waitFor(() => {
      expect(screen.getByTestId("mfa-challenge")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /mfa success/i }));

    expect(mockAuth.completeMFALogin).toHaveBeenCalled();
    expect(mockRouterPush).toHaveBeenCalledWith("/dashboard");
  });

  it("returns to login form on MFA cancel", async () => {
    const mfaResp: MFAChallengeResponse = {
      mfa_required: true,
      mfa_token: "mfa-token",
      methods: ["totp"],
    };
    mockAuth.login.mockResolvedValue(mfaResp);
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.type(screen.getByLabelText(/username/i), "test");
    await user.type(screen.getByLabelText(/password/i), "pass");
    await user.click(screen.getByRole("button", { name: /sign in$/i }));

    await waitFor(() => {
      expect(screen.getByTestId("mfa-challenge")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /mfa cancel/i }));

    // Should be back to the login form
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(screen.queryByTestId("mfa-challenge")).not.toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Passkey login
  // -----------------------------------------------------------------------

  it("calls passkeyLogin and redirects on passkey success", async () => {
    mockAuth.passkeyLogin.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.click(
      screen.getByRole("button", { name: /sign in with passkey/i })
    );

    await waitFor(() => {
      expect(mockAuth.passkeyLogin).toHaveBeenCalled();
      expect(mockRouterPush).toHaveBeenCalledWith("/dashboard");
    });
  });

  it("silently handles NotAllowedError on passkey cancel", async () => {
    const domErr = new DOMException("User cancelled", "NotAllowedError");
    mockAuth.passkeyLogin.mockRejectedValue(domErr);
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.click(
      screen.getByRole("button", { name: /sign in with passkey/i })
    );

    await waitFor(() => {
      expect(mockAuth.passkeyLogin).toHaveBeenCalled();
    });

    // No error should be displayed for user cancellation
    expect(
      screen.queryByText(/passkey login failed/i)
    ).not.toBeInTheDocument();
  });

  it("shows error on passkey failure", async () => {
    mockAuth.passkeyLogin.mockRejectedValue(new Error("bad"));
    const user = userEvent.setup();
    render(<LoginPage />);

    await user.click(
      screen.getByRole("button", { name: /sign in with passkey/i })
    );

    await waitFor(() => {
      expect(screen.getByText("Passkey login failed")).toBeInTheDocument();
    });
  });
});

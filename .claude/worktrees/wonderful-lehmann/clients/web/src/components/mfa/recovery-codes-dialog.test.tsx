import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import {
  RecoveryCodesDisplay,
  RegenerateRecoveryCodesDialog,
} from "./recovery-codes-dialog";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const mockRecoveryCodesHook = {
  regenerate: vi.fn(),
  isLoading: false,
};

vi.mock("@/lib/hooks/use-mfa", () => ({
  useRecoveryCodes: () => mockRecoveryCodesHook,
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

const testCodes = ["aaaa-1111", "bbbb-2222", "cccc-3333", "dddd-4444"];

beforeEach(() => {
  vi.clearAllMocks();
});

describe("RecoveryCodesDisplay", () => {
  function renderDisplay(overrides: Partial<{ codes: string[]; onDone: () => void }> = {}) {
    const props = {
      codes: testCodes,
      onDone: vi.fn(),
      ...overrides,
    };
    return { ...render(<RecoveryCodesDisplay {...props} />), props };
  }

  it("renders all recovery codes", () => {
    renderDisplay();
    for (const code of testCodes) {
      expect(screen.getByText(code)).toBeInTheDocument();
    }
  });

  it("renders instructional text about saving codes", () => {
    renderDisplay();
    expect(
      screen.getByText(/save these recovery codes/i)
    ).toBeInTheDocument();
  });

  describe("Copy All", () => {
    it("copies codes to clipboard when Copy All is clicked", async () => {
      const writeText = vi.fn().mockResolvedValue(undefined);
      Object.defineProperty(window.navigator, "clipboard", {
        value: { writeText },
        configurable: true,
      });

      renderDisplay();
      fireEvent.click(screen.getByRole("button", { name: /copy all/i }));

      expect(writeText).toHaveBeenCalledWith(testCodes.join("\n"));
    });

    it("shows success toast after copying", async () => {
      Object.defineProperty(navigator, "clipboard", {
        value: { writeText: vi.fn().mockResolvedValue(undefined) },
        writable: true,
        configurable: true,
      });
      const { toast } = await import("sonner");

      const user = userEvent.setup();
      renderDisplay();
      await user.click(screen.getByRole("button", { name: /copy all/i }));

      expect(toast.success).toHaveBeenCalledWith("Recovery codes copied to clipboard");
    });
  });

  describe("Download", () => {
    it("triggers a file download when Download is clicked", async () => {
      const clickSpy = vi.fn();
      vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(clickSpy);
      vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:test");
      vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});

      const user = userEvent.setup();
      renderDisplay();
      await user.click(screen.getByRole("button", { name: /download/i }));

      expect(clickSpy).toHaveBeenCalledTimes(1);
      expect(URL.createObjectURL).toHaveBeenCalled();
      expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:test");
    });
  });

  describe("Done button and confirmation checkbox", () => {
    it("Done button is disabled until checkbox is checked", () => {
      renderDisplay();
      const doneBtn = screen.getByRole("button", { name: /done/i });
      expect(doneBtn).toBeDisabled();
    });

    it("checking the checkbox enables the Done button", async () => {
      const user = userEvent.setup();
      renderDisplay();
      const checkbox = screen.getByRole("checkbox");
      await user.click(checkbox);
      expect(screen.getByRole("button", { name: /done/i })).toBeEnabled();
    });

    it("clicking Done calls onDone", async () => {
      const user = userEvent.setup();
      const { props } = renderDisplay();
      // Must check confirmation first
      await user.click(screen.getByRole("checkbox"));
      await user.click(screen.getByRole("button", { name: /done/i }));
      expect(props.onDone).toHaveBeenCalledTimes(1);
    });
  });
});

describe("RegenerateRecoveryCodesDialog", () => {
  function renderDialog(overrides: Partial<{ open: boolean; onOpenChange: (open: boolean) => void }> = {}) {
    const props = {
      open: true,
      onOpenChange: vi.fn(),
      ...overrides,
    };
    return { ...render(<RegenerateRecoveryCodesDialog {...props} />), props };
  }

  it("does not show dialog content when open is false", () => {
    renderDialog({ open: false });
    expect(screen.queryByText("Recovery Codes")).not.toBeInTheDocument();
  });

  it("shows warning text and Regenerate Codes button when open", () => {
    renderDialog({ open: true });
    expect(screen.getByText("Recovery Codes")).toBeInTheDocument();
    expect(
      screen.getByText(/regenerating recovery codes will invalidate/i)
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /regenerate codes/i })
    ).toBeInTheDocument();
  });

  it("shows Cancel button that closes dialog", async () => {
    const user = userEvent.setup();
    const { props } = renderDialog({ open: true });
    await user.click(screen.getByRole("button", { name: /cancel/i }));
    expect(props.onOpenChange).toHaveBeenCalledWith(false);
  });

  it("calls regenerate and shows codes on success", async () => {
    mockRecoveryCodesHook.regenerate.mockResolvedValue({
      codes: ["xxxx-1111", "yyyy-2222"],
    });

    const user = userEvent.setup();
    renderDialog({ open: true });
    await user.click(screen.getByRole("button", { name: /regenerate codes/i }));

    await waitFor(() => {
      expect(screen.getByText("xxxx-1111")).toBeInTheDocument();
      expect(screen.getByText("yyyy-2222")).toBeInTheDocument();
    });
  });

  it("shows error toast when regenerate fails", async () => {
    mockRecoveryCodesHook.regenerate.mockRejectedValue(new Error("Network error"));
    const { toast } = await import("sonner");

    const user = userEvent.setup();
    renderDialog({ open: true });
    await user.click(screen.getByRole("button", { name: /regenerate codes/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("Failed to regenerate recovery codes");
    });
  });
});

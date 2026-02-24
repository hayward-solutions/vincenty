import { screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render, mockAuth } from "@/test/test-utils";
import GeneralSettingsPage from "./page";
import {
  AVAILABLE_SHAPES,
  MARKER_SHAPES,
  PRESET_COLORS,
  markerSVGString,
} from "@/components/map/marker-shapes";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
  toast: mockToast,
}));

const mocks = vi.hoisted(() => ({
  updateMe: vi.fn().mockResolvedValue({}),
  uploadAvatar: vi.fn().mockResolvedValue({}),
  deleteAvatar: vi.fn().mockResolvedValue({}),
}));

vi.mock("@/lib/hooks/use-profile", () => ({
  useUpdateMe: () => ({ updateMe: mocks.updateMe, isLoading: false }),
  useUploadAvatar: () => ({ uploadAvatar: mocks.uploadAvatar, isLoading: false }),
  useDeleteAvatar: () => ({ deleteAvatar: mocks.deleteAvatar, isLoading: false }),
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
// Setup
// ---------------------------------------------------------------------------

const defaultUser = { ...mockAuth.user };

beforeEach(() => {
  vi.clearAllMocks();
  // Reset mockAuth.user to defaults before each test
  Object.assign(mockAuth.user, defaultUser);
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("GeneralSettingsPage", () => {
  // 1. Renders page heading
  it("renders the General heading", () => {
    render(<GeneralSettingsPage />);
    expect(screen.getByText("General")).toBeInTheDocument();
  });

  // 2. Avatar card with initials fallback
  it("renders Avatar card with initials fallback when user has no avatar_url", () => {
    mockAuth.user.avatar_url = "";
    render(<GeneralSettingsPage />);

    expect(screen.getByText("Avatar")).toBeInTheDocument();
    // "Test User" → "TU"
    expect(screen.getByText("TU")).toBeInTheDocument();
  });

  // 3. Upload visible, Remove hidden when no avatar
  it("shows Upload button and does NOT show Remove button when no avatar", () => {
    mockAuth.user.avatar_url = "";
    render(<GeneralSettingsPage />);

    expect(screen.getByRole("button", { name: "Upload" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
  });

  // 4. Remove button shown when user has avatar_url
  it("shows Remove button when user has avatar_url", () => {
    mockAuth.user.avatar_url = "https://example.com/avatar.jpg";
    render(<GeneralSettingsPage />);

    expect(screen.getByRole("button", { name: "Upload" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
  });

  // 5. Profile form fields
  it("renders Profile form with username (disabled), display name, and email fields", () => {
    render(<GeneralSettingsPage />);

    const usernameInput = screen.getByLabelText("Username");
    expect(usernameInput).toBeInTheDocument();
    expect(usernameInput).toBeDisabled();
    expect(usernameInput).toHaveValue("testuser");

    const displayNameInput = screen.getByLabelText("Display Name");
    expect(displayNameInput).toBeInTheDocument();
    expect(displayNameInput).toHaveValue("Test User");

    const emailInput = screen.getByLabelText("Email");
    expect(emailInput).toBeInTheDocument();
    expect(emailInput).toHaveValue("test@example.com");

    expect(
      screen.getByRole("button", { name: "Save Changes" })
    ).toBeInTheDocument();
  });

  // 6. Profile save success
  it("calls updateMe with display_name and email, then refreshUser, shows success toast", async () => {
    mocks.updateMe.mockResolvedValue({});
    const user = userEvent.setup();
    render(<GeneralSettingsPage />);

    const displayNameInput = screen.getByLabelText("Display Name");
    const emailInput = screen.getByLabelText("Email");

    await user.clear(displayNameInput);
    await user.type(displayNameInput, "New Name");
    await user.clear(emailInput);
    await user.type(emailInput, "new@example.com");

    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(mocks.updateMe).toHaveBeenCalledWith({
        display_name: "New Name",
        email: "new@example.com",
      });
    });

    await waitFor(() => {
      expect(mockAuth.refreshUser).toHaveBeenCalled();
      expect(mockToast.success).toHaveBeenCalledWith("Profile updated");
    });
  });

  // 7. Profile save error
  it("shows error toast on profile save failure", async () => {
    mocks.updateMe.mockRejectedValue(new Error("Network error"));
    const user = userEvent.setup();
    render(<GeneralSettingsPage />);

    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith("Failed to update profile");
    });
  });

  // 8. Map Marker section with shape and color buttons
  it("renders Map Marker section with shape buttons and color buttons", () => {
    render(<GeneralSettingsPage />);

    expect(screen.getByText("Map Marker")).toBeInTheDocument();
    expect(screen.getByText("Shape")).toBeInTheDocument();
    expect(screen.getByText("Color")).toBeInTheDocument();

    // All shape labels are rendered
    for (const shape of AVAILABLE_SHAPES) {
      expect(screen.getByText(MARKER_SHAPES[shape].label)).toBeInTheDocument();
    }

    // All preset color buttons are rendered (by title)
    for (const color of PRESET_COLORS) {
      expect(screen.getByTitle(color)).toBeInTheDocument();
    }

    expect(
      screen.getByRole("button", { name: "Save Marker" })
    ).toBeInTheDocument();
  });

  // 9. Clicking a shape button updates the preview
  it("updates the marker preview when clicking a different shape", async () => {
    mockAuth.user.marker_icon = "circle";
    const user = userEvent.setup();
    render(<GeneralSettingsPage />);

    // Click the "Star" shape button
    await user.click(screen.getByText("Star"));

    // The preview area should now contain the star SVG
    const expectedSVG = markerSVGString("star", "#3b82f6", 36);
    // Find the preview container — it has the "Preview" label
    const previewLabel = screen.getByText("Preview");
    const previewContainer = previewLabel.closest("div.flex.flex-col");
    expect(previewContainer?.innerHTML).toContain(
      MARKER_SHAPES.star.path
    );
  });

  // 10. Typing valid custom hex color updates the marker color
  it("updates marker color when typing a valid custom hex", async () => {
    const user = userEvent.setup();
    render(<GeneralSettingsPage />);

    const customInput = screen.getByLabelText("Custom hex:");
    await user.type(customInput, "#ff0000");

    // A valid color swatch should appear next to the input
    await waitFor(() => {
      expect(customInput).toHaveValue("#ff0000");
    });

    // The preview should reflect the custom color
    const previewLabel = screen.getByText("Preview");
    const previewContainer = previewLabel.closest("div.flex.flex-col");
    expect(previewContainer?.innerHTML).toContain('fill="#ff0000"');
  });

  // 11. Save Marker calls updateMe with marker_icon and marker_color
  it("calls updateMe with marker_icon and marker_color on Save Marker", async () => {
    mocks.updateMe.mockResolvedValue({});
    mockAuth.user.marker_icon = "circle";
    mockAuth.user.marker_color = "#3b82f6";
    const user = userEvent.setup();
    render(<GeneralSettingsPage />);

    // Change shape to "diamond"
    await user.click(screen.getByText("Diamond"));

    // Change color to red preset
    await user.click(screen.getByTitle("#ef4444"));

    await user.click(screen.getByRole("button", { name: "Save Marker" }));

    await waitFor(() => {
      expect(mocks.updateMe).toHaveBeenCalledWith({
        marker_icon: "diamond",
        marker_color: "#ef4444",
      });
    });

    await waitFor(() => {
      expect(mockAuth.refreshUser).toHaveBeenCalled();
      expect(mockToast.success).toHaveBeenCalledWith("Map marker updated");
    });
  });

  // 12. Avatar upload rejects non-image file
  it("rejects non-image file on avatar upload", async () => {
    render(<GeneralSettingsPage />);

    const fileInput = document.querySelector(
      'input[type="file"]'
    ) as HTMLInputElement;
    expect(fileInput).toBeTruthy();

    const textFile = new File(["hello"], "document.txt", {
      type: "text/plain",
    });

    fireEvent.change(fileInput, { target: { files: [textFile] } });

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith(
        "Please select a JPEG, PNG, or WebP image"
      );
    });
    expect(mocks.uploadAvatar).not.toHaveBeenCalled();
  });

  // 13. Avatar upload rejects file > 5MB
  it("rejects file larger than 5 MB on avatar upload", async () => {
    render(<GeneralSettingsPage />);

    const fileInput = document.querySelector(
      'input[type="file"]'
    ) as HTMLInputElement;
    expect(fileInput).toBeTruthy();

    // Create a file that's just over 5MB
    const largeContent = new ArrayBuffer(5 * 1024 * 1024 + 1);
    const largeFile = new File([largeContent], "big-image.png", {
      type: "image/png",
    });

    fireEvent.change(fileInput, { target: { files: [largeFile] } });

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith(
        "Image must be smaller than 5 MB"
      );
    });
    expect(mocks.uploadAvatar).not.toHaveBeenCalled();
  });
});

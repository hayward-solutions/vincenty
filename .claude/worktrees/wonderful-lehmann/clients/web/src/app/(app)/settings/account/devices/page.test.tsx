import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import DevicesSettingsPage from "./page";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
  toast: mockToast,
}));

const mocks = vi.hoisted(() => ({
  devicesList: [] as unknown[],
  isLoading: false,
  error: null as string | null,
  fetchDevices: vi.fn(),
  deleteDevice: vi.fn().mockResolvedValue(undefined),
  setPrimary: vi.fn().mockResolvedValue(undefined),
  updateDevice: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@/lib/hooks/use-devices", () => ({
  useMyDevices: () => ({
    devices: mocks.devicesList,
    isLoading: mocks.isLoading,
    error: mocks.error,
    fetch: mocks.fetchDevices,
  }),
  useDeleteDevice: () => ({ deleteDevice: mocks.deleteDevice }),
  useSetPrimaryDevice: () => ({ setPrimary: mocks.setPrimary }),
  useUpdateDevice: () => ({ updateDevice: mocks.updateDevice, isLoading: false }),
}));

// ---------------------------------------------------------------------------
// Device fixture
// ---------------------------------------------------------------------------

const mockDeviceData = [
  {
    id: "device-1",
    user_id: "user-1",
    name: "Web Browser",
    device_type: "web",
    device_uid: "uid-123",
    is_primary: true,
    app_version: "sha-abc1234",
    user_agent:
      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    last_seen_at: new Date().toISOString(),
    created_at: "2025-01-15T00:00:00Z",
    updated_at: "2025-01-15T00:00:00Z",
  },
  {
    id: "device-2",
    user_id: "user-1",
    name: "Phone",
    device_type: "android",
    device_uid: "uid-456",
    is_primary: false,
    created_at: "2025-02-01T00:00:00Z",
    updated_at: "2025-02-01T00:00:00Z",
  },
];

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.clearAllMocks();
  mocks.devicesList = [...mockDeviceData];
  mocks.isLoading = false;
  mocks.error = null;
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("DevicesSettingsPage", () => {
  it('renders heading "Devices" and card title "Your Devices"', () => {
    render(<DevicesSettingsPage />);
    expect(screen.getByText("Devices")).toBeInTheDocument();
    expect(screen.getByText("Your Devices")).toBeInTheDocument();
  });

  it("calls fetch on mount", () => {
    render(<DevicesSettingsPage />);
    expect(mocks.fetchDevices).toHaveBeenCalled();
  });

  it("shows device names in table rows", () => {
    render(<DevicesSettingsPage />);
    expect(screen.getByText("Web Browser")).toBeInTheDocument();
    expect(screen.getByText("Phone")).toBeInTheDocument();
  });

  it('shows "This device" badge for device matching deviceId from websocket context', () => {
    render(<DevicesSettingsPage />);
    expect(screen.getByText("This device")).toBeInTheDocument();
  });

  it('shows "Primary" badge for primary device', () => {
    render(<DevicesSettingsPage />);
    const badges = screen.getAllByText("Primary");
    // One is the table column header, the other is the badge in the row
    const primaryBadge = badges.find(
      (el) => el.getAttribute("data-slot") === "badge"
    );
    expect(primaryBadge).toBeDefined();
  });

  it('shows "Set as primary" button for non-primary device', () => {
    render(<DevicesSettingsPage />);
    expect(
      screen.getByRole("button", { name: /set as primary/i })
    ).toBeInTheDocument();
  });

  it("Remove button is disabled for the current device", () => {
    render(<DevicesSettingsPage />);
    const removeButtons = screen.getAllByRole("button", { name: /remove/i });
    // device-1 is first row, device-2 is second row
    expect(removeButtons[0]).toBeDisabled();
  });

  it("Remove button is enabled for non-current devices", () => {
    render(<DevicesSettingsPage />);
    const removeButtons = screen.getAllByRole("button", { name: /remove/i });
    expect(removeButtons[1]).toBeEnabled();
  });

  it("clicking Remove shows confirm, on confirm calls deleteDevice and shows toast", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const user = userEvent.setup();
    render(<DevicesSettingsPage />);

    const removeButtons = screen.getAllByRole("button", { name: /remove/i });
    await user.click(removeButtons[1]);

    expect(window.confirm).toHaveBeenCalledWith(
      expect.stringContaining("Phone")
    );

    await waitFor(() => {
      expect(mocks.deleteDevice).toHaveBeenCalledWith("device-2");
    });

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith(
        expect.stringContaining("Phone")
      );
    });

    expect(mocks.fetchDevices).toHaveBeenCalled();
  });

  it("displays browser info parsed from user_agent", () => {
    render(<DevicesSettingsPage />);
    // Chrome 120 on macOS from the fixture UA string
    expect(screen.getByText("Chrome 120 on macOS")).toBeInTheDocument();
  });

  it('shows "No devices registered" when device list is empty', () => {
    mocks.devicesList = [];
    render(<DevicesSettingsPage />);
    expect(screen.getByText("No devices registered")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Set as primary
  // -----------------------------------------------------------------------

  it("clicking Set as primary calls setPrimary and shows toast", async () => {
    const user = userEvent.setup();
    render(<DevicesSettingsPage />);

    await user.click(
      screen.getByRole("button", { name: /set as primary/i })
    );

    await waitFor(() => {
      expect(mocks.setPrimary).toHaveBeenCalledWith("device-2");
    });

    await waitFor(() => {
      expect(mockToast.success).toHaveBeenCalledWith(
        expect.stringContaining("Phone")
      );
    });
  });

  // -----------------------------------------------------------------------
  // Remove cancelled
  // -----------------------------------------------------------------------

  it("does not call deleteDevice when confirm is cancelled", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    const user = userEvent.setup();
    render(<DevicesSettingsPage />);

    const removeButtons = screen.getAllByRole("button", { name: /remove/i });
    await user.click(removeButtons[1]);

    expect(window.confirm).toHaveBeenCalled();
    expect(mocks.deleteDevice).not.toHaveBeenCalled();
  });

  // -----------------------------------------------------------------------
  // Loading state
  // -----------------------------------------------------------------------

  it("shows skeletons when loading", () => {
    mocks.isLoading = true;
    mocks.devicesList = [];
    const { container } = render(<DevicesSettingsPage />);
    // Skeletons are rendered as divs with the Skeleton class; there should be 3
    const skeletons = container.querySelectorAll('[class*="animate-pulse"], [data-slot="skeleton"]');
    expect(skeletons.length).toBeGreaterThanOrEqual(3);
  });

  // -----------------------------------------------------------------------
  // Error state
  // -----------------------------------------------------------------------

  it("shows error message when error exists", () => {
    mocks.error = "Network error";
    render(<DevicesSettingsPage />);
    expect(screen.getByText("Network error")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Rename dialog
  // -----------------------------------------------------------------------

  it("renders rename buttons for each device", () => {
    render(<DevicesSettingsPage />);
    const renameButtons = screen.getAllByTitle("Rename device");
    expect(renameButtons).toHaveLength(2);
  });

  // -----------------------------------------------------------------------
  // Version column
  // -----------------------------------------------------------------------

  it("shows app_version in Version column when set", () => {
    render(<DevicesSettingsPage />);
    expect(screen.getByText("sha-abc1234")).toBeInTheDocument();
  });

  it('shows "—" in Version column for devices with no app_version', () => {
    render(<DevicesSettingsPage />);
    // device-2 has no app_version; the cell should display an em-dash
    const versionCells = screen.getAllByText("—");
    expect(versionCells.length).toBeGreaterThanOrEqual(1);
  });
});

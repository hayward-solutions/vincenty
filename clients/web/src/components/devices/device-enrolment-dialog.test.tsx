import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { DeviceEnrolmentDialog } from "./device-enrolment-dialog";
import type { Device } from "@/types/api";

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const mockCreateDevice = vi.fn();
const mockClaimDevice = vi.fn();
vi.mock("@/lib/hooks/use-devices", () => ({
  useCreateDevice: () => ({ createDevice: mockCreateDevice, isLoading: false }),
  useClaimDevice: () => ({ claimDevice: mockClaimDevice, isLoading: false }),
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

const makeDevice = (overrides: Partial<Device> = {}): Device => ({
  id: "device-1",
  user_id: "user-1",
  name: "My Laptop",
  device_type: "browser",
  device_uid: "uid-abc",
  is_primary: true,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
  user_agent:
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
  last_seen_at: "2025-01-15T10:30:00Z",
  ...overrides,
});

const defaultProps = () => ({
  existingDevices: [] as Device[],
  onResolved: vi.fn(),
});

function renderDialog(
  overrides: Partial<ReturnType<typeof defaultProps>> = {}
) {
  const props = { ...defaultProps(), ...overrides };
  return { ...render(<DeviceEnrolmentDialog {...props} />), props };
}

beforeEach(() => {
  mockCreateDevice.mockReset();
  mockClaimDevice.mockReset();
});

describe("DeviceEnrolmentDialog", () => {
  describe("rendering", () => {
    it("renders dialog with title 'Device Not Recognised'", () => {
      renderDialog();
      expect(screen.getByText("Device Not Recognised")).toBeInTheDocument();
    });

    it("renders description text about the unrecognised browser", () => {
      renderDialog();
      expect(
        screen.getByText(/don.t recognise this browser/)
      ).toBeInTheDocument();
    });

    it("renders Register button", () => {
      renderDialog();
      expect(
        screen.getByRole("button", { name: "Register" })
      ).toBeInTheDocument();
    });

    it("renders the device name input", () => {
      renderDialog();
      expect(
        screen.getByText("Register as new device")
      ).toBeInTheDocument();
    });
  });

  describe("existing devices", () => {
    it("shows existing devices section when devices are provided", () => {
      const devices = [makeDevice(), makeDevice({ id: "device-2", name: "Work PC" })];
      renderDialog({ existingDevices: devices });
      expect(screen.getByText("Your existing devices")).toBeInTheDocument();
      expect(screen.getByText("My Laptop")).toBeInTheDocument();
      expect(screen.getByText("Work PC")).toBeInTheDocument();
    });

    it("shows 'Use this' button for each existing device", () => {
      const devices = [makeDevice(), makeDevice({ id: "device-2", name: "Work PC" })];
      renderDialog({ existingDevices: devices });
      const useButtons = screen.getAllByRole("button", { name: "Use this" });
      expect(useButtons).toHaveLength(2);
    });

    it("does not show existing devices section when devices is empty", () => {
      renderDialog({ existingDevices: [] });
      expect(
        screen.queryByText("Your existing devices")
      ).not.toBeInTheDocument();
    });

    it("shows parsed browser name from user_agent", () => {
      const device = makeDevice({
        user_agent:
          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
      });
      renderDialog({ existingDevices: [device] });
      // parseBrowserName should produce "Chrome 120 on Windows"
      expect(screen.getByText(/Chrome 120 on Windows/)).toBeInTheDocument();
    });

    it("shows relative time for last_seen_at", () => {
      // Set last_seen_at to undefined to get "Never"
      const device = makeDevice({ last_seen_at: undefined });
      renderDialog({ existingDevices: [device] });
      expect(screen.getByText(/Never/)).toBeInTheDocument();
    });

    it("shows device_type as a badge", () => {
      const device = makeDevice({ device_type: "browser" });
      renderDialog({ existingDevices: [device] });
      expect(screen.getByText("browser")).toBeInTheDocument();
    });
  });

  describe("claim device", () => {
    it("clicking 'Use this' calls claimDevice with the device id", async () => {
      const user = userEvent.setup();
      const device = makeDevice({ id: "device-42" });
      mockClaimDevice.mockResolvedValue({ id: "device-42" });
      const { props } = renderDialog({ existingDevices: [device] });

      await user.click(screen.getByRole("button", { name: "Use this" }));

      expect(mockClaimDevice).toHaveBeenCalledWith("device-42");
    });

    it("calls onResolved after successful claim", async () => {
      const user = userEvent.setup();
      const device = makeDevice({ id: "device-42" });
      mockClaimDevice.mockResolvedValue({ id: "device-42" });
      const { props } = renderDialog({ existingDevices: [device] });

      await user.click(screen.getByRole("button", { name: "Use this" }));

      await waitFor(() => {
        expect(props.onResolved).toHaveBeenCalledWith("device-42");
      });
    });
  });

  describe("register new device", () => {
    it("clicking Register calls createDevice", async () => {
      const user = userEvent.setup();
      mockCreateDevice.mockResolvedValue({ id: "new-device-1" });
      const { props } = renderDialog();

      await user.click(screen.getByRole("button", { name: "Register" }));

      expect(mockCreateDevice).toHaveBeenCalled();
    });

    it("calls onResolved after successful registration", async () => {
      const user = userEvent.setup();
      mockCreateDevice.mockResolvedValue({ id: "new-device-1" });
      const { props } = renderDialog();

      await user.click(screen.getByRole("button", { name: "Register" }));

      await waitFor(() => {
        expect(props.onResolved).toHaveBeenCalledWith("new-device-1");
      });
    });

    it("uses custom device name when provided", async () => {
      const user = userEvent.setup();
      mockCreateDevice.mockResolvedValue({ id: "new-device-1" });
      renderDialog();

      const input = screen.getByRole("textbox");
      await user.type(input, "My Phone");
      await user.click(screen.getByRole("button", { name: "Register" }));

      expect(mockCreateDevice).toHaveBeenCalledWith("My Phone");
    });
  });
});

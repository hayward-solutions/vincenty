import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { ReplayPanel } from "./replay-panel";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const defaultProps = () => ({
  isLoading: false,
  onStart: vi.fn(),
  onExportGPX: vi.fn(),
  onCancel: vi.fn(),
});

function renderPanel(overrides: Partial<ReturnType<typeof defaultProps>> = {}) {
  const props = { ...defaultProps(), ...overrides };
  return { ...render(<ReplayPanel {...props} />), props };
}

describe("ReplayPanel", () => {
  describe("rendering", () => {
    it("renders preset buttons (1h, 6h, 24h, Custom)", () => {
      renderPanel();
      expect(screen.getByRole("button", { name: "1h" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "6h" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "24h" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Custom" })).toBeInTheDocument();
    });

    it("renders Start, Export GPX, and Cancel buttons", () => {
      renderPanel();
      expect(screen.getByRole("button", { name: "Start" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Export GPX" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    });
  });

  describe("preset selection", () => {
    it("1h preset is active by default", () => {
      renderPanel();
      const btn1h = screen.getByRole("button", { name: "1h" });
      expect(btn1h.className).toContain("bg-primary");
    });

    it("clicking 6h changes active preset", async () => {
      const user = userEvent.setup();
      renderPanel();
      await user.click(screen.getByRole("button", { name: "6h" }));
      expect(screen.getByRole("button", { name: "6h" }).className).toContain("bg-primary");
      expect(screen.getByRole("button", { name: "1h" }).className).toContain("bg-secondary");
    });

    it("clicking Custom shows datetime inputs", async () => {
      const user = userEvent.setup();
      renderPanel();
      expect(screen.queryByText("From")).not.toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: "Custom" }));
      expect(screen.getByText("From")).toBeInTheDocument();
      expect(screen.getByText("To")).toBeInTheDocument();
    });
  });

  describe("actions", () => {
    it("calls onStart with date range when Start is clicked", async () => {
      const user = userEvent.setup();
      const now = Date.now();
      vi.setSystemTime(now);
      const { props } = renderPanel();
      await user.click(screen.getByRole("button", { name: "Start" }));
      expect(props.onStart).toHaveBeenCalledTimes(1);
      const { from, to } = props.onStart.mock.calls[0][0];
      // Default preset is 1h: from should be ~1 hour before to
      expect(to.getTime() - from.getTime()).toBe(60 * 60 * 1000);
      vi.useRealTimers();
    });

    it("calls onExportGPX with date range when Export GPX is clicked", async () => {
      const user = userEvent.setup();
      const now = Date.now();
      vi.setSystemTime(now);
      const { props } = renderPanel();
      await user.click(screen.getByRole("button", { name: "Export GPX" }));
      expect(props.onExportGPX).toHaveBeenCalledTimes(1);
      const [from, to] = props.onExportGPX.mock.calls[0];
      expect(to.getTime() - from.getTime()).toBe(60 * 60 * 1000);
      vi.useRealTimers();
    });

    it("calls onCancel when Cancel is clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel();
      await user.click(screen.getByRole("button", { name: "Cancel" }));
      expect(props.onCancel).toHaveBeenCalledTimes(1);
    });
  });

  describe("loading state", () => {
    it("disables Start button and shows 'Loading...' when isLoading is true", () => {
      renderPanel({ isLoading: true });
      const startBtn = screen.getByRole("button", { name: "Loading..." });
      expect(startBtn).toBeDisabled();
    });

    it("Start button shows 'Start' and is enabled when not loading", () => {
      renderPanel({ isLoading: false });
      const startBtn = screen.getByRole("button", { name: "Start" });
      expect(startBtn).toBeEnabled();
    });
  });
});

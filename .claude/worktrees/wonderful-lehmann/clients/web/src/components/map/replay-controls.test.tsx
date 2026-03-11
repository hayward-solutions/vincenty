import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { ReplayControls } from "./replay-controls";

const from = new Date("2025-06-01T10:00:00Z");
const to = new Date("2025-06-01T11:00:00Z");

const defaultProps = () => ({
  from,
  to,
  onTimeChange: vi.fn(),
  onReset: vi.fn(),
});

function renderControls(overrides: Partial<ReturnType<typeof defaultProps>> = {}) {
  const props = { ...defaultProps(), ...overrides };
  return { ...render(<ReplayControls {...props} />), props };
}

describe("ReplayControls", () => {
  describe("rendering", () => {
    it("renders play/restart button, speed button, slider, and Close button", () => {
      renderControls();
      // Initial progress is 100%, so shows "R" (restart)
      expect(screen.getByRole("button", { name: "R" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "1x" })).toBeInTheDocument();
      expect(screen.getByRole("slider")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
    });

    it("initial progress is 100% showing R (restart)", () => {
      renderControls();
      expect(screen.getByRole("button", { name: "R" })).toBeInTheDocument();
    });
  });

  describe("play/pause toggle", () => {
    it("clicking play when at 100% resets and starts playing (shows ||)", async () => {
      const user = userEvent.setup();
      renderControls();
      // At 100%, button shows "R"
      const playBtn = screen.getByRole("button", { name: "R" });
      await user.click(playBtn);
      // After clicking, progress resets to 0 and starts playing → shows "||"
      expect(screen.getByRole("button", { name: "||" })).toBeInTheDocument();
    });

    it("clicking || pauses playback (shows ▶)", async () => {
      const user = userEvent.setup();
      renderControls();
      // Click R to start playing
      await user.click(screen.getByRole("button", { name: "R" }));
      // Now pause
      await user.click(screen.getByRole("button", { name: "||" }));
      // Should show play icon ▶
      expect(screen.getByRole("button", { name: "▶" })).toBeInTheDocument();
    });
  });

  describe("speed cycling", () => {
    it("cycles through 1x → 2x → 5x → 10x → 1x", async () => {
      const user = userEvent.setup();
      renderControls();

      expect(screen.getByRole("button", { name: "1x" })).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "1x" }));
      expect(screen.getByRole("button", { name: "2x" })).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "2x" }));
      expect(screen.getByRole("button", { name: "5x" })).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "5x" }));
      expect(screen.getByRole("button", { name: "10x" })).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "10x" }));
      expect(screen.getByRole("button", { name: "1x" })).toBeInTheDocument();
    });
  });

  describe("Close button", () => {
    it("calls onReset when Close is clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderControls();
      await user.click(screen.getByRole("button", { name: "Close" }));
      expect(props.onReset).toHaveBeenCalledTimes(1);
    });
  });

  describe("slider", () => {
    it("calls onTimeChange when slider value changes", () => {
      const { props } = renderControls();
      const slider = screen.getByRole("slider");
      // Simulate a change to 50%
      fireEvent.change(slider, { target: { value: "50" } });
      expect(props.onTimeChange).toHaveBeenCalledTimes(1);
      const calledTime = props.onTimeChange.mock.calls[0][0] as Date;
      // 50% of the range: from + 30 minutes
      const expected = new Date(from.getTime() + 30 * 60 * 1000);
      expect(calledTime.getTime()).toBe(expected.getTime());
    });
  });
});

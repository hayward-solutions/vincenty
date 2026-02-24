import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { MapToolbar } from "./map-toolbar";

// Radix UI Tooltip uses ResizeObserver which jsdom doesn't provide
if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
}

function renderToolbar(overrides: Partial<Parameters<typeof MapToolbar>[0]> = {}) {
  const defaults = {
    onReplayClick: vi.fn(),
    replayActive: false,
    filterActive: false,
    onFilterClick: vi.fn(),
    measureActive: false,
    onMeasureClick: vi.fn(),
    drawActive: false,
    onDrawClick: vi.fn(),
  };
  const props = { ...defaults, ...overrides };
  return { ...render(<MapToolbar {...props} />), props };
}

describe("MapToolbar", () => {
  describe("rendering", () => {
    it("renders all four buttons with correct aria-labels", () => {
      renderToolbar();
      expect(screen.getByRole("button", { name: "Replay" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Filters" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Measure" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Draw" })).toBeInTheDocument();
    });
  });

  describe("click handlers", () => {
    it("calls onReplayClick when Replay is clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderToolbar();
      await user.click(screen.getByRole("button", { name: "Replay" }));
      expect(props.onReplayClick).toHaveBeenCalledTimes(1);
    });

    it("calls onFilterClick when Filters is clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderToolbar();
      await user.click(screen.getByRole("button", { name: "Filters" }));
      expect(props.onFilterClick).toHaveBeenCalledTimes(1);
    });

    it("calls onMeasureClick when Measure is clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderToolbar();
      await user.click(screen.getByRole("button", { name: "Measure" }));
      expect(props.onMeasureClick).toHaveBeenCalledTimes(1);
    });

    it("calls onDrawClick when Draw is clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderToolbar();
      await user.click(screen.getByRole("button", { name: "Draw" }));
      expect(props.onDrawClick).toHaveBeenCalledTimes(1);
    });
  });

  describe("active state", () => {
    it.each([
      ["Replay", "replayActive"],
      ["Filters", "filterActive"],
      ["Measure", "measureActive"],
      ["Draw", "drawActive"],
    ] as const)("applies text-foreground when %s is active", (label, prop) => {
      renderToolbar({ [prop]: true });
      const button = screen.getByRole("button", { name: label });
      expect(button.className).toContain("text-foreground");
      expect(button.className).not.toContain("text-muted-foreground");
    });

    it.each([
      ["Replay", "replayActive"],
      ["Filters", "filterActive"],
      ["Measure", "measureActive"],
      ["Draw", "drawActive"],
    ] as const)("applies text-muted-foreground when %s is inactive", (label, prop) => {
      renderToolbar({ [prop]: false });
      const button = screen.getByRole("button", { name: label });
      expect(button.className).toContain("text-muted-foreground");
    });
  });
});

import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { MeasurePanel } from "./measure-panel";
import type { MeasureResult } from "./measure-tool";

vi.mock("./measure-tool", () => ({
  formatDistance: (m: number) =>
    m < 1000 ? `${Math.round(m)}m` : `${(m / 1000).toFixed(2)}km`,
  formatArea: (m: number) =>
    m < 1_000_000
      ? `${Math.round(m).toLocaleString()}m²`
      : `${(m / 1_000_000).toFixed(2)}km²`,
}));

const emptyMeasurements: MeasureResult = { segments: [], total: 0 };

const defaultProps = () => ({
  mode: "line" as const,
  onModeChange: vi.fn(),
  measurements: emptyMeasurements,
  onClear: vi.fn(),
  onClose: vi.fn(),
});

function renderPanel(overrides: Partial<ReturnType<typeof defaultProps>> = {}) {
  const props = { ...defaultProps(), ...overrides };
  return { ...render(<MeasurePanel {...props} />), props };
}

describe("MeasurePanel", () => {
  describe("rendering", () => {
    it("renders Distance and Radius mode buttons", () => {
      renderPanel();
      expect(screen.getByText("Distance")).toBeInTheDocument();
      expect(screen.getByText("Radius")).toBeInTheDocument();
    });

    it("renders Clear and Close buttons", () => {
      renderPanel();
      expect(screen.getByRole("button", { name: "Clear" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
    });
  });

  describe("mode selection", () => {
    it("Distance button has bg-primary class in line mode", () => {
      renderPanel({ mode: "line" });
      const distBtn = screen.getByText("Distance");
      expect(distBtn.className).toContain("bg-primary");
    });

    it("Radius button has bg-primary class in circle mode", () => {
      renderPanel({ mode: "circle" });
      const radiusBtn = screen.getByText("Radius");
      expect(radiusBtn.className).toContain("bg-primary");
    });

    it("Distance button does NOT have bg-primary in circle mode", () => {
      renderPanel({ mode: "circle" });
      const distBtn = screen.getByText("Distance");
      expect(distBtn.className).not.toContain("bg-primary");
    });

    it("Radius button does NOT have bg-primary in line mode", () => {
      renderPanel({ mode: "line" });
      const radiusBtn = screen.getByText("Radius");
      expect(radiusBtn.className).not.toContain("bg-primary");
    });

    it("clicking Distance calls onModeChange with 'line'", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ mode: "circle" });
      await user.click(screen.getByText("Distance"));
      expect(props.onModeChange).toHaveBeenCalledWith("line");
    });

    it("clicking Radius calls onModeChange with 'circle'", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ mode: "line" });
      await user.click(screen.getByText("Radius"));
      expect(props.onModeChange).toHaveBeenCalledWith("circle");
    });
  });

  describe("line mode measurements", () => {
    it("shows instruction text when there are no segments", () => {
      renderPanel({ mode: "line", measurements: { segments: [], total: 0 } });
      expect(
        screen.getByText(/Click on the map to place points/)
      ).toBeInTheDocument();
    });

    it("shows Total with formatted distance when segments exist", () => {
      renderPanel({
        mode: "line",
        measurements: { segments: [500, 300], total: 800 },
      });
      expect(screen.getByText("Total")).toBeInTheDocument();
      expect(screen.getByText("800m")).toBeInTheDocument();
    });

    it("formats distances >= 1000m in km", () => {
      renderPanel({
        mode: "line",
        measurements: { segments: [1500], total: 1500 },
      });
      expect(screen.getByText("1.50km")).toBeInTheDocument();
    });
  });

  describe("circle mode measurements", () => {
    it("shows instruction text when there is no radius", () => {
      renderPanel({ mode: "circle", measurements: { segments: [], total: 0 } });
      expect(
        screen.getByText(/Click to place the centre/)
      ).toBeInTheDocument();
    });

    it("shows Radius and Area when radius is set", () => {
      renderPanel({
        mode: "circle",
        measurements: {
          segments: [],
          total: 0,
          radius: 500,
          area: 785398,
        },
      });
      // "Radius" appears both as the mode button and the measurement label
      const radiusElements = screen.getAllByText("Radius");
      expect(radiusElements.length).toBe(2);
      expect(screen.getByText("500m")).toBeInTheDocument();
      expect(screen.getByText("Area")).toBeInTheDocument();
      expect(screen.getByText("785,398m²")).toBeInTheDocument();
    });

    it("formats large areas in km²", () => {
      renderPanel({
        mode: "circle",
        measurements: {
          segments: [],
          total: 0,
          radius: 5000,
          area: 78_539_816,
        },
      });
      expect(screen.getByText("5.00km")).toBeInTheDocument();
      expect(screen.getByText("78.54km²")).toBeInTheDocument();
    });

    it("does not show area when area is 0", () => {
      renderPanel({
        mode: "circle",
        measurements: { segments: [], total: 0, radius: 100, area: 0 },
      });
      // "Radius" appears as both mode button and measurement label
      const radiusElements = screen.getAllByText("Radius");
      expect(radiusElements.length).toBe(2);
      expect(screen.queryByText("Area")).not.toBeInTheDocument();
    });
  });

  describe("actions", () => {
    it("clicking Clear calls onClear", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel();
      await user.click(screen.getByRole("button", { name: "Clear" }));
      expect(props.onClear).toHaveBeenCalledTimes(1);
    });

    it("clicking Close calls onClose", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel();
      await user.click(screen.getByRole("button", { name: "Close" }));
      expect(props.onClose).toHaveBeenCalledTimes(1);
    });
  });
});

import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { AuditFilterBar } from "./audit-filters";

describe("AuditFilterBar", () => {
  describe("rendering", () => {
    it("renders the Action dropdown", () => {
      render(<AuditFilterBar onApply={vi.fn()} />);
      expect(screen.getByText("Action")).toBeInTheDocument();
      expect(screen.getByDisplayValue("All Actions")).toBeInTheDocument();
    });

    it("renders the Resource Type dropdown", () => {
      render(<AuditFilterBar onApply={vi.fn()} />);
      expect(screen.getByText("Resource Type")).toBeInTheDocument();
      expect(screen.getByDisplayValue("All Types")).toBeInTheDocument();
    });

    it("renders From and To date inputs", () => {
      render(<AuditFilterBar onApply={vi.fn()} />);
      expect(screen.getByText("From")).toBeInTheDocument();
      expect(screen.getByText("To")).toBeInTheDocument();
    });

    it("renders Filter and Reset buttons", () => {
      render(<AuditFilterBar onApply={vi.fn()} />);
      expect(
        screen.getByRole("button", { name: "Filter" })
      ).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Reset" })
      ).toBeInTheDocument();
    });
  });

  describe("filtering", () => {
    it("calls onApply with action when an action is selected and Filter is clicked", async () => {
      const user = userEvent.setup();
      const onApply = vi.fn();
      render(<AuditFilterBar onApply={onApply} />);

      const actionSelect = screen.getByDisplayValue("All Actions");
      await user.selectOptions(actionSelect, "auth.login");
      await user.click(screen.getByRole("button", { name: "Filter" }));

      expect(onApply).toHaveBeenCalledTimes(1);
      expect(onApply).toHaveBeenCalledWith({ action: "auth.login" });
    });

    it("calls onApply with both action and resource_type when both are selected", async () => {
      const user = userEvent.setup();
      const onApply = vi.fn();
      render(<AuditFilterBar onApply={onApply} />);

      const actionSelect = screen.getByDisplayValue("All Actions");
      await user.selectOptions(actionSelect, "auth.login");

      const typeSelect = screen.getByDisplayValue("All Types");
      await user.selectOptions(typeSelect, "session");

      await user.click(screen.getByRole("button", { name: "Filter" }));

      expect(onApply).toHaveBeenCalledWith({
        action: "auth.login",
        resource_type: "session",
      });
    });

    it("does not include empty selections in the filters object", async () => {
      const user = userEvent.setup();
      const onApply = vi.fn();
      render(<AuditFilterBar onApply={onApply} />);

      // Click Filter without selecting anything
      await user.click(screen.getByRole("button", { name: "Filter" }));

      expect(onApply).toHaveBeenCalledWith({});
    });

    it("includes date inputs in filters when set", async () => {
      const user = userEvent.setup();
      const onApply = vi.fn();
      render(<AuditFilterBar onApply={onApply} />);

      // Find datetime-local inputs
      const dateInputs = document.querySelectorAll(
        'input[type="datetime-local"]'
      );
      const fromInput = dateInputs[0] as HTMLInputElement;
      const toInput = dateInputs[1] as HTMLInputElement;

      // Fill in dates using fireEvent since datetime-local inputs
      // don't respond well to userEvent.type
      await user.clear(fromInput);
      await user.type(fromInput, "2025-01-01T00:00");
      await user.clear(toInput);
      await user.type(toInput, "2025-12-31T23:59");

      await user.click(screen.getByRole("button", { name: "Filter" }));

      expect(onApply).toHaveBeenCalledTimes(1);
      const filters = onApply.mock.calls[0][0];
      expect(filters.from).toBeDefined();
      expect(filters.to).toBeDefined();
      // Values should be ISO strings
      expect(filters.from).toMatch(/^\d{4}-\d{2}-\d{2}T/);
      expect(filters.to).toMatch(/^\d{4}-\d{2}-\d{2}T/);
    });
  });

  describe("reset", () => {
    it("clears all filters and calls onApply with empty object", async () => {
      const user = userEvent.setup();
      const onApply = vi.fn();
      render(<AuditFilterBar onApply={onApply} />);

      // Set some filters first
      const actionSelect = screen.getByDisplayValue("All Actions");
      await user.selectOptions(actionSelect, "auth.login");

      const typeSelect = screen.getByDisplayValue("All Types");
      await user.selectOptions(typeSelect, "user");

      // Click Reset
      await user.click(screen.getByRole("button", { name: "Reset" }));

      // onApply should be called with empty object
      const lastCall = onApply.mock.calls[onApply.mock.calls.length - 1];
      expect(lastCall[0]).toEqual({});

      // Dropdowns should be reset to default values
      expect(screen.getByDisplayValue("All Actions")).toBeInTheDocument();
      expect(screen.getByDisplayValue("All Types")).toBeInTheDocument();
    });
  });
});

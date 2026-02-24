import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { mockGroup } from "@/test/fixtures";
import type { Group } from "@/types/api";
import { FilterPanel } from "./filter-panel";

// Mock marker-shapes to avoid SVG generation in tests
vi.mock("@/components/map/marker-shapes", () => ({
  markerSVGString: () => '<svg data-testid="marker-icon"></svg>',
}));

function makeGroup(overrides: Partial<Group> & { id: string; name: string }): Group {
  return { ...mockGroup, ...overrides };
}

const defaultProps = () => ({
  showSelf: true,
  onShowSelfChange: vi.fn(),
  showDrawings: true,
  onShowDrawingsChange: vi.fn(),
  primaryOnly: false,
  onPrimaryOnlyChange: vi.fn(),
  groups: [] as Group[],
  selectedGroupIds: new Set<string>(),
  onGroupToggle: vi.fn(),
  onGroupsClear: vi.fn(),
  users: [] as Array<{ user_id: string; display_name: string; username: string }>,
  selectedUserIds: new Set<string>(),
  onUserToggle: vi.fn(),
  onUsersClear: vi.fn(),
});

function renderPanel(overrides: Partial<ReturnType<typeof defaultProps>> = {}) {
  const props = { ...defaultProps(), ...overrides };
  return { ...render(<FilterPanel {...props} />), props };
}

describe("FilterPanel", () => {
  describe("Show self checkbox", () => {
    it("renders checked when showSelf is true", () => {
      renderPanel({ showSelf: true });
      const checkbox = screen.getByRole("checkbox", { name: /show self/i });
      expect(checkbox).toBeChecked();
    });

    it("renders unchecked when showSelf is false", () => {
      renderPanel({ showSelf: false });
      const checkbox = screen.getByRole("checkbox", { name: /show self/i });
      expect(checkbox).not.toBeChecked();
    });

    it("calls onShowSelfChange when toggled", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ showSelf: true });
      await user.click(screen.getByRole("checkbox", { name: /show self/i }));
      expect(props.onShowSelfChange).toHaveBeenCalledWith(false);
    });
  });

  describe("Show drawings checkbox", () => {
    it("renders checked when showDrawings is true", () => {
      renderPanel({ showDrawings: true });
      const checkbox = screen.getByRole("checkbox", { name: /show drawings/i });
      expect(checkbox).toBeChecked();
    });

    it("calls onShowDrawingsChange when toggled", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ showDrawings: true });
      await user.click(screen.getByRole("checkbox", { name: /show drawings/i }));
      expect(props.onShowDrawingsChange).toHaveBeenCalledWith(false);
    });
  });

  describe("Primary devices only checkbox", () => {
    it("renders unchecked when primaryOnly is false", () => {
      renderPanel({ primaryOnly: false });
      const checkbox = screen.getByRole("checkbox", { name: /primary devices only/i });
      expect(checkbox).not.toBeChecked();
    });

    it("calls onPrimaryOnlyChange when toggled", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ primaryOnly: false });
      await user.click(screen.getByRole("checkbox", { name: /primary devices only/i }));
      expect(props.onPrimaryOnlyChange).toHaveBeenCalledWith(true);
    });
  });

  describe("Groups section", () => {
    const groups = [
      makeGroup({ id: "g1", name: "Alpha Team" }),
      makeGroup({ id: "g2", name: "Bravo Team" }),
    ];

    it("renders group names when groups are provided", () => {
      renderPanel({ groups });
      expect(screen.getByText("Alpha Team")).toBeInTheDocument();
      expect(screen.getByText("Bravo Team")).toBeInTheDocument();
    });

    it('shows "Groups" section header', () => {
      renderPanel({ groups });
      expect(screen.getByText("Groups")).toBeInTheDocument();
    });

    it("does not show Groups header when groups is empty", () => {
      renderPanel({ groups: [] });
      expect(screen.queryByText("Groups")).not.toBeInTheDocument();
    });

    it("checks group checkbox when group is selected", () => {
      renderPanel({ groups, selectedGroupIds: new Set(["g1"]) });
      const checkboxes = screen.getAllByRole("checkbox");
      // First 2 are "show self" and "show drawings", then groups
      const g1Checkbox = screen.getByRole("checkbox", { name: /alpha team/i });
      expect(g1Checkbox).toBeChecked();
    });

    it("calls onGroupToggle with group id when clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ groups });
      await user.click(screen.getByRole("checkbox", { name: /bravo team/i }));
      expect(props.onGroupToggle).toHaveBeenCalledWith("g2");
    });

    it("shows Clear button only when groups are selected", () => {
      const { rerender } = renderPanel({ groups, selectedGroupIds: new Set<string>() });
      // No clear button when nothing selected
      expect(screen.queryByRole("button", { name: /clear/i })).not.toBeInTheDocument();
    });

    it("shows Clear button when groups are selected and calls onGroupsClear", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ groups, selectedGroupIds: new Set(["g1"]) });
      const clearButton = screen.getAllByRole("button", { name: /clear/i })[0];
      expect(clearButton).toBeInTheDocument();
      await user.click(clearButton);
      expect(props.onGroupsClear).toHaveBeenCalledTimes(1);
    });
  });

  describe("Users section", () => {
    const users = [
      { user_id: "u1", display_name: "Alice", username: "alice" },
      { user_id: "u2", display_name: "Bob", username: "bob" },
    ];

    it("renders user names when users are provided", () => {
      renderPanel({ users });
      expect(screen.getByText("Alice")).toBeInTheDocument();
      expect(screen.getByText("Bob")).toBeInTheDocument();
    });

    it('shows "Users" section header', () => {
      renderPanel({ users });
      expect(screen.getByText("Users")).toBeInTheDocument();
    });

    it("does not show Users header when users is empty", () => {
      renderPanel({ users: [] });
      expect(screen.queryByText("Users")).not.toBeInTheDocument();
    });

    it("calls onUserToggle with user id when clicked", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ users });
      await user.click(screen.getByRole("checkbox", { name: /bob/i }));
      expect(props.onUserToggle).toHaveBeenCalledWith("u2");
    });

    it("shows Clear button when users are selected and calls onUsersClear", async () => {
      const user = userEvent.setup();
      const { props } = renderPanel({ users, selectedUserIds: new Set(["u1"]) });
      // May have group clear too — get the users clear button (second one if groups also shown)
      const clearButtons = screen.getAllByRole("button", { name: /clear/i });
      const usersClear = clearButtons[clearButtons.length - 1];
      await user.click(usersClear);
      expect(props.onUsersClear).toHaveBeenCalledTimes(1);
    });
  });

  describe("Search threshold", () => {
    const manyGroups = Array.from({ length: 6 }, (_, i) =>
      makeGroup({ id: `g${i}`, name: `Group ${i}` })
    );
    const fewGroups = Array.from({ length: 3 }, (_, i) =>
      makeGroup({ id: `g${i}`, name: `Group ${i}` })
    );

    it("shows search input when groups exceed SEARCH_THRESHOLD (5)", () => {
      renderPanel({ groups: manyGroups });
      expect(screen.getByPlaceholderText("Search groups...")).toBeInTheDocument();
    });

    it("does not show search input when groups are at or below threshold", () => {
      renderPanel({ groups: fewGroups });
      expect(screen.queryByPlaceholderText("Search groups...")).not.toBeInTheDocument();
    });

    it("shows search input for users when they exceed threshold", () => {
      const manyUsers = Array.from({ length: 6 }, (_, i) => ({
        user_id: `u${i}`,
        display_name: `User ${i}`,
        username: `user${i}`,
      }));
      renderPanel({ users: manyUsers });
      expect(screen.getByPlaceholderText("Search users...")).toBeInTheDocument();
    });

    it("filters groups when searching", async () => {
      const user = userEvent.setup();
      renderPanel({ groups: manyGroups });
      const searchInput = screen.getByPlaceholderText("Search groups...");
      await user.type(searchInput, "Group 0");
      expect(screen.getByText("Group 0")).toBeInTheDocument();
      expect(screen.queryByText("Group 3")).not.toBeInTheDocument();
    });
  });

  describe("Empty sections", () => {
    it("does not render group or user section headers when both are empty", () => {
      renderPanel({ groups: [], users: [] });
      expect(screen.queryByText("Groups")).not.toBeInTheDocument();
      expect(screen.queryByText("Users")).not.toBeInTheDocument();
    });
  });
});

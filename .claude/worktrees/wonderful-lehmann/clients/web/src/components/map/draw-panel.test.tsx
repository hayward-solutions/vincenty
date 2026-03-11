import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { DrawPanel } from "./draw-panel";
import type { DrawMode, DrawStyle, CompletedShape } from "./draw-tool";
import type { Group, DrawingResponse, DrawingShareInfo } from "@/types/api";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const defaultStyle: DrawStyle = {
  stroke: "#ef4444",
  fill: "transparent",
  strokeWidth: 2,
};

const mockGroup: Group = {
  id: "group-1",
  name: "Test Group",
  description: "",
  marker_icon: "default",
  marker_color: "#000",
  created_by: "user-1",
  member_count: 2,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

const mockDrawing: DrawingResponse = {
  id: "drawing-1",
  owner_id: "user-1",
  username: "testuser",
  display_name: "Test User",
  name: "My Drawing",
  geojson: { type: "FeatureCollection", features: [] },
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

const sharedDrawing: DrawingResponse = {
  ...mockDrawing,
  id: "drawing-shared",
  owner_id: "user-2",
  username: "alice",
  display_name: "Alice",
  name: "Alice Drawing",
};

function makeShape(type = "line"): CompletedShape {
  return {
    feature: {
      type: "Feature",
      geometry: { type: "LineString", coordinates: [[0, 0], [1, 1]] },
      properties: { shapeType: type, stroke: "#ef4444" },
    },
  } as CompletedShape;
}

function renderPanel(
  overrides: Partial<React.ComponentProps<typeof DrawPanel>> = {}
) {
  const props: React.ComponentProps<typeof DrawPanel> = {
    mode: "line" as DrawMode,
    onModeChange: vi.fn(),
    style: defaultStyle,
    onStyleChange: vi.fn(),
    shapes: [],
    onRemoveShape: vi.fn(),
    onClearShapes: vi.fn(),
    drawingName: "",
    onDrawingNameChange: vi.fn(),
    onSave: vi.fn(),
    isSaving: false,
    groups: [mockGroup],
    onShare: vi.fn(),
    isSharing: false,
    savedDrawingId: null,
    onClose: vi.fn(),
    ownDrawings: [],
    sharedDrawings: [],
    visibleDrawingIds: new Set<string>(),
    onDrawingToggle: vi.fn(),
    onDrawingDelete: vi.fn(),
    onDrawingShare: vi.fn(),
    onDrawingUnshare: vi.fn(),
    managingDrawingId: null,
    onManagingDrawingChange: vi.fn(),
    drawingShares: [],
    drawingSharesLoading: false,
    ...overrides,
  };
  return { ...render(<DrawPanel {...props} />), props };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("DrawPanel", () => {
  it("renders the Draw header", () => {
    renderPanel();
    expect(screen.getByText("Draw")).toBeInTheDocument();
  });

  it("calls onClose when close button is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel();
    // Close button is the ghost icon-sm button with X icon next to Draw heading
    const closeBtn = screen.getByText("Draw")
      .closest("div")!
      .querySelector('button[data-variant="ghost"]')!;
    await user.click(closeBtn);
    expect(props.onClose).toHaveBeenCalled();
  });

  // -----------------------------------------------------------------------
  // Mode selector
  // -----------------------------------------------------------------------

  it("renders mode selector buttons (Line, Circle, Rect)", () => {
    renderPanel();
    expect(screen.getByText("Line")).toBeInTheDocument();
    expect(screen.getByText("Circle")).toBeInTheDocument();
    expect(screen.getByText("Rect")).toBeInTheDocument();
  });

  it("calls onModeChange when a mode button is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel();
    await user.click(screen.getByText("Circle"));
    expect(props.onModeChange).toHaveBeenCalledWith("circle");
  });

  // -----------------------------------------------------------------------
  // Instructions
  // -----------------------------------------------------------------------

  it("shows line instructions when mode is line and no shapes", () => {
    renderPanel({ mode: "line" });
    expect(
      screen.getByText(/click to place points/i)
    ).toBeInTheDocument();
  });

  it("shows circle instructions when mode is circle", () => {
    renderPanel({ mode: "circle" });
    expect(
      screen.getByText(/click to place centre/i)
    ).toBeInTheDocument();
  });

  it("shows rectangle instructions when mode is rectangle", () => {
    renderPanel({ mode: "rectangle" });
    expect(
      screen.getByText(/click for first corner/i)
    ).toBeInTheDocument();
  });

  it("does not show instructions when shapes exist", () => {
    renderPanel({ shapes: [makeShape()] });
    expect(
      screen.queryByText(/click to place points/i)
    ).not.toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Stroke & fill color selectors
  // -----------------------------------------------------------------------

  it("renders stroke and fill color sections", () => {
    renderPanel();
    expect(screen.getByText("Stroke")).toBeInTheDocument();
    expect(screen.getByText("Fill")).toBeInTheDocument();
  });

  it("calls onStyleChange when a stroke color is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel();
    // #3b82f6 appears in both stroke and fill — use getAllByTitle and pick the first (stroke)
    const blueBtns = screen.getAllByTitle("#3b82f6");
    await user.click(blueBtns[0]);
    expect(props.onStyleChange).toHaveBeenCalledWith(
      expect.objectContaining({ stroke: "#3b82f6" })
    );
  });

  it("calls onStyleChange when a fill color is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel();
    const noFillBtn = screen.getByTitle("No fill");
    await user.click(noFillBtn);
    expect(props.onStyleChange).toHaveBeenCalledWith(
      expect.objectContaining({ fill: "transparent" })
    );
  });

  // -----------------------------------------------------------------------
  // Shapes list
  // -----------------------------------------------------------------------

  it("renders shape list with count when shapes exist", () => {
    renderPanel({ shapes: [makeShape(), makeShape("circle")] });
    expect(screen.getByText("Shapes (2)")).toBeInTheDocument();
  });

  it("shows shape type from properties", () => {
    renderPanel({ shapes: [makeShape("circle")] });
    expect(screen.getByText("circle")).toBeInTheDocument();
  });

  it("calls onRemoveShape when trash button on shape is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel({ shapes: [makeShape()] });
    // The shape row has a button with icon-sm variant containing Trash2
    const shapeRow = screen.getByText("line").closest("div")!.parentElement!;
    const removeBtn = shapeRow.querySelector('button[data-size="icon-sm"]')!;
    await user.click(removeBtn);
    expect(props.onRemoveShape).toHaveBeenCalledWith(0);
  });

  it("calls onClearShapes when Clear all is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel({ shapes: [makeShape()] });
    await user.click(screen.getByText("Clear all"));
    expect(props.onClearShapes).toHaveBeenCalled();
  });

  // -----------------------------------------------------------------------
  // Name + Save
  // -----------------------------------------------------------------------

  it("shows name input and save button when shapes exist", () => {
    renderPanel({ shapes: [makeShape()] });
    expect(
      screen.getByPlaceholderText("Drawing name")
    ).toBeInTheDocument();
    expect(screen.getByText("Save")).toBeInTheDocument();
  });

  it("disables Save when drawing name is empty", () => {
    renderPanel({ shapes: [makeShape()], drawingName: "" });
    const saveBtn = screen.getByText("Save").closest("button");
    expect(saveBtn).toBeDisabled();
  });

  it("enables Save when drawing name has content", () => {
    renderPanel({ shapes: [makeShape()], drawingName: "My Drawing" });
    const saveBtn = screen.getByText("Save").closest("button");
    expect(saveBtn).toBeEnabled();
  });

  it("shows Update instead of Save when savedDrawingId exists", () => {
    renderPanel({
      shapes: [makeShape()],
      drawingName: "My Drawing",
      savedDrawingId: "d-1",
    });
    expect(screen.getByText("Update")).toBeInTheDocument();
  });

  it("calls onSave when Save is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel({
      shapes: [makeShape()],
      drawingName: "Test",
    });
    await user.click(screen.getByText("Save"));
    expect(props.onSave).toHaveBeenCalled();
  });

  it("shows Share button only when savedDrawingId exists", () => {
    renderPanel({ shapes: [makeShape()], drawingName: "Test" });
    expect(screen.queryByText("Share")).not.toBeInTheDocument();

    renderPanel({
      shapes: [makeShape()],
      drawingName: "Test",
      savedDrawingId: "d-1",
    });
    expect(screen.getByText("Share")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Share list
  // -----------------------------------------------------------------------

  it("toggles share group list on Share click", async () => {
    const user = userEvent.setup();
    renderPanel({
      shapes: [makeShape()],
      drawingName: "Test",
      savedDrawingId: "d-1",
    });

    await user.click(screen.getByText("Share"));
    expect(screen.getByText("Share to group")).toBeInTheDocument();
    expect(screen.getByText("Test Group")).toBeInTheDocument();
  });

  it("calls onShare with group info when group is selected", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel({
      shapes: [makeShape()],
      drawingName: "Test",
      savedDrawingId: "d-1",
    });

    await user.click(screen.getByText("Share"));
    await user.click(screen.getByText("Test Group"));

    expect(props.onShare).toHaveBeenCalledWith({
      type: "group",
      id: "group-1",
      name: "Test Group",
    });
  });

  it("shows 'No groups available' when groups is empty", async () => {
    const user = userEvent.setup();
    renderPanel({
      shapes: [makeShape()],
      drawingName: "Test",
      savedDrawingId: "d-1",
      groups: [],
    });

    await user.click(screen.getByText("Share"));
    expect(screen.getByText("No groups available.")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Saved Drawings section
  // -----------------------------------------------------------------------

  it("does not show saved drawings section when none exist", () => {
    renderPanel();
    expect(screen.queryByText("Saved Drawings")).not.toBeInTheDocument();
  });

  it("shows own drawings with visibility toggles", () => {
    renderPanel({ ownDrawings: [mockDrawing] });
    expect(screen.getByText("Saved Drawings")).toBeInTheDocument();
    expect(screen.getByText("Mine")).toBeInTheDocument();
    expect(screen.getByText("My Drawing")).toBeInTheDocument();
  });

  it("calls onDrawingToggle when checkbox is toggled", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel({ ownDrawings: [mockDrawing] });
    const checkbox = screen.getAllByRole("checkbox")[0];
    await user.click(checkbox);
    expect(props.onDrawingToggle).toHaveBeenCalledWith("drawing-1");
  });

  it("calls onDrawingDelete when delete button is clicked", async () => {
    const user = userEvent.setup();
    const { props } = renderPanel({ ownDrawings: [mockDrawing] });
    const deleteBtn = screen.getByTitle("Delete drawing");
    await user.click(deleteBtn);
    expect(props.onDrawingDelete).toHaveBeenCalledWith("drawing-1");
  });

  it("shows shared drawings with owner name", () => {
    renderPanel({ sharedDrawings: [sharedDrawing] });
    expect(screen.getByText("Shared with me")).toBeInTheDocument();
    expect(screen.getByText(/Alice Drawing/)).toBeInTheDocument();
    // Owner name in parentheses — use getAllByText since it appears in nested spans
    const aliceTexts = screen.getAllByText(/Alice/);
    expect(aliceTexts.length).toBeGreaterThanOrEqual(1);
  });

  // -----------------------------------------------------------------------
  // Manage shares popover
  // -----------------------------------------------------------------------

  it("shows manage shares popover content when managingDrawingId matches", () => {
    const share: DrawingShareInfo = {
      type: "group",
      id: "group-1",
      name: "Test Group",
      shared_at: "2025-01-01T00:00:00Z",
      message_id: "msg-1",
    };
    renderPanel({
      ownDrawings: [mockDrawing],
      managingDrawingId: "drawing-1",
      drawingShares: [share],
    });
    expect(screen.getByText("Shared with")).toBeInTheDocument();
  });

  it("shows 'Not shared yet' when drawing has no shares", () => {
    renderPanel({
      ownDrawings: [mockDrawing],
      managingDrawingId: "drawing-1",
      drawingShares: [],
    });
    expect(screen.getByText("Not shared yet")).toBeInTheDocument();
  });

  it("shows loading state for shares", () => {
    renderPanel({
      ownDrawings: [mockDrawing],
      managingDrawingId: "drawing-1",
      drawingSharesLoading: true,
    });
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });
});

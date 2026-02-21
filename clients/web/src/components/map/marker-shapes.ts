/**
 * Predefined marker shape definitions for group map markers.
 *
 * Each shape is defined as an SVG path that fits within a 24x24 viewBox.
 * Shapes are filled with the group's marker_color and rendered as inline SVGs
 * inside MapLibre marker DOM elements.
 */

export type MarkerShape =
  | "circle"
  | "square"
  | "triangle"
  | "diamond"
  | "star"
  | "crosshair"
  | "pentagon"
  | "hexagon"
  | "arrow"
  | "plus";

interface ShapeDefinition {
  label: string;
  /** SVG path d attribute, fitting a 24x24 viewBox */
  path: string;
}

/**
 * All available marker shapes mapped by name.
 * Paths are designed for a 24x24 viewBox centered at (12,12).
 */
export const MARKER_SHAPES: Record<MarkerShape, ShapeDefinition> = {
  circle: {
    label: "Circle",
    path: "M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Z",
  },
  square: {
    label: "Square",
    path: "M3 3h18v18H3Z",
  },
  triangle: {
    label: "Triangle",
    path: "M12 2L22 21H2Z",
  },
  diamond: {
    label: "Diamond",
    path: "M12 1L23 12 12 23 1 12Z",
  },
  star: {
    label: "Star",
    path: "M12 1l3.09 6.26L22 8.27l-5 4.87 1.18 6.88L12 16.77l-6.18 3.25L7 13.14 2 8.27l6.91-1.01Z",
  },
  crosshair: {
    label: "Crosshair",
    path: "M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Zm0 3a7 7 0 1 1 0 14 7 7 0 0 1 0-14Zm-0.5 3v4H7.5v1h4v4h1v-4h4v-1h-4V8Z",
  },
  pentagon: {
    label: "Pentagon",
    path: "M12 1l10.5 7.6-4 11.8H5.5l-4-11.8Z",
  },
  hexagon: {
    label: "Hexagon",
    path: "M12 2l9 5v10l-9 5-9-5V7Z",
  },
  arrow: {
    label: "Arrow",
    path: "M12 1L22 14H16V23H8V14H2Z",
  },
  plus: {
    label: "Plus",
    path: "M8 2h8v6h6v8h-6v6H8v-6H2v-8h6Z",
  },
};

/** Ordered list of shape names for use in picker UIs. */
export const AVAILABLE_SHAPES: MarkerShape[] = [
  "circle",
  "square",
  "triangle",
  "diamond",
  "star",
  "crosshair",
  "pentagon",
  "hexagon",
  "arrow",
  "plus",
];

/** Default preset color palette for the color picker. */
export const PRESET_COLORS = [
  "#3b82f6", // blue
  "#ef4444", // red
  "#22c55e", // green
  "#f59e0b", // amber
  "#8b5cf6", // violet
  "#ec4899", // pink
  "#06b6d4", // cyan
  "#f97316", // orange
  "#64748b", // slate
  "#000000", // black
];

/**
 * Creates an SVG element for a marker shape.
 * Returns an HTMLElement (SVG) that can be inserted into a MapLibre marker.
 */
export function createMarkerSVG(
  shape: string,
  color: string,
  size: number = 18
): SVGSVGElement {
  const def = MARKER_SHAPES[shape as MarkerShape] ?? MARKER_SHAPES.circle;

  const svgNS = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(svgNS, "svg");
  svg.setAttribute("xmlns", svgNS);
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("width", String(size));
  svg.setAttribute("height", String(size));
  svg.style.display = "block";

  const path = document.createElementNS(svgNS, "path");
  path.setAttribute("d", def.path);
  path.setAttribute("fill", color);
  path.setAttribute("stroke", "white");
  path.setAttribute("stroke-width", "1.5");
  path.setAttribute("stroke-linejoin", "round");

  svg.appendChild(path);
  return svg;
}

/**
 * Creates an inline SVG string (for use in React dangerouslySetInnerHTML or static HTML).
 */
export function markerSVGString(
  shape: string,
  color: string,
  size: number = 18
): string {
  const def = MARKER_SHAPES[shape as MarkerShape] ?? MARKER_SHAPES.circle;
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="${size}" height="${size}" style="display:block"><path d="${def.path}" fill="${color}" stroke="white" stroke-width="1.5" stroke-linejoin="round"/></svg>`;
}

import {
  MARKER_SHAPES,
  AVAILABLE_SHAPES,
  PRESET_COLORS,
  createMarkerSVG,
  markerSVGString,
} from "./marker-shapes";

describe("MARKER_SHAPES", () => {
  it("has an entry for every shape in AVAILABLE_SHAPES", () => {
    for (const shape of AVAILABLE_SHAPES) {
      expect(MARKER_SHAPES[shape]).toBeDefined();
      expect(MARKER_SHAPES[shape].label).toBeTruthy();
      expect(MARKER_SHAPES[shape].path).toBeTruthy();
    }
  });

  it("every shape has a non-empty label and path", () => {
    for (const [name, def] of Object.entries(MARKER_SHAPES)) {
      expect(def.label).toBeTruthy();
      expect(def.path).toBeTruthy();
    }
  });
});

describe("AVAILABLE_SHAPES", () => {
  it("contains 10 shapes", () => {
    expect(AVAILABLE_SHAPES).toHaveLength(10);
  });

  it("has no duplicates", () => {
    const unique = new Set(AVAILABLE_SHAPES);
    expect(unique.size).toBe(AVAILABLE_SHAPES.length);
  });
});

describe("PRESET_COLORS", () => {
  it("has 10 items", () => {
    expect(PRESET_COLORS).toHaveLength(10);
  });

  it("all items are valid hex color strings", () => {
    const hexRegex = /^#[0-9a-fA-F]{6}$/;
    for (const color of PRESET_COLORS) {
      expect(color).toMatch(hexRegex);
    }
  });
});

describe("markerSVGString", () => {
  it("returns an SVG string containing the specified color", () => {
    const svg = markerSVGString("circle", "#ff0000");
    expect(svg).toContain('fill="#ff0000"');
  });

  it("returns an SVG with the correct shape path", () => {
    const svg = markerSVGString("square", "#000000");
    expect(svg).toContain(MARKER_SHAPES.square.path);
  });

  it("defaults to size 18", () => {
    const svg = markerSVGString("circle", "#000000");
    expect(svg).toContain('width="18"');
    expect(svg).toContain('height="18"');
  });

  it("accepts a custom size", () => {
    const svg = markerSVGString("circle", "#000000", 32);
    expect(svg).toContain('width="32"');
    expect(svg).toContain('height="32"');
  });

  it("uses viewBox 0 0 24 24", () => {
    const svg = markerSVGString("circle", "#000000");
    expect(svg).toContain('viewBox="0 0 24 24"');
  });

  it("includes white stroke", () => {
    const svg = markerSVGString("circle", "#000000");
    expect(svg).toContain('stroke="white"');
    expect(svg).toContain('stroke-width="1.5"');
  });

  it("falls back to circle for unknown shapes", () => {
    const svg = markerSVGString("nonexistent_shape", "#123456");
    expect(svg).toContain(MARKER_SHAPES.circle.path);
    expect(svg).toContain('fill="#123456"');
  });

  it("produces valid SVG structure", () => {
    const svg = markerSVGString("triangle", "#abcdef");
    expect(svg).toMatch(/^<svg[^>]*>.*<\/svg>$/);
    expect(svg).toContain("<path");
  });
});

describe("createMarkerSVG", () => {
  it("returns an SVGSVGElement", () => {
    const el = createMarkerSVG("circle", "#ff0000");
    expect(el).toBeInstanceOf(SVGSVGElement);
  });

  it("has correct width and height attributes (default size)", () => {
    const el = createMarkerSVG("circle", "#ff0000");
    expect(el.getAttribute("width")).toBe("18");
    expect(el.getAttribute("height")).toBe("18");
  });

  it("applies custom size", () => {
    const el = createMarkerSVG("circle", "#ff0000", 24);
    expect(el.getAttribute("width")).toBe("24");
    expect(el.getAttribute("height")).toBe("24");
  });

  it("has viewBox 0 0 24 24", () => {
    const el = createMarkerSVG("circle", "#ff0000");
    expect(el.getAttribute("viewBox")).toBe("0 0 24 24");
  });

  it("contains a path child with correct fill color", () => {
    const el = createMarkerSVG("diamond", "#abcdef");
    const path = el.querySelector("path");
    expect(path).not.toBeNull();
    expect(path?.getAttribute("fill")).toBe("#abcdef");
  });

  it("path has the correct d attribute for the shape", () => {
    const el = createMarkerSVG("star", "#000000");
    const path = el.querySelector("path");
    expect(path?.getAttribute("d")).toBe(MARKER_SHAPES.star.path);
  });

  it("path has white stroke", () => {
    const el = createMarkerSVG("circle", "#000000");
    const path = el.querySelector("path");
    expect(path?.getAttribute("stroke")).toBe("white");
    expect(path?.getAttribute("stroke-width")).toBe("1.5");
  });

  it("falls back to circle for unknown shapes", () => {
    const el = createMarkerSVG("bogus_shape", "#000000");
    const path = el.querySelector("path");
    expect(path?.getAttribute("d")).toBe(MARKER_SHAPES.circle.path);
  });

  it("sets display:block style", () => {
    const el = createMarkerSVG("circle", "#000000");
    expect(el.style.display).toBe("block");
  });
});

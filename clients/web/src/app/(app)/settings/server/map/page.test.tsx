import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import MapSettingsPage from "./page";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  mapConfigs: [
    {
      id: "mc-1",
      name: "OpenStreetMap",
      source_type: "raster",
      tile_url: "https://tile.osm.org/{z}/{x}/{y}.png",
      min_zoom: 0,
      max_zoom: 19,
      is_default: true,
      is_builtin: true,
      is_enabled: true,
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
    {
      id: "mc-2",
      name: "Custom Map",
      source_type: "remote",
      tile_url: "https://custom.tiles/{z}/{x}/{y}.png",
      min_zoom: 0,
      max_zoom: 18,
      is_default: false,
      is_builtin: false,
      is_enabled: true,
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
  ],
  terrainConfigs: [
    {
      id: "tc-1",
      name: "Default Terrain",
      source_type: "raster-dem",
      terrain_url: "https://example.com/terrain/{z}/{x}/{y}.png",
      terrain_encoding: "terrarium",
      is_default: true,
      is_builtin: true,
      is_enabled: true,
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
  ],
  refetchMap: vi.fn(),
  refetchTerrain: vi.fn(),
  createMapConfig: vi.fn().mockResolvedValue({}),
  updateMapConfig: vi.fn().mockResolvedValue({}),
  deleteMapConfig: vi.fn().mockResolvedValue(undefined),
  createTerrainConfig: vi.fn().mockResolvedValue({}),
  updateTerrainConfig: vi.fn().mockResolvedValue({}),
  deleteTerrainConfig: vi.fn().mockResolvedValue(undefined),
  serverSettings: {
    mfa_required: false,
    mapbox_access_token: "pk.test123",
    google_maps_api_key: "",
  },
  updateSettings: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@/lib/hooks/use-map-settings", () => ({
  useMapConfigs: () => ({
    configs: mocks.mapConfigs,
    isLoading: false,
    refetch: mocks.refetchMap,
  }),
  useCreateMapConfig: () => ({
    createMapConfig: mocks.createMapConfig,
    isLoading: false,
  }),
  useUpdateMapConfig: () => ({
    updateMapConfig: mocks.updateMapConfig,
    isLoading: false,
  }),
  useDeleteMapConfig: () => ({
    deleteMapConfig: mocks.deleteMapConfig,
  }),
  useTerrainConfigs: () => ({
    configs: mocks.terrainConfigs,
    isLoading: false,
    refetch: mocks.refetchTerrain,
  }),
  useCreateTerrainConfig: () => ({
    createTerrainConfig: mocks.createTerrainConfig,
    isLoading: false,
  }),
  useUpdateTerrainConfig: () => ({
    updateTerrainConfig: mocks.updateTerrainConfig,
    isLoading: false,
  }),
  useDeleteTerrainConfig: () => ({
    deleteTerrainConfig: mocks.deleteTerrainConfig,
  }),
}));

vi.mock("@/lib/hooks/use-mfa", () => ({
  useServerSettings: () => ({
    settings: mocks.serverSettings,
    isLoading: false,
    update: mocks.updateSettings,
  }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/lib/api", () => ({
  ApiError: class ApiError extends Error {
    constructor(message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("MapSettingsPage", () => {
  it('renders "API Keys" heading', () => {
    render(<MapSettingsPage />);
    expect(screen.getByText("API Keys")).toBeInTheDocument();
  });

  it('renders "Tile Configs" heading and "Create Config" button', () => {
    render(<MapSettingsPage />);
    expect(screen.getByText("Tile Configs")).toBeInTheDocument();
    // There are two "Create Config" buttons (tile + terrain), grab all
    const buttons = screen.getAllByRole("button", { name: /create config/i });
    expect(buttons.length).toBe(2);
  });

  it('renders "Terrain Configs" heading', () => {
    render(<MapSettingsPage />);
    expect(screen.getByText("Terrain Configs")).toBeInTheDocument();
  });

  it("shows tile config names in the table", () => {
    render(<MapSettingsPage />);
    expect(screen.getByText("OpenStreetMap")).toBeInTheDocument();
    expect(screen.getByText("Custom Map")).toBeInTheDocument();
  });

  it('shows "Built-in" badge for builtin configs', () => {
    render(<MapSettingsPage />);
    // Both OpenStreetMap (tile) and Default Terrain (terrain) are builtin
    const builtInBadges = screen.getAllByText("Built-in");
    expect(builtInBadges.length).toBe(2);
  });

  it('shows "Default" badge for default configs', () => {
    render(<MapSettingsPage />);
    // OpenStreetMap (tile) and Default Terrain (terrain) are both default
    const defaultBadges = screen.getAllByText("Default");
    expect(defaultBadges.length).toBe(2);
  });

  it("shows terrain config name and encoding", () => {
    render(<MapSettingsPage />);
    expect(screen.getByText("Default Terrain")).toBeInTheDocument();
    expect(screen.getByText("terrarium")).toBeInTheDocument();
  });

  it("shows MapBox token input as password type", () => {
    render(<MapSettingsPage />);
    const input = screen.getByLabelText("MapBox Access Token");
    expect(input).toHaveAttribute("type", "password");
  });

  it("toggles MapBox token visibility on show/hide button click", async () => {
    const user = userEvent.setup();
    render(<MapSettingsPage />);

    const toggleBtn = screen.getByRole("button", { name: "Show token" });
    expect(toggleBtn).toBeInTheDocument();

    await user.click(toggleBtn);

    expect(
      screen.getByRole("button", { name: "Hide token" })
    ).toBeInTheDocument();
    expect(screen.getByLabelText("MapBox Access Token")).toHaveAttribute(
      "type",
      "text"
    );
  });

  it("calls update with token values when Save is clicked", async () => {
    const user = userEvent.setup();
    render(<MapSettingsPage />);

    const saveBtn = screen.getByRole("button", { name: "Save" });
    await user.click(saveBtn);

    await waitFor(() => {
      expect(mocks.updateSettings).toHaveBeenCalledWith({
        mapbox_access_token: "pk.test123",
        google_maps_api_key: "",
      });
    });
  });

  it("shows tile URL in the table", () => {
    render(<MapSettingsPage />);
    expect(
      screen.getByText("https://tile.osm.org/{z}/{x}/{y}.png")
    ).toBeInTheDocument();
    expect(
      screen.getByText("https://custom.tiles/{z}/{x}/{y}.png")
    ).toBeInTheDocument();
  });

  it("shows zoom range in the table", () => {
    render(<MapSettingsPage />);
    expect(screen.getByText("0-19")).toBeInTheDocument();
    expect(screen.getByText("0-18")).toBeInTheDocument();
  });
});

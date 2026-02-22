package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// MapConfigService handles map configuration business logic.
type MapConfigService struct {
	repo         *repository.MapConfigRepository
	terrainRepo  *repository.TerrainConfigRepository
	settingsRepo *repository.ServerSettingsRepository
	mapDefaults  config.MapConfig
}

// NewMapConfigService creates a new MapConfigService.
func NewMapConfigService(repo *repository.MapConfigRepository, terrainRepo *repository.TerrainConfigRepository, settingsRepo *repository.ServerSettingsRepository, mapDefaults config.MapConfig) *MapConfigService {
	return &MapConfigService{repo: repo, terrainRepo: terrainRepo, settingsRepo: settingsRepo, mapDefaults: mapDefaults}
}

// BootstrapMapConfigs seeds the built-in map configurations if none exist yet.
// This follows the same pattern as BootstrapAdmin in auth_service.go.
func (s *MapConfigService) BootstrapMapConfigs(ctx context.Context) error {
	count, err := s.repo.CountBuiltin(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Debug("built-in map configs already exist, skipping bootstrap")
		return nil
	}

	osmURL := "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
	satURL := "https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}"

	builtins := []model.MapConfig{
		{
			Name:       "OpenStreetMap",
			SourceType: "remote",
			TileURL:    &osmURL,
			MinZoom:    0,
			MaxZoom:    19,
			IsDefault:  true,
			IsBuiltin:  true,
			IsEnabled:  true,
		},
		{
			Name:       "Satellite (ESRI)",
			SourceType: "remote",
			TileURL:    &satURL,
			MinZoom:    0,
			MaxZoom:    18,
			IsDefault:  false,
			IsBuiltin:  true,
			IsEnabled:  true,
		},
	}

	for i := range builtins {
		if err := s.repo.Create(ctx, &builtins[i]); err != nil {
			return err
		}
		slog.Info("bootstrap map config created", "name", builtins[i].Name)
	}

	return nil
}

// Create creates a new map configuration.
func (s *MapConfigService) Create(ctx context.Context, req *model.CreateMapConfigRequest, createdBy uuid.UUID) (*model.MapConfig, error) {
	var tileURL *string
	if req.TileURL != "" {
		tileURL = &req.TileURL
	}

	minZoom := 0
	if req.MinZoom != nil {
		minZoom = *req.MinZoom
	}
	maxZoom := 18
	if req.MaxZoom != nil {
		maxZoom = *req.MaxZoom
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	// If setting as default, clear existing default first
	if isDefault {
		if err := s.repo.ClearDefault(ctx); err != nil {
			return nil, err
		}
	}

	mc := &model.MapConfig{
		Name:       req.Name,
		SourceType: req.SourceType,
		TileURL:    tileURL,
		StyleJSON:  req.StyleJSON,
		MinZoom:    minZoom,
		MaxZoom:    maxZoom,
		IsDefault:  isDefault,
		IsBuiltin:  false,
		IsEnabled:  true,
		CreatedBy:  &createdBy,
	}

	if err := s.repo.Create(ctx, mc); err != nil {
		return nil, err
	}

	return mc, nil
}

// GetByID retrieves a map configuration by ID.
func (s *MapConfigService) GetByID(ctx context.Context, id uuid.UUID) (*model.MapConfig, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all map configurations.
func (s *MapConfigService) List(ctx context.Context) ([]model.MapConfig, error) {
	return s.repo.List(ctx)
}

// GetSettings returns the map settings for the client, combining the default
// DB configs (tile and terrain) with the server-side environment defaults.
func (s *MapConfigService) GetSettings(ctx context.Context) (*model.MapSettingsResponse, error) {
	configs, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Start with empty defaults — the seeded built-in configs (which cannot
	// be deleted) will override these via the DB default lookup below.
	resp := &model.MapSettingsResponse{
		CenterLat:       s.mapDefaults.DefaultCenterLat,
		CenterLng:       s.mapDefaults.DefaultCenterLng,
		Zoom:            s.mapDefaults.DefaultZoom,
		MinZoom:         0,
		MaxZoom:         18,
		TerrainEncoding: "terrarium",
		Configs:         make([]model.MapConfigResponse, 0, len(configs)),
	}

	// Override tile settings with default+enabled DB map config if one exists
	for _, mc := range configs {
		resp.Configs = append(resp.Configs, mc.ToResponse())
		if mc.IsDefault && mc.IsEnabled {
			if mc.TileURL != nil {
				resp.TileURL = *mc.TileURL
			}
			resp.StyleJSON = mc.StyleJSON
			resp.MinZoom = mc.MinZoom
			resp.MaxZoom = mc.MaxZoom
		}
	}

	// Override terrain settings with default+enabled DB terrain config if one exists
	defaultTerrain, err := s.terrainRepo.GetDefault(ctx)
	if err != nil {
		return nil, err
	}
	if defaultTerrain != nil {
		resp.TerrainURL = defaultTerrain.TerrainURL
		resp.TerrainEncoding = defaultTerrain.TerrainEncoding
	}

	// Include map provider API keys from server settings
	resp.MapboxAccessToken = s.getSettingValue(ctx, "mapbox_access_token")
	resp.GoogleMapsApiKey = s.getSettingValue(ctx, "google_maps_api_key")

	return resp, nil
}

// getSettingValue returns the value for a server setting key, or an empty
// string if the key does not exist.
func (s *MapConfigService) getSettingValue(ctx context.Context, key string) string {
	setting, err := s.settingsRepo.Get(ctx, key)
	if err != nil {
		return ""
	}
	return setting.Value
}

// Update modifies a map configuration.
func (s *MapConfigService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateMapConfigRequest) (*model.MapConfig, error) {
	mc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Built-in configs: only is_default and is_enabled can be changed.
	if mc.IsBuiltin {
		if req.Name != nil || req.SourceType != nil || req.TileURL != nil ||
			req.StyleJSON != nil || req.MinZoom != nil || req.MaxZoom != nil {
			return nil, model.ErrValidation("built-in config fields are read-only")
		}

		// Cannot disable the current default — must change default first.
		if req.IsEnabled != nil && !*req.IsEnabled && mc.IsDefault {
			return nil, model.ErrValidation("cannot disable the current default config; change the default first")
		}

		if req.IsDefault != nil && *req.IsDefault != mc.IsDefault {
			if *req.IsDefault {
				if err := s.repo.ClearDefault(ctx); err != nil {
					return nil, err
				}
			}
			mc.IsDefault = *req.IsDefault
		}
		if req.IsEnabled != nil {
			mc.IsEnabled = *req.IsEnabled
		}

		if err := s.repo.Update(ctx, mc); err != nil {
			return nil, err
		}
		return mc, nil
	}

	// User-created configs: all fields can be changed.
	if req.Name != nil {
		if *req.Name == "" {
			return nil, model.ErrValidation("name cannot be empty")
		}
		if len(*req.Name) > 255 {
			return nil, model.ErrValidation("name must be 255 characters or less")
		}
		mc.Name = *req.Name
	}
	if req.SourceType != nil {
		if *req.SourceType != "remote" && *req.SourceType != "local" && *req.SourceType != "style" {
			return nil, model.ErrValidation("source_type must be 'remote', 'local', or 'style'")
		}
		mc.SourceType = *req.SourceType
	}
	if req.TileURL != nil {
		mc.TileURL = req.TileURL
	}
	if req.StyleJSON != nil {
		mc.StyleJSON = req.StyleJSON
	}
	if req.MinZoom != nil {
		mc.MinZoom = *req.MinZoom
	}
	if req.MaxZoom != nil {
		mc.MaxZoom = *req.MaxZoom
	}
	if req.IsEnabled != nil {
		// Cannot disable the current default — must change default first.
		if !*req.IsEnabled && mc.IsDefault {
			return nil, model.ErrValidation("cannot disable the current default config; change the default first")
		}
		mc.IsEnabled = *req.IsEnabled
	}
	if req.IsDefault != nil && *req.IsDefault != mc.IsDefault {
		if *req.IsDefault {
			if err := s.repo.ClearDefault(ctx); err != nil {
				return nil, err
			}
		}
		mc.IsDefault = *req.IsDefault
	}

	if err := s.repo.Update(ctx, mc); err != nil {
		return nil, err
	}

	return mc, nil
}

// Delete removes a map configuration.
func (s *MapConfigService) Delete(ctx context.Context, id uuid.UUID) error {
	mc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if mc.IsBuiltin {
		return model.ErrValidation("built-in configs cannot be deleted")
	}

	return s.repo.Delete(ctx, id)
}

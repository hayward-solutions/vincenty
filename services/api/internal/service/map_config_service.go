package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// MapConfigService handles map configuration business logic.
type MapConfigService struct {
	repo        *repository.MapConfigRepository
	mapDefaults config.MapConfig
}

// NewMapConfigService creates a new MapConfigService.
func NewMapConfigService(repo *repository.MapConfigRepository, mapDefaults config.MapConfig) *MapConfigService {
	return &MapConfigService{repo: repo, mapDefaults: mapDefaults}
}

// Create creates a new map configuration.
func (s *MapConfigService) Create(ctx context.Context, req *model.CreateMapConfigRequest, createdBy uuid.UUID) (*model.MapConfig, error) {
	var tileURL *string
	if req.TileURL != "" {
		tileURL = &req.TileURL
	}
	var terrainURL *string
	if req.TerrainURL != "" {
		terrainURL = &req.TerrainURL
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
		Name:            req.Name,
		SourceType:      req.SourceType,
		TileURL:         tileURL,
		StyleJSON:       req.StyleJSON,
		MinZoom:         minZoom,
		MaxZoom:         maxZoom,
		TerrainURL:      terrainURL,
		TerrainEncoding: req.TerrainEncoding,
		IsDefault:       isDefault,
		CreatedBy:       &createdBy,
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
// DB config (if any) with the server-side environment defaults.
func (s *MapConfigService) GetSettings(ctx context.Context) (*model.MapSettingsResponse, error) {
	configs, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Start with environment defaults
	resp := &model.MapSettingsResponse{
		TileURL:         s.mapDefaults.DefaultTileURL,
		CenterLat:       s.mapDefaults.DefaultCenterLat,
		CenterLng:       s.mapDefaults.DefaultCenterLng,
		Zoom:            s.mapDefaults.DefaultZoom,
		MinZoom:         0,
		MaxZoom:         18,
		TerrainURL:      s.mapDefaults.DefaultTerrainURL,
		TerrainEncoding: "terrarium",
		Configs:         make([]model.MapConfigResponse, 0, len(configs)),
	}

	// Override with default DB config if one exists
	for _, mc := range configs {
		resp.Configs = append(resp.Configs, mc.ToResponse())
		if mc.IsDefault {
			if mc.TileURL != nil {
				resp.TileURL = *mc.TileURL
			}
			resp.StyleJSON = mc.StyleJSON
			resp.MinZoom = mc.MinZoom
			resp.MaxZoom = mc.MaxZoom
			if mc.TerrainURL != nil {
				resp.TerrainURL = *mc.TerrainURL
			}
			if mc.TerrainEncoding != "" {
				resp.TerrainEncoding = mc.TerrainEncoding
			}
		}
	}

	return resp, nil
}

// Update modifies a map configuration.
func (s *MapConfigService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateMapConfigRequest) (*model.MapConfig, error) {
	mc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

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
	if req.TerrainURL != nil {
		mc.TerrainURL = req.TerrainURL
	}
	if req.TerrainEncoding != nil {
		if *req.TerrainEncoding != "terrarium" && *req.TerrainEncoding != "mapbox" {
			return nil, model.ErrValidation("terrain_encoding must be 'terrarium' or 'mapbox'")
		}
		mc.TerrainEncoding = *req.TerrainEncoding
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
	return s.repo.Delete(ctx, id)
}

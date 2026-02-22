package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// TerrainConfigService handles terrain configuration business logic.
type TerrainConfigService struct {
	repo        *repository.TerrainConfigRepository
	mapDefaults config.MapConfig
}

// NewTerrainConfigService creates a new TerrainConfigService.
func NewTerrainConfigService(repo *repository.TerrainConfigRepository, mapDefaults config.MapConfig) *TerrainConfigService {
	return &TerrainConfigService{repo: repo, mapDefaults: mapDefaults}
}

// Create creates a new terrain configuration.
func (s *TerrainConfigService) Create(ctx context.Context, req *model.CreateTerrainConfigRequest, createdBy uuid.UUID) (*model.TerrainConfig, error) {
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	// If setting as default, clear existing default first.
	if isDefault {
		if err := s.repo.ClearDefault(ctx); err != nil {
			return nil, err
		}
	}

	tc := &model.TerrainConfig{
		Name:            req.Name,
		SourceType:      req.SourceType,
		TerrainURL:      req.TerrainURL,
		TerrainEncoding: req.TerrainEncoding,
		IsDefault:       isDefault,
		CreatedBy:       &createdBy,
	}

	if err := s.repo.Create(ctx, tc); err != nil {
		return nil, err
	}

	return tc, nil
}

// GetByID retrieves a terrain configuration by ID.
func (s *TerrainConfigService) GetByID(ctx context.Context, id uuid.UUID) (*model.TerrainConfig, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all terrain configurations.
func (s *TerrainConfigService) List(ctx context.Context) ([]model.TerrainConfig, error) {
	return s.repo.List(ctx)
}

// Update modifies a terrain configuration.
func (s *TerrainConfigService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateTerrainConfigRequest) (*model.TerrainConfig, error) {
	tc, err := s.repo.GetByID(ctx, id)
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
		tc.Name = *req.Name
	}
	if req.SourceType != nil {
		if *req.SourceType != "remote" && *req.SourceType != "local" {
			return nil, model.ErrValidation("source_type must be 'remote' or 'local'")
		}
		tc.SourceType = *req.SourceType
	}
	if req.TerrainURL != nil {
		if *req.TerrainURL == "" {
			return nil, model.ErrValidation("terrain_url cannot be empty")
		}
		tc.TerrainURL = *req.TerrainURL
	}
	if req.TerrainEncoding != nil {
		if *req.TerrainEncoding != "terrarium" && *req.TerrainEncoding != "mapbox" {
			return nil, model.ErrValidation("terrain_encoding must be 'terrarium' or 'mapbox'")
		}
		tc.TerrainEncoding = *req.TerrainEncoding
	}
	if req.IsDefault != nil && *req.IsDefault != tc.IsDefault {
		if *req.IsDefault {
			if err := s.repo.ClearDefault(ctx); err != nil {
				return nil, err
			}
		}
		tc.IsDefault = *req.IsDefault
	}

	if err := s.repo.Update(ctx, tc); err != nil {
		return nil, err
	}

	return tc, nil
}

// Delete removes a terrain configuration.
func (s *TerrainConfigService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// GetDefaults returns the server-level environment defaults for terrain
// configuration. These are the baseline values used when no database terrain
// config is marked as default.
func (s *TerrainConfigService) GetDefaults() *model.TerrainDefaultsResponse {
	return &model.TerrainDefaultsResponse{
		TerrainURL:      s.mapDefaults.DefaultTerrainURL,
		TerrainEncoding: "terrarium",
	}
}

package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/config"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/repository"
)

// TerrainConfigService handles terrain configuration business logic.
type TerrainConfigService struct {
	repo        repository.TerrainConfigRepo
	mapDefaults config.MapConfig
}

// NewTerrainConfigService creates a new TerrainConfigService.
func NewTerrainConfigService(repo repository.TerrainConfigRepo, mapDefaults config.MapConfig) *TerrainConfigService {
	return &TerrainConfigService{repo: repo, mapDefaults: mapDefaults}
}

// BootstrapTerrainConfigs seeds the built-in terrain configurations if none exist yet.
func (s *TerrainConfigService) BootstrapTerrainConfigs(ctx context.Context) error {
	count, err := s.repo.CountBuiltin(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Debug("built-in terrain configs already exist, skipping bootstrap")
		return nil
	}

	builtins := []model.TerrainConfig{
		{
			Name:            "AWS Terrarium",
			SourceType:      "remote",
			TerrainURL:      "https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png",
			TerrainEncoding: "terrarium",
			IsDefault:       true,
			IsBuiltin:       true,
			IsEnabled:       true,
		},
	}

	for i := range builtins {
		if err := s.repo.Create(ctx, &builtins[i]); err != nil {
			return err
		}
		slog.Info("bootstrap terrain config created", "name", builtins[i].Name)
	}

	return nil
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
		IsBuiltin:       false,
		IsEnabled:       true,
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

	// Built-in configs: only is_default and is_enabled can be changed.
	if tc.IsBuiltin {
		if req.Name != nil || req.SourceType != nil || req.TerrainURL != nil || req.TerrainEncoding != nil {
			return nil, model.ErrValidation("built-in config fields are read-only")
		}

		// Cannot disable the current default — must change default first.
		if req.IsEnabled != nil && !*req.IsEnabled && tc.IsDefault {
			return nil, model.ErrValidation("cannot disable the current default config; change the default first")
		}

		if req.IsDefault != nil && *req.IsDefault != tc.IsDefault {
			if *req.IsDefault {
				if err := s.repo.ClearDefault(ctx); err != nil {
					return nil, err
				}
			}
			tc.IsDefault = *req.IsDefault
		}
		if req.IsEnabled != nil {
			tc.IsEnabled = *req.IsEnabled
		}

		if err := s.repo.Update(ctx, tc); err != nil {
			return nil, err
		}
		return tc, nil
	}

	// User-created configs: all fields can be changed.
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
	if req.IsEnabled != nil {
		// Cannot disable the current default — must change default first.
		if !*req.IsEnabled && tc.IsDefault {
			return nil, model.ErrValidation("cannot disable the current default config; change the default first")
		}
		tc.IsEnabled = *req.IsEnabled
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
	tc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if tc.IsBuiltin {
		return model.ErrValidation("built-in configs cannot be deleted")
	}

	return s.repo.Delete(ctx, id)
}

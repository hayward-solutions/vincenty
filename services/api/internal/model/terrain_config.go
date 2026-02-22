package model

import (
	"time"

	"github.com/google/uuid"
)

// TerrainConfig represents a terrain DEM tile configuration.
type TerrainConfig struct {
	ID              uuid.UUID  `json:"-"`
	Name            string     `json:"-"`
	SourceType      string     `json:"-"`
	TerrainURL      string     `json:"-"`
	TerrainEncoding string     `json:"-"`
	IsDefault       bool       `json:"-"`
	CreatedBy       *uuid.UUID `json:"-"`
	CreatedAt       time.Time  `json:"-"`
	UpdatedAt       time.Time  `json:"-"`
}

// TerrainConfigResponse is the JSON representation returned by the API.
type TerrainConfigResponse struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	SourceType      string     `json:"source_type"`
	TerrainURL      string     `json:"terrain_url"`
	TerrainEncoding string     `json:"terrain_encoding"`
	IsDefault       bool       `json:"is_default"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ToResponse converts a TerrainConfig to its API response.
func (t *TerrainConfig) ToResponse() TerrainConfigResponse {
	return TerrainConfigResponse{
		ID:              t.ID,
		Name:            t.Name,
		SourceType:      t.SourceType,
		TerrainURL:      t.TerrainURL,
		TerrainEncoding: t.TerrainEncoding,
		IsDefault:       t.IsDefault,
		CreatedBy:       t.CreatedBy,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

// CreateTerrainConfigRequest is the expected body for creating a terrain config.
type CreateTerrainConfigRequest struct {
	Name            string `json:"name"`
	SourceType      string `json:"source_type"`
	TerrainURL      string `json:"terrain_url"`
	TerrainEncoding string `json:"terrain_encoding"`
	IsDefault       *bool  `json:"is_default"`
}

// Validate checks that required fields are present.
func (r *CreateTerrainConfigRequest) Validate() error {
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	if len(r.Name) > 255 {
		return ErrValidation("name must be 255 characters or less")
	}
	if r.SourceType == "" {
		r.SourceType = "remote"
	}
	if r.SourceType != "remote" && r.SourceType != "local" {
		return ErrValidation("source_type must be 'remote' or 'local'")
	}
	if r.TerrainURL == "" {
		return ErrValidation("terrain_url is required")
	}
	if r.TerrainEncoding == "" {
		r.TerrainEncoding = "terrarium"
	}
	if r.TerrainEncoding != "terrarium" && r.TerrainEncoding != "mapbox" {
		return ErrValidation("terrain_encoding must be 'terrarium' or 'mapbox'")
	}
	return nil
}

// UpdateTerrainConfigRequest is the expected body for updating a terrain config.
type UpdateTerrainConfigRequest struct {
	Name            *string `json:"name"`
	SourceType      *string `json:"source_type"`
	TerrainURL      *string `json:"terrain_url"`
	TerrainEncoding *string `json:"terrain_encoding"`
	IsDefault       *bool   `json:"is_default"`
}

// TerrainDefaultsResponse contains the server-level environment defaults for
// terrain configuration. These are the baseline values the system falls back to
// when no database terrain config is marked as default.
type TerrainDefaultsResponse struct {
	TerrainURL      string `json:"terrain_url"`
	TerrainEncoding string `json:"terrain_encoding"`
}

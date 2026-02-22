package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MapConfig represents a map tile configuration.
type MapConfig struct {
	ID        uuid.UUID        `json:"-"`
	Name      string           `json:"-"`
	SourceType string          `json:"-"`
	TileURL   *string          `json:"-"`
	StyleJSON *json.RawMessage `json:"-"`
	MinZoom   int              `json:"-"`
	MaxZoom   int              `json:"-"`
	IsDefault bool             `json:"-"`
	CreatedBy *uuid.UUID       `json:"-"`
	CreatedAt time.Time        `json:"-"`
	UpdatedAt time.Time        `json:"-"`
}

// MapConfigResponse is the JSON representation returned by the API.
type MapConfigResponse struct {
	ID        uuid.UUID        `json:"id"`
	Name      string           `json:"name"`
	SourceType string          `json:"source_type"`
	TileURL   string           `json:"tile_url"`
	StyleJSON *json.RawMessage `json:"style_json,omitempty"`
	MinZoom   int              `json:"min_zoom"`
	MaxZoom   int              `json:"max_zoom"`
	IsDefault bool             `json:"is_default"`
	CreatedBy *uuid.UUID       `json:"created_by,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// ToResponse converts a MapConfig to its API response.
func (m *MapConfig) ToResponse() MapConfigResponse {
	tileURL := ""
	if m.TileURL != nil {
		tileURL = *m.TileURL
	}
	return MapConfigResponse{
		ID:        m.ID,
		Name:      m.Name,
		SourceType: m.SourceType,
		TileURL:   tileURL,
		StyleJSON: m.StyleJSON,
		MinZoom:   m.MinZoom,
		MaxZoom:   m.MaxZoom,
		IsDefault: m.IsDefault,
		CreatedBy: m.CreatedBy,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// CreateMapConfigRequest is the expected body for creating a map config.
type CreateMapConfigRequest struct {
	Name       string           `json:"name"`
	SourceType string           `json:"source_type"`
	TileURL    string           `json:"tile_url"`
	StyleJSON  *json.RawMessage `json:"style_json"`
	MinZoom    *int             `json:"min_zoom"`
	MaxZoom    *int             `json:"max_zoom"`
	IsDefault  *bool            `json:"is_default"`
}

// Validate checks that required fields are present.
func (r *CreateMapConfigRequest) Validate() error {
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	if len(r.Name) > 255 {
		return ErrValidation("name must be 255 characters or less")
	}
	if r.SourceType == "" {
		r.SourceType = "remote"
	}
	if r.SourceType != "remote" && r.SourceType != "local" && r.SourceType != "style" {
		return ErrValidation("source_type must be 'remote', 'local', or 'style'")
	}
	if r.SourceType != "style" && r.TileURL == "" {
		return ErrValidation("tile_url is required for remote and local source types")
	}
	if r.MinZoom != nil && (*r.MinZoom < 0 || *r.MinZoom > 24) {
		return ErrValidation("min_zoom must be between 0 and 24")
	}
	if r.MaxZoom != nil && (*r.MaxZoom < 0 || *r.MaxZoom > 24) {
		return ErrValidation("max_zoom must be between 0 and 24")
	}
	if r.MinZoom != nil && r.MaxZoom != nil && *r.MinZoom > *r.MaxZoom {
		return ErrValidation("min_zoom must be less than or equal to max_zoom")
	}
	return nil
}

// UpdateMapConfigRequest is the expected body for updating a map config.
type UpdateMapConfigRequest struct {
	Name       *string          `json:"name"`
	SourceType *string          `json:"source_type"`
	TileURL    *string          `json:"tile_url"`
	StyleJSON  *json.RawMessage `json:"style_json"`
	MinZoom    *int             `json:"min_zoom"`
	MaxZoom    *int             `json:"max_zoom"`
	IsDefault  *bool            `json:"is_default"`
}

// MapDefaultsResponse contains the server-level environment defaults for the
// map configuration. These are the baseline values the system falls back to
// when no database config is marked as default.
type MapDefaultsResponse struct {
	TileURL string `json:"tile_url"`
	MinZoom int    `json:"min_zoom"`
	MaxZoom int    `json:"max_zoom"`
}

// MapSettingsResponse is returned by the public map config endpoint.
// It combines the server-side defaults with the active map configuration.
type MapSettingsResponse struct {
	TileURL         string              `json:"tile_url"`
	StyleJSON       *json.RawMessage    `json:"style_json,omitempty"`
	CenterLat       float64             `json:"center_lat"`
	CenterLng       float64             `json:"center_lng"`
	Zoom            int                 `json:"zoom"`
	MinZoom         int                 `json:"min_zoom"`
	MaxZoom         int                 `json:"max_zoom"`
	TerrainURL      string              `json:"terrain_url"`
	TerrainEncoding string              `json:"terrain_encoding"`
	Configs         []MapConfigResponse `json:"configs"`
}

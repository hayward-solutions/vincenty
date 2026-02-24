package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/model"
	mockrepo "github.com/sitaware/api/internal/repository/mock"
)

func TestMapConfigService_BootstrapMapConfigs_SkipsWhenExist(t *testing.T) {
	repo := &mockrepo.MapConfigRepo{
		CountBuiltinFn: func(ctx context.Context) (int64, error) { return 2, nil },
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	err := svc.BootstrapMapConfigs(context.Background())
	if err != nil {
		t.Fatalf("BootstrapMapConfigs() error = %v", err)
	}
}

func TestMapConfigService_BootstrapMapConfigs_CreatesWhenEmpty(t *testing.T) {
	created := 0
	repo := &mockrepo.MapConfigRepo{
		CountBuiltinFn: func(ctx context.Context) (int64, error) { return 0, nil },
		CreateFn: func(ctx context.Context, mc *model.MapConfig) error {
			created++
			return nil
		},
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	err := svc.BootstrapMapConfigs(context.Background())
	if err != nil {
		t.Fatalf("BootstrapMapConfigs() error = %v", err)
	}
	if created != 2 { // OSM + Satellite
		t.Errorf("created = %d, want 2", created)
	}
}

func TestMapConfigService_Create(t *testing.T) {
	var created *model.MapConfig
	repo := &mockrepo.MapConfigRepo{
		CreateFn: func(ctx context.Context, mc *model.MapConfig) error {
			created = mc
			return nil
		},
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	callerID := uuid.New()
	mc, err := svc.Create(context.Background(), &model.CreateMapConfigRequest{
		Name:       "Custom Map",
		SourceType: "remote",
		TileURL:    "https://tiles.example.com/{z}/{x}/{y}.png",
	}, callerID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if mc.Name != "Custom Map" {
		t.Errorf("Name = %q, want %q", mc.Name, "Custom Map")
	}
	if created.IsBuiltin {
		t.Error("expected IsBuiltin=false")
	}
}

func TestMapConfigService_Create_AsDefault(t *testing.T) {
	defaultCleared := false
	repo := &mockrepo.MapConfigRepo{
		ClearDefaultFn: func(ctx context.Context) error {
			defaultCleared = true
			return nil
		},
		CreateFn: func(ctx context.Context, mc *model.MapConfig) error { return nil },
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	isDefault := true
	_, err := svc.Create(context.Background(), &model.CreateMapConfigRequest{
		Name:       "Default",
		SourceType: "remote",
		IsDefault:  &isDefault,
	}, uuid.New())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !defaultCleared {
		t.Error("expected ClearDefault to be called")
	}
}

func TestMapConfigService_Update_BuiltinReadOnly(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.MapConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.MapConfig, error) {
			return &model.MapConfig{ID: id, IsBuiltin: true}, nil
		},
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	newName := "Changed"
	_, err := svc.Update(context.Background(), id, &model.UpdateMapConfigRequest{Name: &newName})
	if err == nil {
		t.Fatal("expected error for modifying built-in config")
	}
}

func TestMapConfigService_Update_UserCreated(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.MapConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.MapConfig, error) {
			return &model.MapConfig{ID: id, Name: "Old", IsBuiltin: false}, nil
		},
		UpdateFn: func(ctx context.Context, mc *model.MapConfig) error { return nil },
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	newName := "New Name"
	mc, err := svc.Update(context.Background(), id, &model.UpdateMapConfigRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if mc.Name != "New Name" {
		t.Errorf("Name = %q, want %q", mc.Name, "New Name")
	}
}

func TestMapConfigService_Update_EmptyName(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.MapConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.MapConfig, error) {
			return &model.MapConfig{ID: id, IsBuiltin: false}, nil
		},
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	empty := ""
	_, err := svc.Update(context.Background(), id, &model.UpdateMapConfigRequest{Name: &empty})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestMapConfigService_Update_InvalidSourceType(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.MapConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.MapConfig, error) {
			return &model.MapConfig{ID: id, IsBuiltin: false}, nil
		},
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	invalid := "ftp"
	_, err := svc.Update(context.Background(), id, &model.UpdateMapConfigRequest{SourceType: &invalid})
	if err == nil {
		t.Fatal("expected error for invalid source type")
	}
}

func TestMapConfigService_Delete(t *testing.T) {
	id := uuid.New()
	deleted := false
	repo := &mockrepo.MapConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.MapConfig, error) {
			return &model.MapConfig{ID: id, IsBuiltin: false}, nil
		},
		DeleteFn: func(ctx context.Context, rid uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	err := svc.Delete(context.Background(), id)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called")
	}
}

func TestMapConfigService_Delete_BuiltinForbidden(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.MapConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.MapConfig, error) {
			return &model.MapConfig{ID: id, IsBuiltin: true}, nil
		},
	}
	svc := NewMapConfigService(repo, nil, nil, config.MapConfig{})

	err := svc.Delete(context.Background(), id)
	if err == nil {
		t.Fatal("expected error for deleting built-in config")
	}
}

func TestMapConfigService_GetSettings(t *testing.T) {
	tileURL := "https://tiles.example.com/{z}/{x}/{y}.png"
	repo := &mockrepo.MapConfigRepo{
		ListFn: func(ctx context.Context) ([]model.MapConfig, error) {
			return []model.MapConfig{
				{Name: "Default Map", TileURL: &tileURL, MinZoom: 1, MaxZoom: 17, IsDefault: true, IsEnabled: true},
			}, nil
		},
	}
	terrainRepo := &mockrepo.TerrainConfigRepo{
		GetDefaultFn: func(ctx context.Context) (*model.TerrainConfig, error) {
			return &model.TerrainConfig{TerrainURL: "https://terrain.example.com", TerrainEncoding: "mapbox"}, nil
		},
	}
	settingsRepo := &mockrepo.ServerSettingsRepo{
		GetFn: func(ctx context.Context, key string) (*model.ServerSetting, error) {
			if key == "mapbox_access_token" {
				return &model.ServerSetting{Value: "pk.test"}, nil
			}
			return nil, model.ErrNotFound("setting")
		},
	}
	svc := NewMapConfigService(repo, terrainRepo, settingsRepo, config.MapConfig{
		DefaultCenterLat: -33.86,
		DefaultCenterLng: 151.20,
		DefaultZoom:      10,
	})

	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if settings.TileURL != tileURL {
		t.Errorf("TileURL = %q, want %q", settings.TileURL, tileURL)
	}
	if settings.TerrainEncoding != "mapbox" {
		t.Errorf("TerrainEncoding = %q, want %q", settings.TerrainEncoding, "mapbox")
	}
	if settings.MapboxAccessToken != "pk.test" {
		t.Errorf("MapboxAccessToken = %q, want %q", settings.MapboxAccessToken, "pk.test")
	}
	if settings.CenterLat != -33.86 {
		t.Errorf("CenterLat = %v, want -33.86", settings.CenterLat)
	}
}

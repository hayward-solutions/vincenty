package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/model"
	mockrepo "github.com/sitaware/api/internal/repository/mock"
)

func TestTerrainConfigService_BootstrapTerrainConfigs_SkipsWhenExist(t *testing.T) {
	repo := &mockrepo.TerrainConfigRepo{
		CountBuiltinFn: func(ctx context.Context) (int64, error) {
			return 1, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	err := svc.BootstrapTerrainConfigs(context.Background())
	if err != nil {
		t.Fatalf("BootstrapTerrainConfigs() error = %v", err)
	}
}

func TestTerrainConfigService_BootstrapTerrainConfigs_CreatesWhenEmpty(t *testing.T) {
	created := 0
	repo := &mockrepo.TerrainConfigRepo{
		CountBuiltinFn: func(ctx context.Context) (int64, error) {
			return 0, nil
		},
		CreateFn: func(ctx context.Context, tc *model.TerrainConfig) error {
			created++
			if !tc.IsBuiltin {
				t.Error("expected IsBuiltin to be true")
			}
			return nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	err := svc.BootstrapTerrainConfigs(context.Background())
	if err != nil {
		t.Fatalf("BootstrapTerrainConfigs() error = %v", err)
	}
	if created == 0 {
		t.Error("expected at least one terrain config to be created")
	}
}

func TestTerrainConfigService_Create(t *testing.T) {
	var createdTC *model.TerrainConfig
	repo := &mockrepo.TerrainConfigRepo{
		CreateFn: func(ctx context.Context, tc *model.TerrainConfig) error {
			createdTC = tc
			return nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	req := &model.CreateTerrainConfigRequest{
		Name:            "Custom Terrain",
		SourceType:      "remote",
		TerrainURL:      "https://example.com/terrain/{z}/{x}/{y}.png",
		TerrainEncoding: "terrarium",
	}
	callerID := uuid.New()

	tc, err := svc.Create(context.Background(), req, callerID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if tc.Name != "Custom Terrain" {
		t.Errorf("Name = %q, want %q", tc.Name, "Custom Terrain")
	}
	if createdTC.IsBuiltin {
		t.Error("expected IsBuiltin to be false for user-created config")
	}
	if createdTC.CreatedBy == nil || *createdTC.CreatedBy != callerID {
		t.Errorf("CreatedBy = %v, want %v", createdTC.CreatedBy, callerID)
	}
}

func TestTerrainConfigService_Create_WithDefault(t *testing.T) {
	defaultCleared := false
	repo := &mockrepo.TerrainConfigRepo{
		ClearDefaultFn: func(ctx context.Context) error {
			defaultCleared = true
			return nil
		},
		CreateFn: func(ctx context.Context, tc *model.TerrainConfig) error {
			return nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	isDefault := true
	req := &model.CreateTerrainConfigRequest{
		Name:            "Default Terrain",
		SourceType:      "remote",
		TerrainURL:      "https://example.com/terrain",
		TerrainEncoding: "mapbox",
		IsDefault:       &isDefault,
	}

	tc, err := svc.Create(context.Background(), req, uuid.New())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !defaultCleared {
		t.Error("expected ClearDefault to be called")
	}
	if !tc.IsDefault {
		t.Error("expected IsDefault to be true")
	}
}

func TestTerrainConfigService_GetByID(t *testing.T) {
	id := uuid.New()
	expected := &model.TerrainConfig{ID: id, Name: "Test"}
	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			if rid != id {
				t.Errorf("GetByID called with %v, want %v", rid, id)
			}
			return expected, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	tc, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if tc.Name != "Test" {
		t.Errorf("Name = %q, want %q", tc.Name, "Test")
	}
}

func TestTerrainConfigService_List(t *testing.T) {
	expected := []model.TerrainConfig{{Name: "A"}, {Name: "B"}}
	repo := &mockrepo.TerrainConfigRepo{
		ListFn: func(ctx context.Context) ([]model.TerrainConfig, error) {
			return expected, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	configs, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("len(configs) = %d, want 2", len(configs))
	}
}

func TestTerrainConfigService_Update_UserCreated(t *testing.T) {
	id := uuid.New()
	existing := &model.TerrainConfig{ID: id, Name: "Old", SourceType: "remote", IsBuiltin: false}

	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			// Return a copy to avoid mutation issues
			cp := *existing
			return &cp, nil
		},
		UpdateFn: func(ctx context.Context, tc *model.TerrainConfig) error {
			return nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	newName := "New Name"
	tc, err := svc.Update(context.Background(), id, &model.UpdateTerrainConfigRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if tc.Name != "New Name" {
		t.Errorf("Name = %q, want %q", tc.Name, "New Name")
	}
}

func TestTerrainConfigService_Update_BuiltinReadOnly(t *testing.T) {
	id := uuid.New()
	existing := &model.TerrainConfig{ID: id, Name: "Builtin", IsBuiltin: true}

	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			cp := *existing
			return &cp, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	newName := "Changed"
	_, err := svc.Update(context.Background(), id, &model.UpdateTerrainConfigRequest{Name: &newName})
	if err == nil {
		t.Fatal("expected error for modifying built-in config name")
	}
}

func TestTerrainConfigService_Update_CannotDisableDefault(t *testing.T) {
	id := uuid.New()
	existing := &model.TerrainConfig{ID: id, IsBuiltin: false, IsDefault: true}

	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			cp := *existing
			return &cp, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	disabled := false
	_, err := svc.Update(context.Background(), id, &model.UpdateTerrainConfigRequest{IsEnabled: &disabled})
	if err == nil {
		t.Fatal("expected error when disabling default config")
	}
}

func TestTerrainConfigService_Update_EmptyName(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			return &model.TerrainConfig{ID: id, IsBuiltin: false}, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	empty := ""
	_, err := svc.Update(context.Background(), id, &model.UpdateTerrainConfigRequest{Name: &empty})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestTerrainConfigService_Update_InvalidSourceType(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			return &model.TerrainConfig{ID: id, IsBuiltin: false}, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	invalid := "ftp"
	_, err := svc.Update(context.Background(), id, &model.UpdateTerrainConfigRequest{SourceType: &invalid})
	if err == nil {
		t.Fatal("expected error for invalid source type")
	}
}

func TestTerrainConfigService_Update_InvalidEncoding(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			return &model.TerrainConfig{ID: id, IsBuiltin: false}, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	invalid := "invalid"
	_, err := svc.Update(context.Background(), id, &model.UpdateTerrainConfigRequest{TerrainEncoding: &invalid})
	if err == nil {
		t.Fatal("expected error for invalid encoding")
	}
}

func TestTerrainConfigService_Delete(t *testing.T) {
	id := uuid.New()
	deleted := false
	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			return &model.TerrainConfig{ID: id, IsBuiltin: false}, nil
		},
		DeleteFn: func(ctx context.Context, rid uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	err := svc.Delete(context.Background(), id)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called")
	}
}

func TestTerrainConfigService_Delete_BuiltinForbidden(t *testing.T) {
	id := uuid.New()
	repo := &mockrepo.TerrainConfigRepo{
		GetByIDFn: func(ctx context.Context, rid uuid.UUID) (*model.TerrainConfig, error) {
			return &model.TerrainConfig{ID: id, IsBuiltin: true}, nil
		},
	}
	svc := NewTerrainConfigService(repo, config.MapConfig{})

	err := svc.Delete(context.Background(), id)
	if err == nil {
		t.Fatal("expected error for deleting built-in config")
	}
}

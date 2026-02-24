package service

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	mockrepo "github.com/sitaware/api/internal/repository/mock"
	"github.com/sitaware/api/internal/storage"
)

func TestUserService_Create(t *testing.T) {
	var created *model.User
	userRepo := &mockrepo.UserRepo{
		ExistsByUsernameFn: func(ctx context.Context, username string) (bool, error) { return false, nil },
		ExistsByEmailFn:    func(ctx context.Context, email string) (bool, error) { return false, nil },
		CreateFn: func(ctx context.Context, user *model.User) error {
			created = user
			return nil
		},
	}
	svc := NewUserService(userRepo, nil, nil)

	user, err := svc.Create(context.Background(), &model.CreateUserRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("Username = %q, want %q", user.Username, "alice")
	}
	if created.PasswordHash == "" {
		t.Error("expected password hash to be set")
	}
	if !created.IsActive {
		t.Error("expected IsActive=true")
	}
}

func TestUserService_Create_DuplicateUsername(t *testing.T) {
	userRepo := &mockrepo.UserRepo{
		ExistsByUsernameFn: func(ctx context.Context, username string) (bool, error) { return true, nil },
	}
	svc := NewUserService(userRepo, nil, nil)

	_, err := svc.Create(context.Background(), &model.CreateUserRequest{
		Username: "existing",
		Email:    "new@example.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected conflict error for duplicate username")
	}
}

func TestUserService_Create_DuplicateEmail(t *testing.T) {
	userRepo := &mockrepo.UserRepo{
		ExistsByUsernameFn: func(ctx context.Context, username string) (bool, error) { return false, nil },
		ExistsByEmailFn:    func(ctx context.Context, email string) (bool, error) { return true, nil },
	}
	svc := NewUserService(userRepo, nil, nil)

	_, err := svc.Create(context.Background(), &model.CreateUserRequest{
		Username: "new",
		Email:    "existing@example.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected conflict error for duplicate email")
	}
}

func TestUserService_Update_ChangeEmail(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, Username: "bob", Email: "old@example.com"}, nil
		},
		ExistsByEmailFn: func(ctx context.Context, email string) (bool, error) { return false, nil },
		UpdateFn:        func(ctx context.Context, u *model.User) error { return nil },
	}
	svc := NewUserService(userRepo, nil, nil)

	newEmail := "new@example.com"
	user, err := svc.Update(context.Background(), userID, &model.UpdateUserRequest{Email: &newEmail})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "new@example.com")
	}
}

func TestUserService_Update_CannotRemoveLastAdmin(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, IsAdmin: true}, nil
		},
		CountAdminsFn: func(ctx context.Context) (int, error) { return 1, nil },
	}
	svc := NewUserService(userRepo, nil, nil)

	isAdmin := false
	_, err := svc.Update(context.Background(), userID, &model.UpdateUserRequest{IsAdmin: &isAdmin})
	if err == nil {
		t.Fatal("expected error when removing last admin")
	}
}

func TestUserService_Update_ShortPassword(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID}, nil
		},
	}
	svc := NewUserService(userRepo, nil, nil)

	short := "abc"
	_, err := svc.Update(context.Background(), userID, &model.UpdateUserRequest{Password: &short})
	if err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestUserService_Update_DeactivateInvalidatesTokens(t *testing.T) {
	userID := uuid.New()
	tokensDeleted := false
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, IsActive: true}, nil
		},
		UpdateFn: func(ctx context.Context, u *model.User) error { return nil },
	}
	tokenRepo := &mockrepo.TokenRepo{
		DeleteAllForUserFn: func(ctx context.Context, uid uuid.UUID) error {
			tokensDeleted = true
			return nil
		},
	}
	svc := NewUserService(userRepo, tokenRepo, nil)

	isActive := false
	_, err := svc.Update(context.Background(), userID, &model.UpdateUserRequest{IsActive: &isActive})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !tokensDeleted {
		t.Error("expected tokens to be deleted on deactivation")
	}
}

func TestUserService_Delete(t *testing.T) {
	userID := uuid.New()
	deleted := false
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, IsAdmin: false}, nil
		},
		DeleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	tokenRepo := &mockrepo.TokenRepo{
		DeleteAllForUserFn: func(ctx context.Context, uid uuid.UUID) error { return nil },
	}
	storageMock := &storage.MockStorage{
		DeleteFn: func(ctx context.Context, key string) error { return nil },
	}
	svc := NewUserService(userRepo, tokenRepo, storageMock)

	err := svc.Delete(context.Background(), userID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called")
	}
}

func TestUserService_Delete_LastAdmin(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, IsAdmin: true}, nil
		},
		CountAdminsFn: func(ctx context.Context) (int, error) { return 1, nil },
	}
	svc := NewUserService(userRepo, nil, nil)

	err := svc.Delete(context.Background(), userID)
	if err == nil {
		t.Fatal("expected error when deleting last admin")
	}
}

func TestUserService_UploadAvatar(t *testing.T) {
	userID := uuid.New()
	uploaded := false
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID}, nil
		},
		UpdateFn: func(ctx context.Context, u *model.User) error { return nil },
	}
	storageMock := &storage.MockStorage{
		UploadFn: func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
			uploaded = true
			return nil
		},
		DeleteFn: func(ctx context.Context, key string) error { return nil },
	}
	svc := NewUserService(userRepo, nil, storageMock)

	user, err := svc.UploadAvatar(context.Background(), userID, bytes.NewReader([]byte("img")), "photo.jpg", "image/jpeg", 3)
	if err != nil {
		t.Fatalf("UploadAvatar() error = %v", err)
	}
	if !uploaded {
		t.Error("expected Upload to be called")
	}
	if user.AvatarURL == nil {
		t.Error("expected AvatarURL to be set")
	}
}

func TestUserService_DeleteAvatar(t *testing.T) {
	userID := uuid.New()
	avatarURL := "avatars/old.jpg"
	s3Deleted := false
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, AvatarURL: &avatarURL}, nil
		},
		UpdateFn: func(ctx context.Context, u *model.User) error { return nil },
	}
	storageMock := &storage.MockStorage{
		DeleteFn: func(ctx context.Context, key string) error {
			s3Deleted = true
			return nil
		},
	}
	svc := NewUserService(userRepo, nil, storageMock)

	user, err := svc.DeleteAvatar(context.Background(), userID)
	if err != nil {
		t.Fatalf("DeleteAvatar() error = %v", err)
	}
	if !s3Deleted {
		t.Error("expected S3 delete to be called")
	}
	if user.AvatarURL != nil {
		t.Error("expected AvatarURL to be nil")
	}
}

func TestUserService_GetAvatarKey(t *testing.T) {
	url := "avatars/test.jpg"
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{AvatarURL: &url}, nil
		},
	}
	svc := NewUserService(userRepo, nil, nil)

	key, err := svc.GetAvatarKey(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetAvatarKey() error = %v", err)
	}
	if key != "avatars/test.jpg" {
		t.Errorf("key = %q, want %q", key, "avatars/test.jpg")
	}
}

func TestUserService_GetAvatarKey_NoAvatar(t *testing.T) {
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{}, nil
		},
	}
	svc := NewUserService(userRepo, nil, nil)

	key, err := svc.GetAvatarKey(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetAvatarKey() error = %v", err)
	}
	if key != "" {
		t.Errorf("key = %q, want empty", key)
	}
}

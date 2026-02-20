package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/auth"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// UserService handles user management business logic.
type UserService struct {
	userRepo  *repository.UserRepository
	tokenRepo *repository.TokenRepository
}

// NewUserService creates a new UserService.
func NewUserService(userRepo *repository.UserRepository, tokenRepo *repository.TokenRepository) *UserService {
	return &UserService{userRepo: userRepo, tokenRepo: tokenRepo}
}

// Create creates a new user.
func (s *UserService) Create(ctx context.Context, req *model.CreateUserRequest) (*model.User, error) {
	// Check uniqueness
	if exists, _ := s.userRepo.ExistsByUsername(ctx, req.Username); exists {
		return nil, model.ErrConflict("username already taken")
	}
	if exists, _ := s.userRepo.ExistsByEmail(ctx, req.Email); exists {
		return nil, model.ErrConflict("email already in use")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var displayName *string
	if req.DisplayName != "" {
		displayName = &req.DisplayName
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		DisplayName:  displayName,
		IsAdmin:      req.IsAdmin,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by ID.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// List retrieves a paginated list of users.
func (s *UserService) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	return s.userRepo.List(ctx, page, pageSize)
}

// Update modifies a user (admin operation).
func (s *UserService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateUserRequest) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Email != nil && *req.Email != user.Email {
		if exists, _ := s.userRepo.ExistsByEmail(ctx, *req.Email); exists {
			return nil, model.ErrConflict("email already in use")
		}
		user.Email = *req.Email
	}

	if req.DisplayName != nil {
		user.DisplayName = req.DisplayName
	}

	if req.Password != nil {
		if len(*req.Password) < 8 {
			return nil, model.ErrValidation("password must be at least 8 characters")
		}
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = hash
		// Invalidate all refresh tokens when password changes
		_ = s.tokenRepo.DeleteAllForUser(ctx, id)
	}

	if req.IsAdmin != nil {
		// Prevent removing admin from the last admin
		if user.IsAdmin && !*req.IsAdmin {
			count, err := s.userRepo.CountAdmins(ctx)
			if err != nil {
				return nil, err
			}
			if count <= 1 {
				return nil, model.ErrValidation("cannot remove admin role from the last admin")
			}
		}
		user.IsAdmin = *req.IsAdmin
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
		if !*req.IsActive {
			// Invalidate all refresh tokens when deactivating
			_ = s.tokenRepo.DeleteAllForUser(ctx, id)
		}
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateMe modifies the current user's own profile.
func (s *UserService) UpdateMe(ctx context.Context, id uuid.UUID, req *model.UpdateMeRequest) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Email != nil && *req.Email != user.Email {
		if exists, _ := s.userRepo.ExistsByEmail(ctx, *req.Email); exists {
			return nil, model.ErrConflict("email already in use")
		}
		user.Email = *req.Email
	}

	if req.DisplayName != nil {
		user.DisplayName = req.DisplayName
	}

	if req.Password != nil {
		if len(*req.Password) < 8 {
			return nil, model.ErrValidation("password must be at least 8 characters")
		}
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = hash
		_ = s.tokenRepo.DeleteAllForUser(ctx, id)
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Delete removes a user. Prevents deleting the last admin.
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if user.IsAdmin {
		count, err := s.userRepo.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return model.ErrValidation("cannot delete the last admin user")
		}
	}

	// Invalidate all refresh tokens
	_ = s.tokenRepo.DeleteAllForUser(ctx, id)

	return s.userRepo.Delete(ctx, id)
}

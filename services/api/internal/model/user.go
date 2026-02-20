package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account in the system.
type User struct {
	ID           uuid.UUID `json:"-"`
	Username     string    `json:"-"`
	Email        string    `json:"-"`
	PasswordHash string    `json:"-"`
	DisplayName  *string   `json:"-"`
	AvatarURL    *string   `json:"-"`
	IsAdmin      bool      `json:"-"`
	IsActive     bool      `json:"-"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
}

// UserResponse is the JSON representation of a user returned by the API.
type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	IsAdmin     bool      `json:"is_admin"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts a User to its API response representation.
func (u *User) ToResponse() UserResponse {
	displayName := ""
	if u.DisplayName != nil {
		displayName = *u.DisplayName
	}
	avatarURL := ""
	if u.AvatarURL != nil {
		avatarURL = *u.AvatarURL
	}
	return UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		IsAdmin:     u.IsAdmin,
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// CreateUserRequest is the expected body for creating a new user.
type CreateUserRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	IsAdmin     bool   `json:"is_admin"`
}

// Validate checks that required fields are present.
func (r *CreateUserRequest) Validate() error {
	if r.Username == "" {
		return ErrValidation("username is required")
	}
	if r.Email == "" {
		return ErrValidation("email is required")
	}
	if r.Password == "" {
		return ErrValidation("password is required")
	}
	if len(r.Password) < 8 {
		return ErrValidation("password must be at least 8 characters")
	}
	return nil
}

// UpdateUserRequest is the expected body for updating a user (admin).
type UpdateUserRequest struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Password    *string `json:"password"`
	IsAdmin     *bool   `json:"is_admin"`
	IsActive    *bool   `json:"is_active"`
}

// UpdateMeRequest is the expected body for updating own profile.
type UpdateMeRequest struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
}

// ChangePasswordRequest is the expected body for changing own password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Validate checks that required fields are present and valid.
func (r *ChangePasswordRequest) Validate() error {
	if r.CurrentPassword == "" {
		return ErrValidation("current password is required")
	}
	if r.NewPassword == "" {
		return ErrValidation("new password is required")
	}
	if len(r.NewPassword) < 8 {
		return ErrValidation("new password must be at least 8 characters")
	}
	return nil
}

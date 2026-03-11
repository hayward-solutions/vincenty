package model

// LoginRequest is the expected body for the login endpoint.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Validate checks that required fields are present.
func (r *LoginRequest) Validate() error {
	if r.Username == "" {
		return ErrValidation("username is required")
	}
	if r.Password == "" {
		return ErrValidation("password is required")
	}
	return nil
}

// RefreshRequest is the expected body for the token refresh endpoint.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Validate checks that required fields are present.
func (r *RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return ErrValidation("refresh_token is required")
	}
	return nil
}

// LogoutRequest is the expected body for the logout endpoint.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthResponse is returned on successful login or token refresh.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

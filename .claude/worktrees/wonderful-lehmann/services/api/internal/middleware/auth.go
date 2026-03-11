package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/auth"
	"github.com/vincenty/api/internal/model"
)

type contextKey string

const claimsKey contextKey = "claims"

// TokenValidator validates a raw API token string and returns claims.
// Implemented by service.APITokenService.
type TokenValidator interface {
	ValidateToken(ctx context.Context, raw string) (*auth.Claims, error)
}

// Auth provides authentication and authorization middleware.
type Auth struct {
	jwt      *auth.JWTService
	tokenVal TokenValidator // optional; nil if API tokens are not configured
}

// NewAuth creates a new Auth middleware.
func NewAuth(jwt *auth.JWTService, tokenVal TokenValidator) *Auth {
	return &Auth{jwt: jwt, tokenVal: tokenVal}
}

// resolveToken validates a bearer token string. If the token carries the API
// token prefix it is validated against the database; otherwise it is treated
// as a JWT.
func (a *Auth) resolveToken(ctx context.Context, tokenStr string) (*auth.Claims, error) {
	if a.tokenVal != nil && strings.HasPrefix(tokenStr, model.APITokenPrefix) {
		return a.tokenVal.ValidateToken(ctx, tokenStr)
	}
	return a.jwt.ValidateAccessToken(tokenStr)
}

// Authenticate verifies the JWT access token in the Authorization header
// and injects the claims into the request context.
func (a *Auth) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing authorization header")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header format")
			return
		}

		claims, err := a.resolveToken(r.Context(), parts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateWithQueryToken verifies the JWT access token from the
// Authorization header first, falling back to a "token" query parameter.
// This is needed for routes where the browser makes requests that cannot
// include an Authorization header (e.g. <img src> and <a href>).
func (a *Auth) AuthenticateWithQueryToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// Try Authorization header first
		if header := r.Header.Get("Authorization"); header != "" {
			parts := strings.SplitN(header, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
				tokenStr = parts[1]
			}
		}

		// Fall back to query parameter
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("token")
		}

		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing authorization")
			return
		}

		claims, err := a.resolveToken(r.Context(), tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin checks that the authenticated user is an admin.
// Must be used after Authenticate.
func (a *Auth) RequireAdmin(next http.Handler) http.Handler {
	return a.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || !claims.IsAdmin {
			writeError(w, http.StatusForbidden, "forbidden", "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// ClaimsFromContext retrieves JWT claims from the request context.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*auth.Claims)
	return claims, ok
}

// ContextWithClaims returns a new context with the given claims attached.
// This is exported for use in handler tests that need to inject claims
// without going through the full auth middleware.
func ContextWithClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// MFASetupChecker provides MFA enforcement state. Implemented by ServerSettingsRepository.
type MFASetupChecker interface {
	IsMFARequired(ctx context.Context) bool
}

// RequireMFASetup returns middleware that blocks access if the server-wide
// MFA requirement is enabled and the authenticated user hasn't configured MFA.
// Requests to MFA setup endpoints are always allowed through.
func (a *Auth) RequireMFASetup(checker MFASetupChecker, getUserMFAEnabled func(ctx context.Context, userID uuid.UUID) (bool, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow MFA setup endpoints through unconditionally
			path := r.URL.Path
			if strings.HasPrefix(path, "/api/v1/users/me/mfa/") ||
				path == "/api/v1/users/me" ||
				path == "/api/v1/server/settings" {
				next.ServeHTTP(w, r)
				return
			}

			if !checker.IsMFARequired(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			mfaEnabled, err := getUserMFAEnabled(r.Context(), claims.UserID)
			if err != nil || !mfaEnabled {
				writeError(w, http.StatusForbidden, "mfa_setup_required", "MFA must be configured before accessing this resource")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeError writes a JSON error response without importing the handler package.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sitaware/api/internal/auth"
)

type contextKey string

const claimsKey contextKey = "claims"

// Auth provides authentication and authorization middleware.
type Auth struct {
	jwt *auth.JWTService
}

// NewAuth creates a new Auth middleware.
func NewAuth(jwt *auth.JWTService) *Auth {
	return &Auth{jwt: jwt}
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

		claims, err := a.jwt.ValidateAccessToken(parts[1])
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

		claims, err := a.jwt.ValidateAccessToken(tokenStr)
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

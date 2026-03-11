package middleware

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/auth"
	"github.com/vincenty/api/internal/service"
)

// auditAction describes how to log a matched route.
type auditAction struct {
	Action       string
	ResourceType string
	// GroupIDFrom describes where to find the group_id:
	//   "path:{name}" — from path param
	//   "form:group_id" — from form/multipart value
	//   "" — not group-scoped
	GroupIDFrom string
}

// routeKey identifies a route by method + registered pattern.
type routeKey struct {
	Method  string
	Pattern string
}

// auditRoutes maps (method, pattern) to the audit action to record.
// Only mutations and login/logout are audited.
var auditRoutes = map[routeKey]auditAction{
	// Auth
	{"POST", "/api/v1/auth/login"}:  {"auth.login", "session", ""},
	{"POST", "/api/v1/auth/logout"}: {"auth.logout", "session", ""},

	// Users
	{"POST", "/api/v1/users"}:        {"user.create", "user", ""},
	{"PUT", "/api/v1/users/{id}"}:    {"user.update", "user", ""},
	{"DELETE", "/api/v1/users/{id}"}: {"user.delete", "user", ""},
	{"PUT", "/api/v1/users/me"}:      {"user.update_self", "user", ""},

	// Devices
	{"POST", "/api/v1/users/me/devices"}: {"device.create", "device", ""},
	{"PUT", "/api/v1/devices/{id}"}:      {"device.update", "device", ""},
	{"DELETE", "/api/v1/devices/{id}"}:   {"device.delete", "device", ""},

	// Groups
	{"POST", "/api/v1/groups"}:        {"group.create", "group", ""},
	{"PUT", "/api/v1/groups/{id}"}:    {"group.update", "group", "path:id"},
	{"DELETE", "/api/v1/groups/{id}"}: {"group.delete", "group", "path:id"},

	// Group members
	{"POST", "/api/v1/groups/{id}/members"}:            {"group.member_add", "group", "path:id"},
	{"PUT", "/api/v1/groups/{id}/members/{userId}"}:    {"group.member_update", "group", "path:id"},
	{"DELETE", "/api/v1/groups/{id}/members/{userId}"}: {"group.member_remove", "group", "path:id"},

	// Messages
	{"POST", "/api/v1/messages"}:        {"message.send", "message", "form:group_id"},
	{"DELETE", "/api/v1/messages/{id}"}: {"message.delete", "message", ""},

	// Map configs
	{"POST", "/api/v1/map-configs"}:        {"map_config.create", "map_config", ""},
	{"PUT", "/api/v1/map-configs/{id}"}:    {"map_config.update", "map_config", ""},
	{"DELETE", "/api/v1/map-configs/{id}"}: {"map_config.delete", "map_config", ""},

	// CoT
	{"POST", "/api/v1/cot/events"}: {"cot.ingest", "cot_event", ""},
}

// Audit returns middleware that records API actions to the audit_logs table.
// It captures response status and body (for POST requests) to extract resource IDs.
//
// jwtService is required because this middleware runs outside the per-route
// auth middleware. The auth middleware stores claims in a NEW *http.Request
// via r.WithContext(ctx), so by the time control returns here the original
// request's context does not contain claims. We therefore parse the JWT
// directly from the Authorization header.
func Audit(auditService *service.AuditService, jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the ResponseWriter to capture status and body
			rc := &responseCapture{
				ResponseWriter: w,
				status:         http.StatusOK,
				captureBody:    r.Method == http.MethodPost,
			}

			next.ServeHTTP(rc, r)

			// Only audit successful responses (2xx)
			if rc.status < 200 || rc.status >= 300 {
				return
			}

			// Match the route pattern
			pattern := r.Pattern
			if pattern == "" {
				return
			}
			// r.Pattern includes the method prefix (e.g. "POST /api/v1/users"),
			// strip it to get just the path pattern.
			parts := strings.SplitN(pattern, " ", 2)
			pathPattern := pattern
			if len(parts) == 2 {
				pathPattern = parts[1]
			}

			action, ok := auditRoutes[routeKey{Method: r.Method, Pattern: pathPattern}]
			if !ok {
				return
			}

			// Extract user ID from the Authorization header directly.
			// We cannot use ClaimsFromContext because the auth middleware
			// stores claims in a new *http.Request (via r.WithContext),
			// which is invisible to this outer middleware.
			var userID uuid.UUID
			if token := extractBearerToken(r); token != "" {
				if claims, err := jwtService.ValidateAccessToken(token); err == nil {
					userID = claims.UserID
				}
			}

			// For login, there is no auth header yet — extract user_id
			// from the response body instead.
			if action.Action == "auth.login" && rc.captureBody && rc.body.Len() > 0 {
				userID = extractUserIDFromBody(rc.body.Bytes())
			}

			// Skip if we have no user at all
			if userID == uuid.Nil {
				return
			}

			// Extract resource ID
			var resourceID *uuid.UUID
			if r.Method == http.MethodPut || r.Method == http.MethodDelete {
				// From path param {id}
				if v := r.PathValue("id"); v != "" {
					if id, err := uuid.Parse(v); err == nil {
						resourceID = &id
					}
				}
			} else if r.Method == http.MethodPost && rc.captureBody && rc.body.Len() > 0 {
				// From response body
				if id := extractIDFromBody(rc.body.Bytes()); id != uuid.Nil {
					rid := id
					resourceID = &rid
				}
			}

			// Extract group ID
			var groupID *uuid.UUID
			groupID = extractGroupID(r, action.GroupIDFrom)

			// Extract IP address
			ip := ExtractIP(r)

			// Fire-and-forget: write audit log asynchronously
			go func() {
				err := auditService.LogAction(context.Background(), service.CreateAuditParams{
					UserID:       userID,
					Action:       action.Action,
					ResourceType: action.ResourceType,
					ResourceID:   resourceID,
					GroupID:      groupID,
					IPAddress:    ip,
				})
				if err != nil {
					slog.Error("failed to write audit log", "error", err, "action", action.Action)
				}
			}()
		})
	}
}

// responseCapture wraps http.ResponseWriter to capture status code and optionally the body.
type responseCapture struct {
	http.ResponseWriter
	status      int
	captureBody bool
	body        bytes.Buffer
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if rc.captureBody {
		rc.body.Write(b) // tee into buffer
	}
	return rc.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker for WebSocket compatibility.
func (rc *responseCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rc.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// extractIDFromBody parses the "id" field from a JSON response body.
func extractIDFromBody(body []byte) uuid.UUID {
	var result struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return uuid.Nil
	}
	return result.ID
}

// extractUserIDFromBody parses "user.id" or "user_id" from a login response.
func extractUserIDFromBody(body []byte) uuid.UUID {
	// Try nested {"user":{"id":"..."}}
	var nested struct {
		User struct {
			ID uuid.UUID `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &nested); err == nil && nested.User.ID != uuid.Nil {
		return nested.User.ID
	}
	// Try flat {"user_id":"..."}
	var flat struct {
		UserID uuid.UUID `json:"user_id"`
	}
	if err := json.Unmarshal(body, &flat); err == nil {
		return flat.UserID
	}
	return uuid.Nil
}

// extractBearerToken pulls the raw JWT string from the Authorization header.
func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

// extractGroupID extracts the group_id from the request based on the source descriptor.
func extractGroupID(r *http.Request, source string) *uuid.UUID {
	if source == "" {
		return nil
	}
	parts := strings.SplitN(source, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	var raw string
	switch parts[0] {
	case "path":
		raw = r.PathValue(parts[1])
	case "form":
		raw = r.FormValue(parts[1])
	}
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

// ExtractIP returns the client IP address, preferring X-Forwarded-For.
func ExtractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First IP in the comma-separated list
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Strip port from RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

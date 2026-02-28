package ws

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/auth"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
	"nhooyr.io/websocket"
)

// TokenValidator validates a raw API token string and returns auth claims.
type TokenValidator interface {
	ValidateToken(ctx context.Context, raw string) (*auth.Claims, error)
}

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub        *Hub
	jwt        *auth.JWTService
	tokenVal   TokenValidator // optional; nil if API tokens are not configured
	deviceRepo repository.DeviceRepo
	groupRepo  repository.GroupRepo
}

// NewHandler creates a new WebSocket Handler.
func NewHandler(hub *Hub, jwt *auth.JWTService, tokenVal TokenValidator, deviceRepo repository.DeviceRepo, groupRepo repository.GroupRepo) *Handler {
	return &Handler{
		hub:        hub,
		jwt:        jwt,
		tokenVal:   tokenVal,
		deviceRepo: deviceRepo,
		groupRepo:  groupRepo,
	}
}

// resolveToken validates a token string. API tokens (sat_ prefix) are checked
// against the database; everything else is treated as a JWT.
func (h *Handler) resolveToken(ctx context.Context, tokenStr string) (*auth.Claims, error) {
	if h.tokenVal != nil && strings.HasPrefix(tokenStr, model.APITokenPrefix) {
		return h.tokenVal.ValidateToken(ctx, tokenStr)
	}
	return h.jwt.ValidateAccessToken(tokenStr)
}

// ServeHTTP handles GET /api/v1/ws?token=<jwt|sat_token>&device_id=<uuid>
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// --- Validate token from query parameter ---
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token parameter", http.StatusUnauthorized)
		return
	}

	claims, err := h.resolveToken(ctx, tokenStr)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	// --- Validate device_id ---
	deviceIDStr := r.URL.Query().Get("device_id")
	if deviceIDStr == "" {
		http.Error(w, "missing device_id parameter", http.StatusBadRequest)
		return
	}

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		http.Error(w, "invalid device_id", http.StatusBadRequest)
		return
	}

	// Verify device belongs to user
	device, err := h.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		http.Error(w, "device not found", http.StatusBadRequest)
		return
	}
	if device.UserID != claims.UserID {
		http.Error(w, "device does not belong to user", http.StatusForbidden)
		return
	}

	// Record the connecting app version (optional query param). This keeps the
	// version up to date on every connect without requiring a separate API call.
	appVersionStr := r.URL.Query().Get("app_version")
	var appVersionPtr *string
	if appVersionStr != "" {
		appVersionPtr = &appVersionStr
	}
	ua := r.UserAgent()
	var uaPtr *string
	if ua != "" {
		uaPtr = &ua
	}
	_ = h.deviceRepo.TouchLastSeen(ctx, deviceID, uaPtr, appVersionPtr)

	// --- Load user's group memberships ---
	groups, _, err := h.groupRepo.ListByUserID(ctx, claims.UserID)
	if err != nil {
		slog.Error("failed to load user groups for ws", "error", err, "user_id", claims.UserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	groupIDs := make([]uuid.UUID, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}

	// --- Get username for broadcasts ---
	user, err := h.hub.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		slog.Error("failed to load user for ws", "error", err, "user_id", claims.UserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// --- Upgrade to WebSocket ---
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Allow connections from any origin in dev; tighten for production.
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("ws upgrade failed", "error", err)
		return
	}

	// --- Create and register client ---
	client := NewClient(h.hub, conn, claims.UserID, deviceID, device.Name, device.IsPrimary, user.Username, claims.IsAdmin, groupIDs)

	h.hub.register <- client

	// Use a detached context for the WebSocket lifecycle so it is not
	// tied to the HTTP request's context (which may carry server deadlines).
	wsCtx := context.Background()

	// Send initial messages (connected ack + location snapshots)
	h.hub.SendConnected(wsCtx, client)
	h.hub.SendSnapshot(wsCtx, client)

	// Run the client (blocks until disconnect)
	client.Run(wsCtx)
}

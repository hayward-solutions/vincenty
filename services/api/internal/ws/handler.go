package ws

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/auth"
	"github.com/sitaware/api/internal/repository"
	"nhooyr.io/websocket"
)

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub        *Hub
	jwt        *auth.JWTService
	deviceRepo *repository.DeviceRepository
	groupRepo  *repository.GroupRepository
}

// NewHandler creates a new WebSocket Handler.
func NewHandler(hub *Hub, jwt *auth.JWTService, deviceRepo *repository.DeviceRepository, groupRepo *repository.GroupRepository) *Handler {
	return &Handler{
		hub:        hub,
		jwt:        jwt,
		deviceRepo: deviceRepo,
		groupRepo:  groupRepo,
	}
}

// ServeHTTP handles GET /api/v1/ws?token=<jwt>&device_id=<uuid>
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// --- Validate JWT from query parameter ---
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token parameter", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwt.ValidateAccessToken(tokenStr)
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
	client := NewClient(h.hub, conn, claims.UserID, deviceID, user.Username, claims.IsAdmin, groupIDs)

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

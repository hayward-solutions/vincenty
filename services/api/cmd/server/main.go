package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sitaware/api/internal/auth"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/database"
	"github.com/sitaware/api/internal/handler"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
	"github.com/sitaware/api/internal/service"
	"github.com/sitaware/api/internal/storage"
	"github.com/sitaware/api/internal/ws"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Configure structured logging
	logLevel := slog.LevelInfo
	switch cfg.Server.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	slog.Info("starting SitAware API",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
	)

	// -----------------------------------------------------------------------
	// Database connection with retry
	// -----------------------------------------------------------------------
	db, err := connectDB(context.Background(), cfg.DB)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run database migrations
	if err := database.RunMigrations(cfg.DB.DSN()); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// Redis connection
	// -----------------------------------------------------------------------
	rdb, err := connectRedis(context.Background(), cfg.Redis)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// -----------------------------------------------------------------------
	// S3 / Minio client
	// -----------------------------------------------------------------------
	s3Client, err := connectS3(context.Background(), cfg.S3)
	if err != nil {
		slog.Error("failed to connect to object storage", "error", err)
		os.Exit(1)
	}
	// -----------------------------------------------------------------------
	// Storage Service (S3/Minio)
	// -----------------------------------------------------------------------
	storageSvc := storage.NewStorageService(s3Client, cfg.S3.Bucket)

	// -----------------------------------------------------------------------
	// Repositories
	// -----------------------------------------------------------------------
	userRepo := repository.NewUserRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	locationRepo := repository.NewLocationRepository(db)
	mapConfigRepo := repository.NewMapConfigRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	cotRepo := repository.NewCotRepository(db)

	// -----------------------------------------------------------------------
	// Pub/Sub
	// -----------------------------------------------------------------------
	ps := pubsub.NewRedisPubSub(rdb)
	defer ps.Close()

	// -----------------------------------------------------------------------
	// Services
	// -----------------------------------------------------------------------
	jwtService := auth.NewJWTService(cfg.JWT)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtService)
	userService := service.NewUserService(userRepo, tokenRepo)
	groupService := service.NewGroupService(groupRepo, userRepo)
	locationService := service.NewLocationService(locationRepo, groupRepo, ps, cfg.WS.LocationThrottle)
	mapConfigService := service.NewMapConfigService(mapConfigRepo, cfg.Map)
	messageService := service.NewMessageService(messageRepo, groupRepo, storageSvc, ps)
	auditService := service.NewAuditService(auditRepo, groupRepo)
	cotService := service.NewCotService(cotRepo, deviceRepo, userRepo, groupRepo, locationService)

	// -----------------------------------------------------------------------
	// Bootstrap admin user
	// -----------------------------------------------------------------------
	if err := authService.BootstrapAdmin(context.Background(), cfg.Admin); err != nil {
		slog.Error("failed to bootstrap admin user", "error", err)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// Handlers
	// -----------------------------------------------------------------------
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	deviceHandler := handler.NewDeviceHandler(deviceRepo)
	groupHandler := handler.NewGroupHandler(groupService)
	mapConfigHandler := handler.NewMapConfigHandler(mapConfigService)
	locationHandler := handler.NewLocationHandler(locationService)
	messageHandler := handler.NewMessageHandler(messageService)
	auditHandler := handler.NewAuditHandler(auditService)
	cotHandler := handler.NewCotHandler(cotService)

	// -----------------------------------------------------------------------
	// WebSocket Hub
	// -----------------------------------------------------------------------
	hub := ws.NewHub(ps, locationService, groupRepo, userRepo)
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()
	go hub.Run(hubCtx)

	wsHandler := ws.NewHandler(hub, jwtService, deviceRepo, groupRepo)

	// -----------------------------------------------------------------------
	// Token cleanup (purge expired refresh tokens on a schedule)
	// -----------------------------------------------------------------------
	go func() {
		ticker := time.NewTicker(cfg.TokenCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count, err := tokenRepo.DeleteExpired(context.Background())
				if err != nil {
					slog.Error("token cleanup failed", "error", err)
				} else if count > 0 {
					slog.Info("expired tokens cleaned up", "count", count)
				}
			case <-hubCtx.Done():
				return
			}
		}
	}()

	// -----------------------------------------------------------------------
	// Middleware
	// -----------------------------------------------------------------------
	authMW := middleware.NewAuth(jwtService)

	// -----------------------------------------------------------------------
	// HTTP Router
	// -----------------------------------------------------------------------
	mux := http.NewServeMux()

	// Health checks (public)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unavailable","error":"%s"}`, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ready"}`)
	})

	// API info (public)
	mux.HandleFunc("GET /api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"service":"sitaware-api","version":"0.1.0"}`)
	})

	// WebSocket (auth via query param in handler)
	mux.Handle("GET /api/v1/ws", wsHandler)

	// Auth (public)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)

	// Auth (authenticated)
	mux.Handle("POST /api/v1/auth/logout", authMW.Authenticate(http.HandlerFunc(authHandler.Logout)))

	// Users - self (authenticated)
	mux.Handle("GET /api/v1/users/me", authMW.Authenticate(http.HandlerFunc(userHandler.GetMe)))
	mux.Handle("PUT /api/v1/users/me", authMW.Authenticate(http.HandlerFunc(userHandler.UpdateMe)))

	// Devices - self (authenticated)
	mux.Handle("GET /api/v1/users/me/devices", authMW.Authenticate(http.HandlerFunc(deviceHandler.List)))
	mux.Handle("POST /api/v1/users/me/devices", authMW.Authenticate(http.HandlerFunc(deviceHandler.Create)))
	mux.Handle("PUT /api/v1/devices/{id}", authMW.Authenticate(http.HandlerFunc(deviceHandler.Update)))
	mux.Handle("DELETE /api/v1/devices/{id}", authMW.Authenticate(http.HandlerFunc(deviceHandler.Delete)))

	// Users - admin
	mux.Handle("GET /api/v1/users", authMW.RequireAdmin(http.HandlerFunc(userHandler.List)))
	mux.Handle("POST /api/v1/users", authMW.RequireAdmin(http.HandlerFunc(userHandler.Create)))
	mux.Handle("GET /api/v1/users/{id}", authMW.RequireAdmin(http.HandlerFunc(userHandler.Get)))
	mux.Handle("PUT /api/v1/users/{id}", authMW.RequireAdmin(http.HandlerFunc(userHandler.Update)))
	mux.Handle("DELETE /api/v1/users/{id}", authMW.RequireAdmin(http.HandlerFunc(userHandler.Delete)))

	// Groups - my groups (authenticated)
	mux.Handle("GET /api/v1/users/me/groups", authMW.Authenticate(http.HandlerFunc(groupHandler.ListMyGroups)))

	// Groups - members (authenticated, permission checked in service)
	mux.Handle("GET /api/v1/groups/{id}/members", authMW.Authenticate(http.HandlerFunc(groupHandler.ListMembers)))
	mux.Handle("POST /api/v1/groups/{id}/members", authMW.Authenticate(http.HandlerFunc(groupHandler.AddMember)))
	mux.Handle("PUT /api/v1/groups/{id}/members/{userId}", authMW.Authenticate(http.HandlerFunc(groupHandler.UpdateMember)))
	mux.Handle("DELETE /api/v1/groups/{id}/members/{userId}", authMW.Authenticate(http.HandlerFunc(groupHandler.RemoveMember)))

	// Groups - CRUD (admin)
	mux.Handle("GET /api/v1/groups", authMW.RequireAdmin(http.HandlerFunc(groupHandler.List)))
	mux.Handle("POST /api/v1/groups", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Create)))
	mux.Handle("GET /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Get)))
	mux.Handle("PUT /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Update)))
	mux.Handle("DELETE /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Delete)))

	// Locations - group history (authenticated, permission checked in service)
	mux.Handle("GET /api/v1/groups/{id}/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetGroupHistory)))

	// Locations - all latest (admin)
	mux.Handle("GET /api/v1/locations", authMW.RequireAdmin(http.HandlerFunc(locationHandler.GetAllLocations)))

	// Map settings (authenticated)
	mux.Handle("GET /api/v1/map/settings", authMW.Authenticate(http.HandlerFunc(mapConfigHandler.GetSettings)))

	// Map configs - CRUD (admin)
	mux.Handle("GET /api/v1/map-configs", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.List)))
	mux.Handle("POST /api/v1/map-configs", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.Create)))
	mux.Handle("GET /api/v1/map-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.Get)))
	mux.Handle("PUT /api/v1/map-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.Update)))
	mux.Handle("DELETE /api/v1/map-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.Delete)))

	// Messages (authenticated, permission checked in service)
	mux.Handle("POST /api/v1/messages", authMW.Authenticate(http.HandlerFunc(messageHandler.Send)))
	mux.Handle("GET /api/v1/groups/{id}/messages", authMW.Authenticate(http.HandlerFunc(messageHandler.ListGroupMessages)))
	mux.Handle("GET /api/v1/messages/conversations", authMW.Authenticate(http.HandlerFunc(messageHandler.ListDMConversations)))
	mux.Handle("GET /api/v1/messages/direct/{userId}", authMW.Authenticate(http.HandlerFunc(messageHandler.ListDirectMessages)))
	mux.Handle("GET /api/v1/messages/{id}", authMW.Authenticate(http.HandlerFunc(messageHandler.GetMessage)))
	mux.Handle("DELETE /api/v1/messages/{id}", authMW.Authenticate(http.HandlerFunc(messageHandler.DeleteMessage)))
	mux.Handle("GET /api/v1/attachments/{id}/download", authMW.AuthenticateWithQueryToken(http.HandlerFunc(messageHandler.DownloadAttachment)))

	// Audit logs - self (authenticated)
	mux.Handle("GET /api/v1/audit-logs/me", authMW.Authenticate(http.HandlerFunc(auditHandler.GetMyLogs)))
	mux.Handle("GET /api/v1/audit-logs/me/export", authMW.Authenticate(http.HandlerFunc(auditHandler.ExportMyLogs)))

	// Audit logs - group (authenticated, permission checked in service)
	mux.Handle("GET /api/v1/groups/{id}/audit-logs", authMW.Authenticate(http.HandlerFunc(auditHandler.GetGroupLogs)))

	// Audit logs - admin
	mux.Handle("GET /api/v1/audit-logs", authMW.RequireAdmin(http.HandlerFunc(auditHandler.GetAllLogs)))
	mux.Handle("GET /api/v1/audit-logs/export", authMW.RequireAdmin(http.HandlerFunc(auditHandler.ExportAllLogs)))

	// Location history - self (authenticated)
	mux.Handle("GET /api/v1/users/me/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetMyHistory)))
	mux.Handle("GET /api/v1/users/me/locations/export", authMW.Authenticate(http.HandlerFunc(locationHandler.ExportGPX)))

	// CoT (Cursor on Target) - authenticated
	mux.Handle("POST /api/v1/cot/events", authMW.Authenticate(http.HandlerFunc(cotHandler.IngestEvents)))
	mux.Handle("GET /api/v1/cot/events", authMW.Authenticate(http.HandlerFunc(cotHandler.ListEvents)))
	mux.Handle("GET /api/v1/cot/events/{uid}", authMW.Authenticate(http.HandlerFunc(cotHandler.GetLatestByUID)))

	// -----------------------------------------------------------------------
	// Apply global middleware and start server
	// -----------------------------------------------------------------------
	auditMW := middleware.Audit(auditService, jwtService)

	var rootHandler http.Handler = mux
	rootHandler = auditMW(rootHandler)
	rootHandler = middleware.Logging(rootHandler)
	rootHandler = middleware.RateLimit(cfg.RateLimit)(rootHandler)
	rootHandler = middleware.MaxBodySize(cfg.Security.MaxRequestBodyBytes)(rootHandler)
	rootHandler = middleware.CORS(cfg.CORS)(rootHandler)

	srv := &http.Server{
		Addr:        cfg.Addr(),
		Handler:     rootHandler,
		IdleTimeout: 120 * time.Second,
		// ReadTimeout and WriteTimeout are intentionally unset (0 = no timeout).
		// nhooyr.io/websocket v1 does not hijack connections, so net/http server
		// timeouts would kill long-lived WebSocket connections. The WS client
		// manages its own liveness via ping/pong.
	}

	// Channel to listen for errors from the server
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", srv.Addr)
		serverErr <- srv.ListenAndServe()
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("server error", "error", err)
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig.String())
	}

	// Stop the WebSocket hub
	hubCancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}

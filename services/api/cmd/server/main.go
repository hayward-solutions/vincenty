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

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/vincenty/api/internal/auth"
	"github.com/vincenty/api/internal/config"
	"github.com/vincenty/api/internal/database"
	"github.com/vincenty/api/internal/handler"
	"github.com/vincenty/api/internal/middleware"
	"github.com/vincenty/api/internal/pubsub"
	"github.com/vincenty/api/internal/repository"
	"github.com/vincenty/api/internal/service"
	"github.com/vincenty/api/internal/storage"
	"github.com/vincenty/api/internal/ws"
)

// version is set at build time via -ldflags "-X main.version=<value>".
// Falls back to "dev" for local builds.
var version = "dev"

func main() {
	// Subcommand: healthcheck (used as container health probe in distroless images)
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runHealthcheck()
	}

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

	slog.Info("starting Vincenty API",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"version", version,
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
	terrainConfigRepo := repository.NewTerrainConfigRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	drawingRepo := repository.NewDrawingRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	cotRepo := repository.NewCotRepository(db)
	mfaRepo := repository.NewMFARepository(db)
	serverSettingsRepo := repository.NewServerSettingsRepository(db)
	apiTokenRepo := repository.NewAPITokenRepository(db)
	garminInReachRepo := repository.NewGarminInReachRepository(db)

	// -----------------------------------------------------------------------
	// Pub/Sub
	// -----------------------------------------------------------------------
	ps := pubsub.NewRedisPubSub(rdb)
	defer ps.Close()

	// -----------------------------------------------------------------------
	// Services
	// -----------------------------------------------------------------------
	jwtService := auth.NewJWTService(cfg.JWT)

	// -----------------------------------------------------------------------
	// MFA encryption + WebAuthn
	// -----------------------------------------------------------------------
	var encryptor auth.SecretEncryptor
	if cfg.MFA.KMSKeyARN != "" {
		// Use AWS KMS for TOTP secret encryption
		kmsClient, err := connectKMS(context.Background(), cfg.S3.Region)
		if err != nil {
			slog.Error("failed to create KMS client", "error", err)
			os.Exit(1)
		}
		encryptor = auth.NewKMSEncryptor(kmsClient, cfg.MFA.KMSKeyARN)
		slog.Info("using AWS KMS for MFA secret encryption", "key_arn", cfg.MFA.KMSKeyARN)
	} else {
		// Derive encryption key from JWT secret via HKDF
		var err error
		encryptor, err = auth.NewLocalEncryptor([]byte(cfg.JWT.Secret))
		if err != nil {
			slog.Error("failed to create local encryptor", "error", err)
			os.Exit(1)
		}
		slog.Info("using HKDF-derived local encryption for MFA secrets")
	}

	waConfig := &webauthn.Config{
		RPDisplayName: cfg.WebAuthn.RPDisplayName,
		RPID:          cfg.WebAuthn.RPID,
		RPOrigins:     cfg.WebAuthn.RPOrigins,
	}
	wa, err := webauthn.New(waConfig)
	if err != nil {
		slog.Error("failed to create WebAuthn instance", "error", err)
		os.Exit(1)
	}

	permissionPolicyService := service.NewPermissionPolicyService(serverSettingsRepo)
	mfaService := service.NewMFAService(mfaRepo, userRepo, encryptor, rdb, wa)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtService, mfaService)
	userService := service.NewUserService(userRepo, tokenRepo, storageSvc)
	groupService := service.NewGroupService(groupRepo, userRepo, permissionPolicyService, ps)
	locationService := service.NewLocationService(locationRepo, groupRepo, ps, cfg.WS.LocationThrottle)
	mapConfigService := service.NewMapConfigService(mapConfigRepo, terrainConfigRepo, serverSettingsRepo, cfg.Map)
	terrainConfigService := service.NewTerrainConfigService(terrainConfigRepo, cfg.Map)
	auditService := service.NewAuditService(auditRepo, groupRepo, permissionPolicyService)
	cotService := service.NewCotService(cotRepo, deviceRepo, userRepo, groupRepo, locationService)
	apiTokenService := service.NewAPITokenService(apiTokenRepo)
	garminInReachService := service.NewGarminInReachService(garminInReachRepo, deviceRepo, userRepo, groupRepo, locationService)

	// -----------------------------------------------------------------------
	// Bootstrap admin user
	// -----------------------------------------------------------------------
	if err := authService.BootstrapAdmin(context.Background(), cfg.Admin); err != nil {
		slog.Error("failed to bootstrap admin user", "error", err)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// Bootstrap built-in map and terrain configs
	// -----------------------------------------------------------------------
	if err := mapConfigService.BootstrapMapConfigs(context.Background()); err != nil {
		slog.Error("failed to bootstrap map configs", "error", err)
		os.Exit(1)
	}
	if err := terrainConfigService.BootstrapTerrainConfigs(context.Background()); err != nil {
		slog.Error("failed to bootstrap terrain configs", "error", err)
		os.Exit(1)
	}

	messageService := service.NewMessageService(messageRepo, groupRepo, storageSvc, ps, permissionPolicyService)
	drawingService := service.NewDrawingService(drawingRepo, messageRepo, groupRepo, ps, permissionPolicyService)

	// -----------------------------------------------------------------------
	// WebSocket Hub
	// -----------------------------------------------------------------------
	hub := ws.NewHub(ps, locationService, permissionPolicyService, groupRepo, userRepo)
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()
	go hub.Run(hubCtx)

	wsHandler := ws.NewHandler(hub, jwtService, apiTokenService, deviceRepo, groupRepo)

	// -----------------------------------------------------------------------
	// Handlers
	// -----------------------------------------------------------------------
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService, storageSvc)
	deviceHandler := handler.NewDeviceHandler(deviceRepo)
	groupHandler := handler.NewGroupHandler(groupService)
	mapConfigHandler := handler.NewMapConfigHandler(mapConfigService)
	terrainConfigHandler := handler.NewTerrainConfigHandler(terrainConfigService)
	locationHandler := handler.NewLocationHandler(locationService)
	messageHandler := handler.NewMessageHandler(messageService)
	drawingHandler := handler.NewDrawingHandler(drawingService)
	auditHandler := handler.NewAuditHandler(auditService)
	cotHandler := handler.NewCotHandler(cotService)
	mfaHandler := handler.NewMFAHandler(mfaService, authService)
	serverSettingsHandler := handler.NewServerSettingsHandler(serverSettingsRepo)
	permissionPolicyHandler := handler.NewPermissionPolicyHandler(permissionPolicyService)
	apiTokenHandler := handler.NewAPITokenHandler(apiTokenService)
	garminInReachHandler := handler.NewGarminInReachHandler(garminInReachService)

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
					slog.Info("expired refresh tokens cleaned up", "count", count)
				}
				apiCount, apiErr := apiTokenRepo.DeleteExpired(context.Background())
				if apiErr != nil {
					slog.Error("api token cleanup failed", "error", apiErr)
				} else if apiCount > 0 {
					slog.Info("expired api tokens cleaned up", "count", apiCount)
				}
			case <-hubCtx.Done():
				return
			}
		}
	}()

	// -----------------------------------------------------------------------
	// Garmin InReach background poller
	// -----------------------------------------------------------------------
	if cfg.Garmin.Enabled {
		go garminInReachService.RunPoller(hubCtx, cfg.Garmin.PollTick)
	}

	// -----------------------------------------------------------------------
	// Middleware
	// -----------------------------------------------------------------------
	authMW := middleware.NewAuth(jwtService, apiTokenService)

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
	mux.HandleFunc("GET /api/v1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":"vincenty-api","version":%q}`, version)
	})

	// WebSocket (auth via query param in handler)
	mux.Handle("GET /api/v1/ws", wsHandler)

	// Auth (public)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)

	// Auth (authenticated)
	mux.Handle("POST /api/v1/auth/logout", authMW.Authenticate(http.HandlerFunc(authHandler.Logout)))

	// MFA login challenge (public — requires mfa_token in body)
	mux.HandleFunc("POST /api/v1/auth/mfa/totp", mfaHandler.MFAVerifyTOTP)
	mux.HandleFunc("POST /api/v1/auth/mfa/recovery", mfaHandler.MFARecovery)
	mux.HandleFunc("POST /api/v1/auth/mfa/webauthn/begin", mfaHandler.MFAWebAuthnBegin)
	mux.HandleFunc("POST /api/v1/auth/mfa/webauthn/finish", mfaHandler.MFAWebAuthnFinish)

	// Passkey passwordless login (public)
	mux.HandleFunc("POST /api/v1/auth/passkey/begin", mfaHandler.PasskeyBegin)
	mux.HandleFunc("POST /api/v1/auth/passkey/finish", mfaHandler.PasskeyFinish)

	// Users - self (authenticated)
	mux.Handle("GET /api/v1/users/me", authMW.Authenticate(http.HandlerFunc(userHandler.GetMe)))
	mux.Handle("PUT /api/v1/users/me", authMW.Authenticate(http.HandlerFunc(userHandler.UpdateMe)))
	mux.Handle("PUT /api/v1/users/me/password", authMW.Authenticate(http.HandlerFunc(userHandler.ChangePassword)))
	mux.Handle("PUT /api/v1/users/me/avatar", authMW.Authenticate(http.HandlerFunc(userHandler.UploadAvatar)))
	mux.Handle("DELETE /api/v1/users/me/avatar", authMW.Authenticate(http.HandlerFunc(userHandler.DeleteAvatar)))
	mux.Handle("GET /api/v1/users/{id}/avatar", authMW.AuthenticateWithQueryToken(http.HandlerFunc(userHandler.ServeAvatar)))

	// MFA setup (authenticated)
	mux.Handle("POST /api/v1/users/me/mfa/totp/setup", authMW.Authenticate(http.HandlerFunc(mfaHandler.SetupTOTP)))
	mux.Handle("POST /api/v1/users/me/mfa/totp/verify", authMW.Authenticate(http.HandlerFunc(mfaHandler.VerifyTOTPSetup)))
	mux.Handle("POST /api/v1/users/me/mfa/webauthn/register/begin", authMW.Authenticate(http.HandlerFunc(mfaHandler.BeginWebAuthnRegister)))
	mux.Handle("POST /api/v1/users/me/mfa/webauthn/register/finish", authMW.Authenticate(http.HandlerFunc(mfaHandler.FinishWebAuthnRegister)))
	mux.Handle("GET /api/v1/users/me/mfa/methods", authMW.Authenticate(http.HandlerFunc(mfaHandler.ListMethods)))
	mux.Handle("DELETE /api/v1/users/me/mfa/methods/{id}", authMW.Authenticate(http.HandlerFunc(mfaHandler.DeleteMethod)))
	mux.Handle("PUT /api/v1/users/me/mfa/webauthn/{id}/passwordless", authMW.Authenticate(http.HandlerFunc(mfaHandler.TogglePasswordless)))
	mux.Handle("POST /api/v1/users/me/mfa/recovery-codes", authMW.Authenticate(http.HandlerFunc(mfaHandler.RegenerateRecoveryCodes)))

	// Devices - self (authenticated)
	mux.Handle("GET /api/v1/users/me/devices", authMW.Authenticate(http.HandlerFunc(deviceHandler.List)))
	mux.Handle("POST /api/v1/users/me/devices", authMW.Authenticate(http.HandlerFunc(deviceHandler.Create)))
	mux.Handle("POST /api/v1/users/me/devices/resolve", authMW.Authenticate(http.HandlerFunc(deviceHandler.Resolve)))
	mux.Handle("POST /api/v1/users/me/devices/{id}/claim", authMW.Authenticate(http.HandlerFunc(deviceHandler.Claim)))
	mux.Handle("PUT /api/v1/users/me/devices/{id}/primary", authMW.Authenticate(http.HandlerFunc(deviceHandler.SetPrimary)))
	mux.Handle("PUT /api/v1/devices/{id}", authMW.Authenticate(http.HandlerFunc(deviceHandler.Update)))
	mux.Handle("DELETE /api/v1/devices/{id}", authMW.Authenticate(http.HandlerFunc(deviceHandler.Delete)))

	// API tokens - self (authenticated)
	mux.Handle("POST /api/v1/users/me/api-tokens", authMW.Authenticate(http.HandlerFunc(apiTokenHandler.Create)))
	mux.Handle("GET /api/v1/users/me/api-tokens", authMW.Authenticate(http.HandlerFunc(apiTokenHandler.List)))
	mux.Handle("DELETE /api/v1/users/me/api-tokens/{id}", authMW.Authenticate(http.HandlerFunc(apiTokenHandler.Delete)))

	// Users - admin (includes MFA reset)
	mux.Handle("DELETE /api/v1/users/{id}/mfa", authMW.RequireAdmin(http.HandlerFunc(mfaHandler.AdminResetMFA)))
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

	// Groups - marker settings (authenticated, permission checked in service: group admin or system admin)
	mux.Handle("PUT /api/v1/groups/{id}/marker", authMW.Authenticate(http.HandlerFunc(groupHandler.UpdateMarker)))

	// Groups - CRUD (admin)
	mux.Handle("GET /api/v1/groups", authMW.RequireAdmin(http.HandlerFunc(groupHandler.List)))
	mux.Handle("POST /api/v1/groups", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Create)))
	mux.Handle("GET /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Get)))
	mux.Handle("PUT /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Update)))
	mux.Handle("DELETE /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Delete)))

	// Locations - group history (authenticated, permission checked in service)
	mux.Handle("GET /api/v1/groups/{id}/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetGroupHistory)))

	// Locations - visible history (authenticated, permission checked in service)
	mux.Handle("GET /api/v1/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetVisibleHistory)))

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

	// Terrain configs - CRUD (admin)
	mux.Handle("GET /api/v1/terrain-configs", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.List)))
	mux.Handle("POST /api/v1/terrain-configs", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.Create)))
	mux.Handle("GET /api/v1/terrain-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.Get)))
	mux.Handle("PUT /api/v1/terrain-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.Update)))
	mux.Handle("DELETE /api/v1/terrain-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.Delete)))

	// Messages (authenticated, permission checked in service)
	mux.Handle("POST /api/v1/messages", authMW.Authenticate(http.HandlerFunc(messageHandler.Send)))
	mux.Handle("GET /api/v1/groups/{id}/messages", authMW.Authenticate(http.HandlerFunc(messageHandler.ListGroupMessages)))
	mux.Handle("GET /api/v1/messages/conversations", authMW.Authenticate(http.HandlerFunc(messageHandler.ListDMConversations)))
	mux.Handle("GET /api/v1/messages/direct/{userId}", authMW.Authenticate(http.HandlerFunc(messageHandler.ListDirectMessages)))
	mux.Handle("GET /api/v1/messages/{id}", authMW.Authenticate(http.HandlerFunc(messageHandler.GetMessage)))
	mux.Handle("DELETE /api/v1/messages/{id}", authMW.Authenticate(http.HandlerFunc(messageHandler.DeleteMessage)))
	mux.Handle("GET /api/v1/attachments/{id}/download", authMW.AuthenticateWithQueryToken(http.HandlerFunc(messageHandler.DownloadAttachment)))

	// Drawings (authenticated, permission checked in service)
	mux.Handle("POST /api/v1/drawings", authMW.Authenticate(http.HandlerFunc(drawingHandler.Create)))
	mux.Handle("GET /api/v1/drawings", authMW.Authenticate(http.HandlerFunc(drawingHandler.ListOwn)))
	mux.Handle("GET /api/v1/drawings/shared", authMW.Authenticate(http.HandlerFunc(drawingHandler.ListShared)))
	mux.Handle("GET /api/v1/drawings/{id}", authMW.Authenticate(http.HandlerFunc(drawingHandler.Get)))
	mux.Handle("PUT /api/v1/drawings/{id}", authMW.Authenticate(http.HandlerFunc(drawingHandler.Update)))
	mux.Handle("DELETE /api/v1/drawings/{id}", authMW.Authenticate(http.HandlerFunc(drawingHandler.Delete)))
	mux.Handle("GET /api/v1/drawings/{id}/shares", authMW.Authenticate(http.HandlerFunc(drawingHandler.ListShares)))
	mux.Handle("DELETE /api/v1/drawings/{id}/shares/{messageId}", authMW.Authenticate(http.HandlerFunc(drawingHandler.Unshare)))
	mux.Handle("POST /api/v1/drawings/{id}/share", authMW.Authenticate(http.HandlerFunc(drawingHandler.Share)))

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

	// Location history - specific user (authenticated, permission checked in service)
	mux.Handle("GET /api/v1/users/{userId}/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetUserHistory)))

	// Server settings (admin)
	mux.Handle("GET /api/v1/server/settings", authMW.RequireAdmin(http.HandlerFunc(serverSettingsHandler.GetSettings)))
	mux.Handle("PUT /api/v1/server/settings", authMW.RequireAdmin(http.HandlerFunc(serverSettingsHandler.UpdateSettings)))

	// Permission policy (admin — hardcoded, never subject to matrix)
	mux.Handle("GET /api/v1/server/permissions", authMW.RequireAdmin(http.HandlerFunc(permissionPolicyHandler.GetPolicy)))
	mux.Handle("PUT /api/v1/server/permissions", authMW.RequireAdmin(http.HandlerFunc(permissionPolicyHandler.UpdatePolicy)))

	// CoT (Cursor on Target) - authenticated
	mux.Handle("POST /api/v1/cot/events", authMW.Authenticate(http.HandlerFunc(cotHandler.IngestEvents)))
	mux.Handle("GET /api/v1/cot/events", authMW.Authenticate(http.HandlerFunc(cotHandler.ListEvents)))
	mux.Handle("GET /api/v1/cot/events/{uid}", authMW.Authenticate(http.HandlerFunc(cotHandler.GetLatestByUID)))

	// Garmin InReach feeds (admin)
	mux.Handle("POST /api/v1/garmin/inreach/feeds", authMW.RequireAdmin(http.HandlerFunc(garminInReachHandler.Create)))
	mux.Handle("GET /api/v1/garmin/inreach/feeds", authMW.RequireAdmin(http.HandlerFunc(garminInReachHandler.List)))
	mux.Handle("GET /api/v1/garmin/inreach/feeds/{id}", authMW.RequireAdmin(http.HandlerFunc(garminInReachHandler.Get)))
	mux.Handle("PUT /api/v1/garmin/inreach/feeds/{id}", authMW.RequireAdmin(http.HandlerFunc(garminInReachHandler.Update)))
	mux.Handle("DELETE /api/v1/garmin/inreach/feeds/{id}", authMW.RequireAdmin(http.HandlerFunc(garminInReachHandler.Delete)))
	mux.Handle("POST /api/v1/garmin/inreach/poll", authMW.RequireAdmin(http.HandlerFunc(garminInReachHandler.Poll)))

	// Garmin InReach webhook (authenticated — Garmin Explore outbound)
	mux.Handle("POST /api/v1/webhooks/garmin/inreach/{mapshareId}", authMW.Authenticate(http.HandlerFunc(garminInReachHandler.Webhook)))

	// -----------------------------------------------------------------------
	// Apply global middleware and start server
	// -----------------------------------------------------------------------
	auditMW := middleware.Audit(auditService, jwtService)

	// MFA enforcement middleware: when mfa_required is enabled server-wide,
	// block non-MFA-setup requests for users without MFA configured.
	mfaChecker := &mfaSetupChecker{settingsRepo: serverSettingsRepo}
	getUserMFAEnabled := func(ctx context.Context, userID uuid.UUID) (bool, error) {
		u, err := userRepo.GetByID(ctx, userID)
		if err != nil {
			return false, err
		}
		return u.MFAEnabled, nil
	}
	mfaEnforcementMW := authMW.RequireMFASetup(mfaChecker, getUserMFAEnabled)

	var rootHandler http.Handler = mux
	rootHandler = mfaEnforcementMW(rootHandler)
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

// mfaSetupChecker adapts ServerSettingsRepo to the middleware.MFASetupChecker interface.
type mfaSetupChecker struct {
	settingsRepo repository.ServerSettingsRepo
}

func (c *mfaSetupChecker) IsMFARequired(ctx context.Context) bool {
	s, err := c.settingsRepo.Get(ctx, "mfa_required")
	if err != nil {
		return false
	}
	return s.Value == "true"
}

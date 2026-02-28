// Package testutil provides shared test infrastructure for integration tests.
// It spins up PostgreSQL (PostGIS), Redis, and MinIO containers using
// testcontainers-go and wires the full application stack (repos, services,
// handlers, middleware) into an httptest.Server.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	miniomod "github.com/testcontainers/testcontainers-go/modules/minio"
	pgmod "github.com/testcontainers/testcontainers-go/modules/postgres"
	redismod "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

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

// TestEnv holds a fully-wired test server and all its dependencies.
type TestEnv struct {
	Server *httptest.Server

	// Direct access for test helpers
	Pool   *pgxpool.Pool
	Redis  redis.UniversalClient
	JWT    *auth.JWTService
	Config *config.Config

	// Repositories (exposed for direct DB manipulation in tests)
	UserRepo           *repository.UserRepository
	TokenRepo          *repository.TokenRepository
	GroupRepo          *repository.GroupRepository
	DeviceRepo         *repository.DeviceRepository
	LocationRepo       *repository.LocationRepository
	MessageRepo        *repository.MessageRepository
	DrawingRepo        *repository.DrawingRepository
	AuditRepo          *repository.AuditRepository
	CotRepo            *repository.CotRepository
	MapConfigRepo      *repository.MapConfigRepository
	TerrainConfigRepo  *repository.TerrainConfigRepository
	MFARepo            *repository.MFARepository
	ServerSettingsRepo *repository.ServerSettingsRepository

	// Services
	AuthService *service.AuthService
	UserService *service.UserService
	MFAService  *service.MFAService
	LocationSvc *service.LocationService

	// Cleanup
	cleanup []func()
}

// Setup creates containers for PostGIS, Redis, and MinIO, runs migrations,
// wires the full application, and returns a running httptest.Server.
// Call env.Teardown() when done. If sharing across tests via TestMain,
// do NOT rely on t.Cleanup — call Teardown explicitly in TestMain.
func Setup(t *testing.T) *TestEnv {
	t.Helper()
	ctx := context.Background()

	// Suppress noisy log output during tests
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	env := &TestEnv{}

	// ---------------------------------------------------------------
	// PostgreSQL (PostGIS)
	// ---------------------------------------------------------------
	pgContainer, err := pgmod.Run(ctx,
		"postgis/postgis:16-3.4-alpine",
		pgmod.WithDatabase("testdb"),
		pgmod.WithUsername("testuser"),
		pgmod.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	env.cleanup = append(env.cleanup, func() { pgContainer.Terminate(ctx) })

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	// Verify connectivity before running migrations
	pool, err := pgxpool.New(ctx, pgDSN)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	env.cleanup = append(env.cleanup, func() { pool.Close() })
	for i := range 10 {
		if err := pool.Ping(ctx); err == nil {
			break
		}
		if i == 9 {
			t.Fatalf("postgres not ready after 10 retries")
		}
		time.Sleep(500 * time.Millisecond)
	}
	env.Pool = pool

	// Run migrations
	if err := database.RunMigrations(pgDSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// ---------------------------------------------------------------
	// Redis
	// ---------------------------------------------------------------
	redisContainer, err := redismod.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	env.cleanup = append(env.cleanup, func() { redisContainer.Terminate(ctx) })

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("redis endpoint: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisEndpoint})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	env.cleanup = append(env.cleanup, func() { rdb.Close() })
	env.Redis = rdb

	// ---------------------------------------------------------------
	// MinIO (S3-compatible object storage)
	// ---------------------------------------------------------------
	minioContainer, err := miniomod.Run(ctx, "minio/minio:latest")
	if err != nil {
		t.Fatalf("start minio container: %v", err)
	}
	env.cleanup = append(env.cleanup, func() { minioContainer.Terminate(ctx) })

	minioEndpoint, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("minio endpoint: %v", err)
	}
	if !strings.HasPrefix(minioEndpoint, "http") {
		minioEndpoint = "http://" + minioEndpoint
	}

	// Create S3 client pointing at MinIO
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""),
		),
	)
	if err != nil {
		t.Fatalf("aws config: %v", err)
	}
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(minioEndpoint)
		o.UsePathStyle = true
	})

	// Create test bucket
	bucketName := "test-sitaware"
	_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		t.Fatalf("create test bucket: %v", err)
	}
	storageSvc := storage.NewStorageService(s3Client, bucketName)

	// ---------------------------------------------------------------
	// Configuration
	// ---------------------------------------------------------------
	jwtSecret := "test-jwt-secret-that-is-long-enough-for-hmac-sha256"
	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "AdminPass123!",
			Email:    "admin@test.local",
		},
		JWT: config.JWTConfig{
			Secret:          jwtSecret,
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
		},
		WS: config.WSConfig{
			LocationThrottle: 0, // no throttle in tests
		},
		Map: config.MapConfig{
			DefaultCenterLat: 0,
			DefaultCenterLng: 0,
			DefaultZoom:      2,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
		RateLimit: config.RateLimitConfig{
			RPS:   1000, // effectively unlimited in tests
			Burst: 2000,
		},
		MFA: config.MFAConfig{},
		WebAuthn: config.WebAuthnConfig{
			RPID:          "localhost",
			RPDisplayName: "SitAware Test",
			RPOrigins:     []string{"http://localhost"},
		},
		Security: config.SecurityConfig{
			MaxRequestBodyBytes: 10 << 20,
		},
		TokenCleanupInterval: 1 * time.Hour,
	}
	env.Config = cfg

	// ---------------------------------------------------------------
	// Repositories
	// ---------------------------------------------------------------
	env.UserRepo = repository.NewUserRepository(pool)
	env.TokenRepo = repository.NewTokenRepository(pool)
	env.GroupRepo = repository.NewGroupRepository(pool)
	env.DeviceRepo = repository.NewDeviceRepository(pool)
	env.LocationRepo = repository.NewLocationRepository(pool)
	env.MessageRepo = repository.NewMessageRepository(pool)
	env.DrawingRepo = repository.NewDrawingRepository(pool)
	env.AuditRepo = repository.NewAuditRepository(pool)
	env.CotRepo = repository.NewCotRepository(pool)
	env.MapConfigRepo = repository.NewMapConfigRepository(pool)
	env.TerrainConfigRepo = repository.NewTerrainConfigRepository(pool)
	env.MFARepo = repository.NewMFARepository(pool)
	env.ServerSettingsRepo = repository.NewServerSettingsRepository(pool)

	// ---------------------------------------------------------------
	// Services
	// ---------------------------------------------------------------
	jwtService := auth.NewJWTService(cfg.JWT)
	env.JWT = jwtService

	encryptor, err := auth.NewLocalEncryptor([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}

	waConfig := &webauthn.Config{
		RPDisplayName: cfg.WebAuthn.RPDisplayName,
		RPID:          cfg.WebAuthn.RPID,
		RPOrigins:     cfg.WebAuthn.RPOrigins,
	}
	wa, err := webauthn.New(waConfig)
	if err != nil {
		t.Fatalf("create webauthn: %v", err)
	}

	ps := pubsub.NewRedisPubSub(rdb)
	env.cleanup = append(env.cleanup, func() { ps.Close() })

	mfaService := service.NewMFAService(env.MFARepo, env.UserRepo, encryptor, rdb, wa)
	env.MFAService = mfaService
	authService := service.NewAuthService(env.UserRepo, env.TokenRepo, jwtService, mfaService)
	env.AuthService = authService
	userService := service.NewUserService(env.UserRepo, env.TokenRepo, storageSvc)
	env.UserService = userService
	groupService := service.NewGroupService(env.GroupRepo, env.UserRepo)
	locationService := service.NewLocationService(env.LocationRepo, env.GroupRepo, ps, cfg.WS.LocationThrottle)
	env.LocationSvc = locationService
	mapConfigService := service.NewMapConfigService(env.MapConfigRepo, env.TerrainConfigRepo, env.ServerSettingsRepo, cfg.Map)
	terrainConfigService := service.NewTerrainConfigService(env.TerrainConfigRepo, cfg.Map)
	messageService := service.NewMessageService(env.MessageRepo, env.GroupRepo, storageSvc, ps)
	drawingService := service.NewDrawingService(env.DrawingRepo, env.MessageRepo, env.GroupRepo, ps)
	auditService := service.NewAuditService(env.AuditRepo, env.GroupRepo)
	cotService := service.NewCotService(env.CotRepo, env.DeviceRepo, env.UserRepo, env.GroupRepo, locationService)

	// Bootstrap admin user
	if err := authService.BootstrapAdmin(ctx, cfg.Admin); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	// Bootstrap built-in map and terrain configs
	if err := mapConfigService.BootstrapMapConfigs(ctx); err != nil {
		t.Fatalf("bootstrap map configs: %v", err)
	}
	if err := terrainConfigService.BootstrapTerrainConfigs(ctx); err != nil {
		t.Fatalf("bootstrap terrain configs: %v", err)
	}

	// ---------------------------------------------------------------
	// Handlers
	// ---------------------------------------------------------------
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService, storageSvc)
	deviceHandler := handler.NewDeviceHandler(env.DeviceRepo)
	groupHandler := handler.NewGroupHandler(groupService)
	mapConfigHandler := handler.NewMapConfigHandler(mapConfigService)
	terrainConfigHandler := handler.NewTerrainConfigHandler(terrainConfigService)
	locationHandler := handler.NewLocationHandler(locationService)
	messageHandler := handler.NewMessageHandler(messageService)
	drawingHandler := handler.NewDrawingHandler(drawingService)
	auditHandler := handler.NewAuditHandler(auditService)
	cotHandler := handler.NewCotHandler(cotService)
	mfaHandler := handler.NewMFAHandler(mfaService, authService)
	serverSettingsHandler := handler.NewServerSettingsHandler(env.ServerSettingsRepo)
	apiTokenRepo := repository.NewAPITokenRepository(pool)
	apiTokenService := service.NewAPITokenService(apiTokenRepo)
	apiTokenHandler := handler.NewAPITokenHandler(apiTokenService)

	// WebSocket hub
	hub := ws.NewHub(ps, locationService, env.GroupRepo, env.UserRepo)
	hubCtx, hubCancel := context.WithCancel(ctx)
	env.cleanup = append(env.cleanup, hubCancel)
	go hub.Run(hubCtx)
	wsHandler := ws.NewHandler(hub, jwtService, apiTokenService, env.DeviceRepo, env.GroupRepo)

	// ---------------------------------------------------------------
	// Middleware
	// ---------------------------------------------------------------
	authMW := middleware.NewAuth(jwtService, apiTokenService)

	// ---------------------------------------------------------------
	// Router (mirrors cmd/server/main.go exactly)
	// ---------------------------------------------------------------
	mux := http.NewServeMux()

	// Health checks
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unavailable","error":"%s"}`, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ready"}`)
	})

	// API info
	mux.HandleFunc("GET /api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"service":"sitaware-api","version":"dev"}`)
	})

	// WebSocket
	mux.Handle("GET /api/v1/ws", wsHandler)

	// Auth (public)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)

	// Auth (authenticated)
	mux.Handle("POST /api/v1/auth/logout", authMW.Authenticate(http.HandlerFunc(authHandler.Logout)))

	// MFA login challenge (public)
	mux.HandleFunc("POST /api/v1/auth/mfa/totp", mfaHandler.MFAVerifyTOTP)
	mux.HandleFunc("POST /api/v1/auth/mfa/recovery", mfaHandler.MFARecovery)
	mux.HandleFunc("POST /api/v1/auth/mfa/webauthn/begin", mfaHandler.MFAWebAuthnBegin)
	mux.HandleFunc("POST /api/v1/auth/mfa/webauthn/finish", mfaHandler.MFAWebAuthnFinish)

	// Passkey (public)
	mux.HandleFunc("POST /api/v1/auth/passkey/begin", mfaHandler.PasskeyBegin)
	mux.HandleFunc("POST /api/v1/auth/passkey/finish", mfaHandler.PasskeyFinish)

	// Users - self
	mux.Handle("GET /api/v1/users/me", authMW.Authenticate(http.HandlerFunc(userHandler.GetMe)))
	mux.Handle("PUT /api/v1/users/me", authMW.Authenticate(http.HandlerFunc(userHandler.UpdateMe)))
	mux.Handle("PUT /api/v1/users/me/password", authMW.Authenticate(http.HandlerFunc(userHandler.ChangePassword)))
	mux.Handle("PUT /api/v1/users/me/avatar", authMW.Authenticate(http.HandlerFunc(userHandler.UploadAvatar)))
	mux.Handle("DELETE /api/v1/users/me/avatar", authMW.Authenticate(http.HandlerFunc(userHandler.DeleteAvatar)))
	mux.Handle("GET /api/v1/users/{id}/avatar", authMW.AuthenticateWithQueryToken(http.HandlerFunc(userHandler.ServeAvatar)))

	// MFA setup
	mux.Handle("POST /api/v1/users/me/mfa/totp/setup", authMW.Authenticate(http.HandlerFunc(mfaHandler.SetupTOTP)))
	mux.Handle("POST /api/v1/users/me/mfa/totp/verify", authMW.Authenticate(http.HandlerFunc(mfaHandler.VerifyTOTPSetup)))
	mux.Handle("POST /api/v1/users/me/mfa/webauthn/register/begin", authMW.Authenticate(http.HandlerFunc(mfaHandler.BeginWebAuthnRegister)))
	mux.Handle("POST /api/v1/users/me/mfa/webauthn/register/finish", authMW.Authenticate(http.HandlerFunc(mfaHandler.FinishWebAuthnRegister)))
	mux.Handle("GET /api/v1/users/me/mfa/methods", authMW.Authenticate(http.HandlerFunc(mfaHandler.ListMethods)))
	mux.Handle("DELETE /api/v1/users/me/mfa/methods/{id}", authMW.Authenticate(http.HandlerFunc(mfaHandler.DeleteMethod)))
	mux.Handle("PUT /api/v1/users/me/mfa/webauthn/{id}/passwordless", authMW.Authenticate(http.HandlerFunc(mfaHandler.TogglePasswordless)))
	mux.Handle("POST /api/v1/users/me/mfa/recovery-codes", authMW.Authenticate(http.HandlerFunc(mfaHandler.RegenerateRecoveryCodes)))

	// Devices - self
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

	// Users - admin
	mux.Handle("DELETE /api/v1/users/{id}/mfa", authMW.RequireAdmin(http.HandlerFunc(mfaHandler.AdminResetMFA)))
	mux.Handle("GET /api/v1/users", authMW.RequireAdmin(http.HandlerFunc(userHandler.List)))
	mux.Handle("POST /api/v1/users", authMW.RequireAdmin(http.HandlerFunc(userHandler.Create)))
	mux.Handle("GET /api/v1/users/{id}", authMW.RequireAdmin(http.HandlerFunc(userHandler.Get)))
	mux.Handle("PUT /api/v1/users/{id}", authMW.RequireAdmin(http.HandlerFunc(userHandler.Update)))
	mux.Handle("DELETE /api/v1/users/{id}", authMW.RequireAdmin(http.HandlerFunc(userHandler.Delete)))

	// Groups - my groups
	mux.Handle("GET /api/v1/users/me/groups", authMW.Authenticate(http.HandlerFunc(groupHandler.ListMyGroups)))

	// Groups - members
	mux.Handle("GET /api/v1/groups/{id}/members", authMW.Authenticate(http.HandlerFunc(groupHandler.ListMembers)))
	mux.Handle("POST /api/v1/groups/{id}/members", authMW.Authenticate(http.HandlerFunc(groupHandler.AddMember)))
	mux.Handle("PUT /api/v1/groups/{id}/members/{userId}", authMW.Authenticate(http.HandlerFunc(groupHandler.UpdateMember)))
	mux.Handle("DELETE /api/v1/groups/{id}/members/{userId}", authMW.Authenticate(http.HandlerFunc(groupHandler.RemoveMember)))

	// Groups - marker
	mux.Handle("PUT /api/v1/groups/{id}/marker", authMW.Authenticate(http.HandlerFunc(groupHandler.UpdateMarker)))

	// Groups - admin CRUD
	mux.Handle("GET /api/v1/groups", authMW.RequireAdmin(http.HandlerFunc(groupHandler.List)))
	mux.Handle("POST /api/v1/groups", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Create)))
	mux.Handle("GET /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Get)))
	mux.Handle("PUT /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Update)))
	mux.Handle("DELETE /api/v1/groups/{id}", authMW.RequireAdmin(http.HandlerFunc(groupHandler.Delete)))

	// Locations
	mux.Handle("GET /api/v1/groups/{id}/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetGroupHistory)))
	mux.Handle("GET /api/v1/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetVisibleHistory)))
	mux.Handle("GET /api/v1/locations", authMW.RequireAdmin(http.HandlerFunc(locationHandler.GetAllLocations)))

	// Map settings
	mux.Handle("GET /api/v1/map/settings", authMW.Authenticate(http.HandlerFunc(mapConfigHandler.GetSettings)))

	// Map configs - admin
	mux.Handle("GET /api/v1/map-configs", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.List)))
	mux.Handle("POST /api/v1/map-configs", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.Create)))
	mux.Handle("GET /api/v1/map-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.Get)))
	mux.Handle("PUT /api/v1/map-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.Update)))
	mux.Handle("DELETE /api/v1/map-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(mapConfigHandler.Delete)))

	// Terrain configs - admin
	mux.Handle("GET /api/v1/terrain-configs", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.List)))
	mux.Handle("POST /api/v1/terrain-configs", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.Create)))
	mux.Handle("GET /api/v1/terrain-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.Get)))
	mux.Handle("PUT /api/v1/terrain-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.Update)))
	mux.Handle("DELETE /api/v1/terrain-configs/{id}", authMW.RequireAdmin(http.HandlerFunc(terrainConfigHandler.Delete)))

	// Messages
	mux.Handle("POST /api/v1/messages", authMW.Authenticate(http.HandlerFunc(messageHandler.Send)))
	mux.Handle("GET /api/v1/groups/{id}/messages", authMW.Authenticate(http.HandlerFunc(messageHandler.ListGroupMessages)))
	mux.Handle("GET /api/v1/messages/conversations", authMW.Authenticate(http.HandlerFunc(messageHandler.ListDMConversations)))
	mux.Handle("GET /api/v1/messages/direct/{userId}", authMW.Authenticate(http.HandlerFunc(messageHandler.ListDirectMessages)))
	mux.Handle("GET /api/v1/messages/{id}", authMW.Authenticate(http.HandlerFunc(messageHandler.GetMessage)))
	mux.Handle("DELETE /api/v1/messages/{id}", authMW.Authenticate(http.HandlerFunc(messageHandler.DeleteMessage)))
	mux.Handle("GET /api/v1/attachments/{id}/download", authMW.AuthenticateWithQueryToken(http.HandlerFunc(messageHandler.DownloadAttachment)))

	// Drawings
	mux.Handle("POST /api/v1/drawings", authMW.Authenticate(http.HandlerFunc(drawingHandler.Create)))
	mux.Handle("GET /api/v1/drawings", authMW.Authenticate(http.HandlerFunc(drawingHandler.ListOwn)))
	mux.Handle("GET /api/v1/drawings/shared", authMW.Authenticate(http.HandlerFunc(drawingHandler.ListShared)))
	mux.Handle("GET /api/v1/drawings/{id}", authMW.Authenticate(http.HandlerFunc(drawingHandler.Get)))
	mux.Handle("PUT /api/v1/drawings/{id}", authMW.Authenticate(http.HandlerFunc(drawingHandler.Update)))
	mux.Handle("DELETE /api/v1/drawings/{id}", authMW.Authenticate(http.HandlerFunc(drawingHandler.Delete)))
	mux.Handle("GET /api/v1/drawings/{id}/shares", authMW.Authenticate(http.HandlerFunc(drawingHandler.ListShares)))
	mux.Handle("DELETE /api/v1/drawings/{id}/shares/{messageId}", authMW.Authenticate(http.HandlerFunc(drawingHandler.Unshare)))
	mux.Handle("POST /api/v1/drawings/{id}/share", authMW.Authenticate(http.HandlerFunc(drawingHandler.Share)))

	// Audit logs - self
	mux.Handle("GET /api/v1/audit-logs/me", authMW.Authenticate(http.HandlerFunc(auditHandler.GetMyLogs)))
	mux.Handle("GET /api/v1/audit-logs/me/export", authMW.Authenticate(http.HandlerFunc(auditHandler.ExportMyLogs)))

	// Audit logs - group
	mux.Handle("GET /api/v1/groups/{id}/audit-logs", authMW.Authenticate(http.HandlerFunc(auditHandler.GetGroupLogs)))

	// Audit logs - admin
	mux.Handle("GET /api/v1/audit-logs", authMW.RequireAdmin(http.HandlerFunc(auditHandler.GetAllLogs)))
	mux.Handle("GET /api/v1/audit-logs/export", authMW.RequireAdmin(http.HandlerFunc(auditHandler.ExportAllLogs)))

	// Location history - self
	mux.Handle("GET /api/v1/users/me/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetMyHistory)))
	mux.Handle("GET /api/v1/users/me/locations/export", authMW.Authenticate(http.HandlerFunc(locationHandler.ExportGPX)))

	// Location history - specific user
	mux.Handle("GET /api/v1/users/{userId}/locations/history", authMW.Authenticate(http.HandlerFunc(locationHandler.GetUserHistory)))

	// Server settings - admin
	mux.Handle("GET /api/v1/server/settings", authMW.RequireAdmin(http.HandlerFunc(serverSettingsHandler.GetSettings)))
	mux.Handle("PUT /api/v1/server/settings", authMW.RequireAdmin(http.HandlerFunc(serverSettingsHandler.UpdateSettings)))

	// CoT
	mux.Handle("POST /api/v1/cot/events", authMW.Authenticate(http.HandlerFunc(cotHandler.IngestEvents)))
	mux.Handle("GET /api/v1/cot/events", authMW.Authenticate(http.HandlerFunc(cotHandler.ListEvents)))
	mux.Handle("GET /api/v1/cot/events/{uid}", authMW.Authenticate(http.HandlerFunc(cotHandler.GetLatestByUID)))

	// ---------------------------------------------------------------
	// Global middleware stack (same order as main.go, but skip MFA
	// enforcement for simplicity — it can be tested separately)
	// ---------------------------------------------------------------
	auditMW := middleware.Audit(auditService, jwtService)

	var rootHandler http.Handler = mux
	rootHandler = auditMW(rootHandler)
	rootHandler = middleware.CORS(cfg.CORS)(rootHandler)

	env.Server = httptest.NewServer(rootHandler)
	env.cleanup = append(env.cleanup, env.Server.Close)

	return env
}

// Teardown shuts down all containers and the test server.
func (e *TestEnv) Teardown() {
	// Run cleanup in reverse order
	for i := len(e.cleanup) - 1; i >= 0; i-- {
		e.cleanup[i]()
	}
}

// ---------------------------------------------------------------------------
// Helper types and functions
// ---------------------------------------------------------------------------

// AuthTokens holds access and refresh tokens from a login.
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// LoginAdmin logs in with the bootstrap admin credentials and returns tokens.
func (e *TestEnv) LoginAdmin(t *testing.T) AuthTokens {
	t.Helper()
	return e.Login(t, e.Config.Admin.Username, e.Config.Admin.Password)
}

// Login authenticates with the given credentials and returns tokens.
func (e *TestEnv) Login(t *testing.T, username, password string) AuthTokens {
	t.Helper()
	body := map[string]string{
		"username": username,
		"password": password,
	}
	resp := e.DoJSON(t, "POST", "/api/v1/auth/login", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Login(%s) status=%d body=%s", username, resp.StatusCode, string(b))
	}

	var tokens AuthTokens
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return tokens
}

// CreateUser creates a user via the admin API and returns the user ID.
func (e *TestEnv) CreateUser(t *testing.T, adminToken, username, email, password string, isAdmin bool) uuid.UUID {
	t.Helper()
	body := map[string]any{
		"username": username,
		"email":    email,
		"password": password,
		"is_admin": isAdmin,
	}
	resp := e.DoJSON(t, "POST", "/api/v1/users", body, adminToken)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("CreateUser(%s) status=%d body=%s", username, resp.StatusCode, string(b))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode create user response: %v", err)
	}
	id, err := uuid.Parse(result.ID)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	return id
}

// CreateGroup creates a group via the admin API and returns the group ID.
func (e *TestEnv) CreateGroup(t *testing.T, adminToken, name string) uuid.UUID {
	t.Helper()
	body := map[string]string{"name": name}
	resp := e.DoJSON(t, "POST", "/api/v1/groups", body, adminToken)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("CreateGroup(%s) status=%d body=%s", name, resp.StatusCode, string(b))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode create group response: %v", err)
	}
	id, err := uuid.Parse(result.ID)
	if err != nil {
		t.Fatalf("parse group id: %v", err)
	}
	return id
}

// AddGroupMember adds a user to a group with full permissions.
func (e *TestEnv) AddGroupMember(t *testing.T, token string, groupID, userID uuid.UUID, role string) {
	t.Helper()
	canRead := true
	canWrite := true
	isGroupAdmin := role == "admin"
	body := map[string]any{
		"user_id":        userID.String(),
		"can_read":       canRead,
		"can_write":      canWrite,
		"is_group_admin": isGroupAdmin,
	}
	resp := e.DoJSON(t, "POST", fmt.Sprintf("/api/v1/groups/%s/members", groupID), body, token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("AddGroupMember status=%d body=%s", resp.StatusCode, string(b))
	}
}

// DoMultipartForm sends a multipart/form-data request with the given fields.
func (e *TestEnv) DoMultipartForm(t *testing.T, method, path string, fields map[string]string, token string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	writer.Close()
	return e.Do(t, method, path, writer.FormDataContentType(), &buf, token)
}

// DoJSON performs an HTTP request with JSON body and optional bearer token.
func (e *TestEnv) DoJSON(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	return e.Do(t, method, path, "application/json", bodyReader, token)
}

// Do performs an HTTP request with optional content type and bearer token.
func (e *TestEnv) Do(t *testing.T, method, path, contentType string, body io.Reader, token string) *http.Response {
	t.Helper()
	url := e.Server.URL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, path, err)
	}
	return resp
}

// AuthHeader returns "Bearer <token>" for use in manual requests.
func AuthHeader(token string) string {
	return "Bearer " + token
}

// DecodeJSON decodes a response body into the target.
func DecodeJSON(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// RequireStatus checks the response status code.
func RequireStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(b))
	}
}

// init sets log output to discard during tests to reduce noise.
func init() {
	if os.Getenv("TEST_VERBOSE") == "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
}

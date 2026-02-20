package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the API service.
type Config struct {
	// Admin bootstrap
	Admin AdminConfig

	// HTTP server
	Server ServerConfig

	// JWT authentication
	JWT JWTConfig

	// PostgreSQL
	DB DBConfig

	// Redis pub/sub
	Redis RedisConfig

	// S3-compatible object storage
	S3 S3Config

	// WebSocket
	WS WSConfig

	// Map defaults
	Map MapConfig

	// CORS
	CORS CORSConfig

	// Rate limiting
	RateLimit RateLimitConfig

	// Security
	Security SecurityConfig

	// Token cleanup
	TokenCleanupInterval time.Duration
}

type WSConfig struct {
	LocationThrottle time.Duration
}

type AdminConfig struct {
	Username string
	Password string
	Email    string
}

type ServerConfig struct {
	Host     string
	Port     int
	LogLevel string
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN returns a PostgreSQL connection string.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

// Addr returns host:port for Redis.
func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type S3Config struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	Bucket       string
	Region       string
	UsePathStyle bool
}

type MapConfig struct {
	DefaultTileURL   string
	DefaultCenterLat float64
	DefaultCenterLng float64
	DefaultZoom      int
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	RPS   float64
	Burst int
}

type SecurityConfig struct {
	MaxRequestBodyBytes int64
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Admin: AdminConfig{
			Username: envStr("ADMIN_USERNAME", "admin"),
			Password: envStr("ADMIN_PASSWORD", "changeme"),
			Email:    envStr("ADMIN_EMAIL", "admin@sitaware.local"),
		},
		Server: ServerConfig{
			Host:     envStr("API_HOST", "0.0.0.0"),
			Port:     envInt("API_PORT", 8080),
			LogLevel: envStr("API_LOG_LEVEL", "debug"),
		},
		JWT: JWTConfig{
			Secret:          envStr("JWT_SECRET", "change-this-to-a-random-secret-in-production"),
			AccessTokenTTL:  envDuration("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: envDuration("JWT_REFRESH_TOKEN_TTL", 168*time.Hour),
		},
		DB: DBConfig{
			Host:     envStr("DB_HOST", "localhost"),
			Port:     envInt("DB_PORT", 5432),
			User:     envStr("DB_USER", "sitaware"),
			Password: envStr("DB_PASSWORD", "sitaware"),
			Name:     envStr("DB_NAME", "sitaware"),
			SSLMode:  envStr("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     envStr("REDIS_HOST", "localhost"),
			Port:     envInt("REDIS_PORT", 6379),
			Password: envStr("REDIS_PASSWORD", ""),
		},
		WS: WSConfig{
			LocationThrottle: envDuration("WS_LOCATION_THROTTLE", 1*time.Second),
		},
		S3: S3Config{
			Endpoint:     envStr("S3_ENDPOINT", "http://localhost:9000"),
			AccessKey:    envStr("S3_ACCESS_KEY", "sitaware"),
			SecretKey:    envStr("S3_SECRET_KEY", "sitaware123"),
			Bucket:       envStr("S3_BUCKET", "sitaware"),
			Region:       envStr("S3_REGION", "us-east-1"),
			UsePathStyle: envBool("S3_USE_PATH_STYLE", true),
		},
		Map: MapConfig{
			DefaultTileURL:   envStr("MAP_DEFAULT_TILE_URL", "https://tile.openstreetmap.org/{z}/{x}/{y}.png"),
			DefaultCenterLat: envFloat("MAP_DEFAULT_CENTER_LAT", 0),
			DefaultCenterLng: envFloat("MAP_DEFAULT_CENTER_LNG", 0),
			DefaultZoom:      envInt("MAP_DEFAULT_ZOOM", 2),
		},
		CORS: CORSConfig{
			AllowedOrigins: envStrSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
		},
		RateLimit: RateLimitConfig{
			RPS:   envFloat("RATE_LIMIT_RPS", 10),
			Burst: envInt("RATE_LIMIT_BURST", 20),
		},
		Security: SecurityConfig{
			MaxRequestBodyBytes: envInt64("MAX_REQUEST_BODY_BYTES", 10<<20), // 10MB
		},
		TokenCleanupInterval: envDuration("TOKEN_CLEANUP_INTERVAL", 1*time.Hour),
	}

	return cfg, nil
}

// Addr returns the listen address for the HTTP server.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func envStrSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}

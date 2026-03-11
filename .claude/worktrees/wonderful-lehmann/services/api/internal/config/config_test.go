package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might affect defaults
	envKeys := []string{
		"API_HOST", "API_PORT", "API_LOG_LEVEL",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"REDIS_HOST", "REDIS_PORT",
		"JWT_SECRET", "JWT_ACCESS_TOKEN_TTL", "JWT_REFRESH_TOKEN_TTL",
		"ADMIN_USERNAME", "ADMIN_PASSWORD",
	}
	saved := make(map[string]string)
	for _, k := range envKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = v
			os.Unsetenv(k)
		}
	}
	defer func() {
		for k, v := range saved {
			os.Setenv(k, v)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "localhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 5432)
	}
	if cfg.Redis.Host != "localhost" {
		t.Errorf("Redis.Host = %q, want %q", cfg.Redis.Host, "localhost")
	}
	if cfg.Admin.Username != "admin" {
		t.Errorf("Admin.Username = %q, want %q", cfg.Admin.Username, "admin")
	}
	if cfg.JWT.AccessTokenTTL != 15*time.Minute {
		t.Errorf("JWT.AccessTokenTTL = %v, want %v", cfg.JWT.AccessTokenTTL, 15*time.Minute)
	}
	if cfg.JWT.RefreshTokenTTL != 168*time.Hour {
		t.Errorf("JWT.RefreshTokenTTL = %v, want %v", cfg.JWT.RefreshTokenTTL, 168*time.Hour)
	}
	if cfg.RateLimit.RPS != 10 {
		t.Errorf("RateLimit.RPS = %f, want %f", cfg.RateLimit.RPS, 10.0)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Setenv("API_PORT", "9090")
	os.Setenv("DB_HOST", "db.example.com")
	defer os.Unsetenv("API_PORT")
	defer os.Unsetenv("DB_HOST")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.DB.Host != "db.example.com" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "db.example.com")
	}
}

func TestDBConfig_DSN(t *testing.T) {
	db := DBConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Name:     "testdb",
		SSLMode:  "disable",
	}

	dsn := db.DSN()
	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if dsn != expected {
		t.Errorf("DSN() = %q, want %q", dsn, expected)
	}
}

func TestRedisConfig_Addr(t *testing.T) {
	r := RedisConfig{Host: "redis.example.com", Port: 6380}
	if r.Addr() != "redis.example.com:6380" {
		t.Errorf("Addr() = %q, want %q", r.Addr(), "redis.example.com:6380")
	}
}

func TestConfig_Addr(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Host: "0.0.0.0", Port: 8080},
	}
	if cfg.Addr() != "0.0.0.0:8080" {
		t.Errorf("Addr() = %q, want %q", cfg.Addr(), "0.0.0.0:8080")
	}
}

func TestEnvStrSlice(t *testing.T) {
	os.Setenv("TEST_SLICE", "a, b , c")
	defer os.Unsetenv("TEST_SLICE")

	result := envStrSlice("TEST_SLICE", nil)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("result = %v, want [a b c]", result)
	}
}

func TestEnvStrSlice_Fallback(t *testing.T) {
	result := envStrSlice("NONEXISTENT_KEY", []string{"default"})
	if len(result) != 1 || result[0] != "default" {
		t.Errorf("result = %v, want [default]", result)
	}
}

func TestEnvDuration(t *testing.T) {
	os.Setenv("TEST_DUR", "30s")
	defer os.Unsetenv("TEST_DUR")

	result := envDuration("TEST_DUR", 0)
	if result != 30*time.Second {
		t.Errorf("result = %v, want 30s", result)
	}
}

func TestEnvDuration_InvalidFallback(t *testing.T) {
	os.Setenv("TEST_DUR_BAD", "not-a-duration")
	defer os.Unsetenv("TEST_DUR_BAD")

	result := envDuration("TEST_DUR_BAD", 5*time.Minute)
	if result != 5*time.Minute {
		t.Errorf("result = %v, want 5m", result)
	}
}

func TestEnvBool(t *testing.T) {
	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")

	result := envBool("TEST_BOOL", false)
	if !result {
		t.Error("result should be true")
	}
}

func TestEnvInt_Invalid(t *testing.T) {
	os.Setenv("TEST_INT_BAD", "abc")
	defer os.Unsetenv("TEST_INT_BAD")

	result := envInt("TEST_INT_BAD", 42)
	if result != 42 {
		t.Errorf("result = %d, want 42 (fallback)", result)
	}
}

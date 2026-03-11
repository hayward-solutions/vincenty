package main

import (
	"strings"
	"testing"
	"time"
)

// mockEnv returns a getenv function backed by the given map.
func mockEnv(vars map[string]string) func(string) string {
	return func(key string) string {
		return vars[key]
	}
}

// noEnv returns a getenv function that always returns "".
func noEnv() func(string) string {
	return func(string) string { return "" }
}

// ---------------------------------------------------------------------------
// Flags only (no env vars)
// ---------------------------------------------------------------------------

func TestParseConfig_FlagsOnly(t *testing.T) {
	args := []string{
		"-server", "https://api.example.com",
		"-token", "sat_abc123",
		"-file", "track.gpx",
		"-device-name", "my-device",
		"-speed", "2.5",
		"-interval", "500ms",
		"-loop",
		"-quiet",
	}
	cfg, err := parseConfig(args, noEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "https://api.example.com" {
		t.Errorf("Server = %q, want %q", cfg.Server, "https://api.example.com")
	}
	if cfg.Token != "sat_abc123" {
		t.Errorf("Token = %q, want %q", cfg.Token, "sat_abc123")
	}
	if cfg.File != "track.gpx" {
		t.Errorf("File = %q, want %q", cfg.File, "track.gpx")
	}
	if cfg.DeviceName != "my-device" {
		t.Errorf("DeviceName = %q, want %q", cfg.DeviceName, "my-device")
	}
	if cfg.Speed != 2.5 {
		t.Errorf("Speed = %f, want 2.5", cfg.Speed)
	}
	if cfg.Interval != 500*time.Millisecond {
		t.Errorf("Interval = %v, want 500ms", cfg.Interval)
	}
	if !cfg.Loop {
		t.Error("Loop = false, want true")
	}
	if !cfg.Quiet {
		t.Error("Quiet = false, want true")
	}
}

// ---------------------------------------------------------------------------
// Env vars only (no flags)
// ---------------------------------------------------------------------------

func TestParseConfig_EnvOnly(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_SERVER":      "https://env.example.com",
		"VINCENTY_TOKEN":       "sat_envtoken",
		"VINCENTY_FILE":        "env-track.geojson",
		"VINCENTY_DEVICE_NAME": "env-device",
		"VINCENTY_SPEED":       "3.0",
		"VINCENTY_INTERVAL":    "2s",
		"VINCENTY_LOOP":        "true",
		"VINCENTY_QUIET":       "1",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "https://env.example.com" {
		t.Errorf("Server = %q, want %q", cfg.Server, "https://env.example.com")
	}
	if cfg.Token != "sat_envtoken" {
		t.Errorf("Token = %q, want %q", cfg.Token, "sat_envtoken")
	}
	if cfg.File != "env-track.geojson" {
		t.Errorf("File = %q, want %q", cfg.File, "env-track.geojson")
	}
	if cfg.DeviceName != "env-device" {
		t.Errorf("DeviceName = %q, want %q", cfg.DeviceName, "env-device")
	}
	if cfg.Speed != 3.0 {
		t.Errorf("Speed = %f, want 3.0", cfg.Speed)
	}
	if cfg.Interval != 2*time.Second {
		t.Errorf("Interval = %v, want 2s", cfg.Interval)
	}
	if !cfg.Loop {
		t.Error("Loop = false, want true")
	}
	if !cfg.Quiet {
		t.Error("Quiet = false, want true")
	}
}

// ---------------------------------------------------------------------------
// Flags take precedence over env vars
// ---------------------------------------------------------------------------

func TestParseConfig_FlagOverridesEnv(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_SERVER": "https://env.example.com",
		"VINCENTY_TOKEN":  "sat_envtoken",
		"VINCENTY_FILE":   "env-track.gpx",
	})
	args := []string{
		"-server", "https://flag.example.com",
		"-token", "sat_flagtoken",
		"-file", "flag-track.gpx",
	}
	cfg, err := parseConfig(args, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "https://flag.example.com" {
		t.Errorf("Server = %q, want flag value %q", cfg.Server, "https://flag.example.com")
	}
	if cfg.Token != "sat_flagtoken" {
		t.Errorf("Token = %q, want flag value %q", cfg.Token, "sat_flagtoken")
	}
	if cfg.File != "flag-track.gpx" {
		t.Errorf("File = %q, want flag value %q", cfg.File, "flag-track.gpx")
	}
}

// ---------------------------------------------------------------------------
// Defaults when neither flag nor env is set
// ---------------------------------------------------------------------------

func TestParseConfig_Defaults(t *testing.T) {
	args := []string{"-token", "sat_tok", "-file", "t.gpx"}
	cfg, err := parseConfig(args, noEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "http://localhost:8080" {
		t.Errorf("Server = %q, want default %q", cfg.Server, "http://localhost:8080")
	}
	if cfg.Speed != 1.0 {
		t.Errorf("Speed = %f, want default 1.0", cfg.Speed)
	}
	if cfg.Interval != time.Second {
		t.Errorf("Interval = %v, want default 1s", cfg.Interval)
	}
	if cfg.Loop {
		t.Error("Loop should default to false")
	}
	if cfg.Quiet {
		t.Error("Quiet should default to false")
	}
	// DeviceName should be auto-generated.
	if !strings.HasPrefix(cfg.DeviceName, "cli-") {
		t.Errorf("DeviceName = %q, want cli-xxxx prefix", cfg.DeviceName)
	}
}

// ---------------------------------------------------------------------------
// Validation: missing required fields
// ---------------------------------------------------------------------------

func TestParseConfig_NoAuth(t *testing.T) {
	// Neither token nor username/password — should error mentioning both options.
	args := []string{"-file", "track.gpx"}
	_, err := parseConfig(args, noEnv())
	if err == nil {
		t.Fatal("expected error when no auth is provided")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "token") &&
		!strings.Contains(strings.ToLower(err.Error()), "username") {
		t.Errorf("error %q should mention token or username", err.Error())
	}
}

func TestParseConfig_UsernamePassword(t *testing.T) {
	// username + password without token is valid.
	args := []string{"-username", "admin", "-password", "secret", "-file", "track.gpx"}
	cfg, err := parseConfig(args, noEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Username != "admin" {
		t.Errorf("Username = %q, want %q", cfg.Username, "admin")
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want %q", cfg.Password, "secret")
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
}

func TestParseConfig_UsernamePasswordFromEnv(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_USERNAME": "admin",
		"VINCENTY_PASSWORD": "secret",
		"VINCENTY_FILE":     "track.gpx",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Username != "admin" {
		t.Errorf("Username = %q, want %q", cfg.Username, "admin")
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want %q", cfg.Password, "secret")
	}
}

func TestParseConfig_UsernameWithoutPassword(t *testing.T) {
	args := []string{"-username", "admin", "-file", "track.gpx"}
	_, err := parseConfig(args, noEnv())
	if err == nil {
		t.Fatal("expected error for username without password")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "password") {
		t.Errorf("error %q should mention password", err.Error())
	}
}

func TestParseConfig_PasswordWithoutUsername(t *testing.T) {
	args := []string{"-password", "secret", "-file", "track.gpx"}
	_, err := parseConfig(args, noEnv())
	if err == nil {
		t.Fatal("expected error for password without username")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "username") {
		t.Errorf("error %q should mention username", err.Error())
	}
}

func TestParseConfig_TokenTakesPrecedence(t *testing.T) {
	// Token wins when all three are set; username/password are stored but token
	// being non-empty means Login() won't be called (enforced in main.go).
	args := []string{
		"-token", "sat_abc123",
		"-username", "admin",
		"-password", "secret",
		"-file", "track.gpx",
	}
	cfg, err := parseConfig(args, noEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "sat_abc123" {
		t.Errorf("Token = %q, want sat_abc123", cfg.Token)
	}
}

func TestParseConfig_MissingFile(t *testing.T) {
	args := []string{"-token", "sat_abc"}
	_, err := parseConfig(args, noEnv())
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "file") && !strings.Contains(err.Error(), "FILE") {
		t.Errorf("error %q should mention file", err.Error())
	}
}

func TestParseConfig_InvalidSpeed(t *testing.T) {
	args := []string{"-token", "sat_abc", "-file", "t.gpx", "-speed", "0"}
	_, err := parseConfig(args, noEnv())
	if err == nil {
		t.Fatal("expected error for zero speed")
	}
	if !strings.Contains(err.Error(), "speed") {
		t.Errorf("error %q should mention speed", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Validation: invalid env var values
// ---------------------------------------------------------------------------

func TestParseConfig_InvalidSpeedEnv(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_TOKEN": "sat_tok",
		"VINCENTY_FILE":  "t.gpx",
		"VINCENTY_SPEED": "notanumber",
	})
	_, err := parseConfig(nil, env)
	if err == nil {
		t.Fatal("expected error for invalid VINCENTY_SPEED")
	}
	if !strings.Contains(err.Error(), "VINCENTY_SPEED") {
		t.Errorf("error %q should mention VINCENTY_SPEED", err.Error())
	}
}

func TestParseConfig_InvalidIntervalEnv(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_TOKEN":    "sat_tok",
		"VINCENTY_FILE":     "t.gpx",
		"VINCENTY_INTERVAL": "badvalue",
	})
	_, err := parseConfig(nil, env)
	if err == nil {
		t.Fatal("expected error for invalid VINCENTY_INTERVAL")
	}
	if !strings.Contains(err.Error(), "VINCENTY_INTERVAL") {
		t.Errorf("error %q should mention VINCENTY_INTERVAL", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Boolean env var parsing
// ---------------------------------------------------------------------------

func TestParseConfig_LoopEnvTrue(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_TOKEN": "sat_tok",
		"VINCENTY_FILE":  "t.gpx",
		"VINCENTY_LOOP":  "true",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Loop {
		t.Error("Loop = false, want true for VINCENTY_LOOP=true")
	}
}

func TestParseConfig_QuietEnvOne(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_TOKEN": "sat_tok",
		"VINCENTY_FILE":  "t.gpx",
		"VINCENTY_QUIET": "1",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Quiet {
		t.Error("Quiet = false, want true for VINCENTY_QUIET=1")
	}
}

func TestParseConfig_BoolEnvFalseValues(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_TOKEN": "sat_tok",
		"VINCENTY_FILE":  "t.gpx",
		"VINCENTY_LOOP":  "false",
		"VINCENTY_QUIET": "0",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Loop {
		t.Error("Loop = true, want false for VINCENTY_LOOP=false")
	}
	if cfg.Quiet {
		t.Error("Quiet = true, want false for VINCENTY_QUIET=0")
	}
}

// ---------------------------------------------------------------------------
// DeviceName from env
// ---------------------------------------------------------------------------

func TestParseConfig_DeviceNameFromEnv(t *testing.T) {
	env := mockEnv(map[string]string{
		"VINCENTY_TOKEN":       "sat_tok",
		"VINCENTY_FILE":        "t.gpx",
		"VINCENTY_DEVICE_NAME": "env-device",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DeviceName != "env-device" {
		t.Errorf("DeviceName = %q, want %q", cfg.DeviceName, "env-device")
	}
}

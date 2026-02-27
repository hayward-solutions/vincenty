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
		"SITAWARE_SERVER":      "https://env.example.com",
		"SITAWARE_TOKEN":       "sat_envtoken",
		"SITAWARE_FILE":        "env-track.geojson",
		"SITAWARE_DEVICE_NAME": "env-device",
		"SITAWARE_SPEED":       "3.0",
		"SITAWARE_INTERVAL":    "2s",
		"SITAWARE_LOOP":        "true",
		"SITAWARE_QUIET":       "1",
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
		"SITAWARE_SERVER": "https://env.example.com",
		"SITAWARE_TOKEN":  "sat_envtoken",
		"SITAWARE_FILE":   "env-track.gpx",
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

func TestParseConfig_MissingToken(t *testing.T) {
	args := []string{"-file", "track.gpx"}
	_, err := parseConfig(args, noEnv())
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "token") && !strings.Contains(err.Error(), "TOKEN") {
		t.Errorf("error %q should mention token", err.Error())
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
		"SITAWARE_TOKEN": "sat_tok",
		"SITAWARE_FILE":  "t.gpx",
		"SITAWARE_SPEED": "notanumber",
	})
	_, err := parseConfig(nil, env)
	if err == nil {
		t.Fatal("expected error for invalid SITAWARE_SPEED")
	}
	if !strings.Contains(err.Error(), "SITAWARE_SPEED") {
		t.Errorf("error %q should mention SITAWARE_SPEED", err.Error())
	}
}

func TestParseConfig_InvalidIntervalEnv(t *testing.T) {
	env := mockEnv(map[string]string{
		"SITAWARE_TOKEN":    "sat_tok",
		"SITAWARE_FILE":     "t.gpx",
		"SITAWARE_INTERVAL": "badvalue",
	})
	_, err := parseConfig(nil, env)
	if err == nil {
		t.Fatal("expected error for invalid SITAWARE_INTERVAL")
	}
	if !strings.Contains(err.Error(), "SITAWARE_INTERVAL") {
		t.Errorf("error %q should mention SITAWARE_INTERVAL", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Boolean env var parsing
// ---------------------------------------------------------------------------

func TestParseConfig_LoopEnvTrue(t *testing.T) {
	env := mockEnv(map[string]string{
		"SITAWARE_TOKEN": "sat_tok",
		"SITAWARE_FILE":  "t.gpx",
		"SITAWARE_LOOP":  "true",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Loop {
		t.Error("Loop = false, want true for SITAWARE_LOOP=true")
	}
}

func TestParseConfig_QuietEnvOne(t *testing.T) {
	env := mockEnv(map[string]string{
		"SITAWARE_TOKEN": "sat_tok",
		"SITAWARE_FILE":  "t.gpx",
		"SITAWARE_QUIET": "1",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Quiet {
		t.Error("Quiet = false, want true for SITAWARE_QUIET=1")
	}
}

func TestParseConfig_BoolEnvFalseValues(t *testing.T) {
	env := mockEnv(map[string]string{
		"SITAWARE_TOKEN": "sat_tok",
		"SITAWARE_FILE":  "t.gpx",
		"SITAWARE_LOOP":  "false",
		"SITAWARE_QUIET": "0",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Loop {
		t.Error("Loop = true, want false for SITAWARE_LOOP=false")
	}
	if cfg.Quiet {
		t.Error("Quiet = true, want false for SITAWARE_QUIET=0")
	}
}

// ---------------------------------------------------------------------------
// DeviceName from env
// ---------------------------------------------------------------------------

func TestParseConfig_DeviceNameFromEnv(t *testing.T) {
	env := mockEnv(map[string]string{
		"SITAWARE_TOKEN":       "sat_tok",
		"SITAWARE_FILE":        "t.gpx",
		"SITAWARE_DEVICE_NAME": "env-device",
	})
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DeviceName != "env-device" {
		t.Errorf("DeviceName = %q, want %q", cfg.DeviceName, "env-device")
	}
}

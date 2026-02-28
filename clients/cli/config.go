package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

// config holds all resolved CLI configuration.
type config struct {
	Server     string
	Token      string
	Username   string
	Password   string
	File       string
	DeviceName string
	Speed      float64
	Interval   time.Duration
	Loop       bool
	Quiet      bool
}

// parseConfig resolves configuration from command-line flags and environment
// variables. Flags always take precedence over env vars.
//
// getenv is injected (typically os.Getenv) so tests can supply values without
// mutating process environment.
func parseConfig(args []string, getenv func(string) string) (config, error) {
	fs := flag.NewFlagSet("sitaware-cli", flag.ContinueOnError)

	// Compute defaults: env var → hard-coded fallback.
	serverDefault := envOr(getenv, "SITAWARE_SERVER", "http://localhost:8080")
	tokenDefault := getenv("SITAWARE_TOKEN")
	usernameDefault := getenv("SITAWARE_USERNAME")
	passwordDefault := getenv("SITAWARE_PASSWORD")
	fileDefault := getenv("SITAWARE_FILE")
	deviceDefault := getenv("SITAWARE_DEVICE_NAME")

	speedDefault := 1.0
	if v := getenv("SITAWARE_SPEED"); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return config{}, fmt.Errorf("invalid SITAWARE_SPEED %q: %w", v, err)
		}
		speedDefault = parsed
	}

	intervalDefault := time.Second
	if v := getenv("SITAWARE_INTERVAL"); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return config{}, fmt.Errorf("invalid SITAWARE_INTERVAL %q: %w", v, err)
		}
		intervalDefault = parsed
	}

	loopDefault := envBool(getenv, "SITAWARE_LOOP")
	quietDefault := envBool(getenv, "SITAWARE_QUIET")

	// Define flags with env-derived defaults.
	server := fs.String("server", serverDefault, "API base URL (env: SITAWARE_SERVER)")
	token := fs.String("token", tokenDefault, "API token sat_... (env: SITAWARE_TOKEN)")
	username := fs.String("username", usernameDefault, "Username for login (env: SITAWARE_USERNAME)")
	password := fs.String("password", passwordDefault, "Password for login (env: SITAWARE_PASSWORD)")
	file := fs.String("file", fileDefault, "Path to .gpx or .geojson file (env: SITAWARE_FILE)")
	deviceName := fs.String("device-name", deviceDefault, "Device name (env: SITAWARE_DEVICE_NAME)")
	speed := fs.Float64("speed", speedDefault, "Playback speed multiplier (env: SITAWARE_SPEED)")
	interval := fs.Duration("interval", intervalDefault, "Send interval when track has no timestamps (env: SITAWARE_INTERVAL)")
	loop := fs.Bool("loop", loopDefault, "Loop the track continuously (env: SITAWARE_LOOP)")
	quiet := fs.Bool("quiet", quietDefault, "Suppress per-point log output (env: SITAWARE_QUIET)")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	// Validate auth: require either a token or a username+password pair.
	if *token == "" {
		if *username == "" && *password == "" {
			return config{}, fmt.Errorf("authentication required: provide -token / SITAWARE_TOKEN, or -username / SITAWARE_USERNAME with -password / SITAWARE_PASSWORD")
		}
		if *username == "" {
			return config{}, fmt.Errorf("-username or SITAWARE_USERNAME is required when using password authentication")
		}
		if *password == "" {
			return config{}, fmt.Errorf("-password or SITAWARE_PASSWORD is required when using password authentication")
		}
	}
	if *file == "" {
		return config{}, fmt.Errorf("-file or SITAWARE_FILE is required")
	}
	if *speed <= 0 {
		return config{}, fmt.Errorf("-speed must be positive")
	}

	if *deviceName == "" {
		*deviceName = fmt.Sprintf("cli-%04x", rand.IntN(0xFFFF))
	}

	return config{
		Server:     *server,
		Token:      *token,
		Username:   *username,
		Password:   *password,
		File:       *file,
		DeviceName: *deviceName,
		Speed:      *speed,
		Interval:   *interval,
		Loop:       *loop,
		Quiet:      *quiet,
	}, nil
}

// envOr returns the env var value if non-empty, otherwise fallback.
func envOr(getenv func(string) string, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return fallback
}

// envBool returns true if the env var is "true" or "1" (case-insensitive).
func envBool(getenv func(string) string, key string) bool {
	v := strings.ToLower(getenv(key))
	return v == "true" || v == "1"
}

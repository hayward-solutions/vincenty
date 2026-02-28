// Command sitaware-cli streams GPX or GeoJSON track files to the SitAware
// API over WebSocket, simulating a moving device. Useful for demonstrations,
// load testing, and resource rightsizing.
//
// Usage:
//
//	sitaware-cli [flags]
//
// Required (flag or environment variable):
//
//	-token   API token (sat_...)          env: SITAWARE_TOKEN
//	-file    Path to a .gpx or .geojson   env: SITAWARE_FILE
//
// Optional:
//
//	-server        API base URL            env: SITAWARE_SERVER   (default http://localhost:8080)
//	-device-name   Device name             env: SITAWARE_DEVICE_NAME (default cli-<random>)
//	-speed         Playback speed          env: SITAWARE_SPEED    (default 1.0)
//	-interval      Send interval           env: SITAWARE_INTERVAL (default 1s)
//	-loop          Loop the track          env: SITAWARE_LOOP     (default false)
//	-quiet         Suppress per-point log  env: SITAWARE_QUIET    (default false)
//	-version       Print version and exit
//
// Flags take precedence over environment variables.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sitaware/cli/internal/client"
	"github.com/sitaware/cli/internal/track"
)

// version is set at build time via -ldflags "-X main.version=<value>".
// Falls back to "dev" for local builds.
var version = "dev"

func main() {
	// Handle -version / --version before full flag parsing.
	for _, arg := range os.Args[1:] {
		if arg == "-version" || arg == "--version" {
			fmt.Println(version)
			return
		}
	}

	cfg, err := parseConfig(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// Logger
	// -----------------------------------------------------------------------
	logLevel := slog.LevelInfo
	if cfg.Quiet {
		logLevel = slog.LevelWarn
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	// -----------------------------------------------------------------------
	// Parse track file
	// -----------------------------------------------------------------------
	slog.Info("loading track file", "path", cfg.File)
	points, err := track.Load(cfg.File)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	slog.Info("loaded track", "points", len(points), "has_timestamps", points[0].Time != nil)

	// -----------------------------------------------------------------------
	// Context with signal handling
	// -----------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	// -----------------------------------------------------------------------
	// Create API client and device
	// -----------------------------------------------------------------------
	api := client.New(cfg.Server, cfg.Token)

	slog.Info("creating device", "name", cfg.DeviceName, "version", version)
	device, err := api.CreateDevice(ctx, cfg.DeviceName, "cli", version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating device: %v\n", err)
		os.Exit(1)
	}
	slog.Info("device created", "id", device.ID, "name", device.Name)

	// Clean up device on exit
	defer func() {
		slog.Info("cleaning up device", "id", device.ID)
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := api.DeleteDevice(cleanupCtx, device.ID); err != nil {
			slog.Warn("failed to delete device", "error", err)
		} else {
			slog.Info("device deleted")
		}
	}()

	// -----------------------------------------------------------------------
	// Connect WebSocket
	// -----------------------------------------------------------------------
	ws, err := api.ConnectWS(ctx, device.ID, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting websocket: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()
	slog.Info("websocket connected")

	// Drain incoming messages in the background
	go ws.DrainMessages(ctx)

	// -----------------------------------------------------------------------
	// Stream track points
	// -----------------------------------------------------------------------
	iteration := 0
	for {
		iteration++
		if cfg.Loop {
			slog.Info("starting track iteration", "iteration", iteration)
		}

		if err := streamTrack(ctx, ws, device.ID, points, cfg.Speed, cfg.Interval, cfg.Quiet); err != nil {
			if ctx.Err() != nil {
				slog.Info("streaming stopped")
				return
			}
			fmt.Fprintf(os.Stderr, "error streaming: %v\n", err)
			os.Exit(1)
		}

		if !cfg.Loop {
			slog.Info("track complete", "points_sent", len(points))
			return
		}
	}
}

func streamTrack(ctx context.Context, ws *client.WSConn, deviceID string, points []track.Point, speed float64, fallbackInterval time.Duration, quiet bool) error {
	for i, pt := range points {
		// Check for cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Calculate heading and speed from consecutive points
		var heading *float64
		var spd *float64
		if i+1 < len(points) {
			heading = track.Heading(pt, points[i+1])
			spd = track.Speed(pt, points[i+1])
		}

		msg := client.LocationUpdate{
			DeviceID: deviceID,
			Lat:      pt.Lat,
			Lng:      pt.Lng,
			Altitude: pt.Altitude,
			Heading:  heading,
			Speed:    spd,
		}

		if err := ws.SendLocationUpdate(ctx, msg); err != nil {
			return fmt.Errorf("send point %d: %w", i, err)
		}

		if !quiet {
			slog.Info("sent",
				"point", fmt.Sprintf("%d/%d", i+1, len(points)),
				"lat", pt.Lat,
				"lng", pt.Lng,
			)
		}

		// Sleep until next point
		if i+1 < len(points) {
			delay := fallbackInterval
			if pt.Time != nil && points[i+1].Time != nil {
				dt := points[i+1].Time.Sub(*pt.Time)
				if dt > 0 {
					delay = time.Duration(float64(dt) / speed)
				}
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return nil
}

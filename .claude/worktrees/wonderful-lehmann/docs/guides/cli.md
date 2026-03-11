# CLI Client

The Vincenty CLI streams GPX or GeoJSON track files to the API over WebSocket, simulating a moving device. Useful for demonstrations, load testing, and connection capacity planning.

Each CLI invocation creates a temporary device, streams the track, and cleans up the device on exit. Run multiple instances in parallel to simulate concurrent users.

## Installation

### Docker (recommended)

No installation needed. The image is published to GHCR on every merge to `main`:

```bash
docker run --rm ghcr.io/<org>/vincenty/cli --help
```

### Build from source

```bash
# From the repository root
make cli-build
./clients/cli/bin/vincenty-cli --help
```

### Go install

```bash
go install github.com/vincenty/cli@latest
vincenty-cli --help
```

## Creating an API Token

The CLI authenticates with a long-lived API token instead of a JWT. Any user can create tokens for themselves.

### Via curl

```bash
# Login to get a JWT
TOKEN=$(curl -s http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"changeme"}' | jq -r '.access_token')

# Create an API token
curl -s http://localhost:8080/api/v1/users/me/api-tokens \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"my-cli-token"}' | jq .
```

The response includes a `token` field (e.g. `sat_a1b2c3...`). Save this value -- it is only shown once.

### Optional expiry

```bash
curl -s http://localhost:8080/api/v1/users/me/api-tokens \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"temp-token","expires_at":"2026-12-31T23:59:59Z"}' | jq .
```

### Managing tokens

```bash
# List your tokens
curl -s http://localhost:8080/api/v1/users/me/api-tokens \
  -H "Authorization: Bearer $TOKEN" | jq .

# Delete a token
curl -s -X DELETE http://localhost:8080/api/v1/users/me/api-tokens/<token-id> \
  -H "Authorization: Bearer $TOKEN"
```

## Configuration

All parameters can be set via flags or environment variables. Flags take precedence.

| Flag | Env var | Required | Default | Description |
|---|---|---|---|---|
| `-token` | `VINCENTY_TOKEN` | Yes | | API token (`sat_...`) |
| `-file` | `VINCENTY_FILE` | Yes | | Path to `.gpx` or `.geojson` file |
| `-server` | `VINCENTY_SERVER` | No | `http://localhost:8080` | API base URL |
| `-device-name` | `VINCENTY_DEVICE_NAME` | No | `cli-<random>` | Device name |
| `-speed` | `VINCENTY_SPEED` | No | `1.0` | Playback speed multiplier |
| `-interval` | `VINCENTY_INTERVAL` | No | `1s` | Send interval (when track has no timestamps) |
| `-loop` | `VINCENTY_LOOP` | No | `false` | Loop the track continuously |
| `-quiet` | `VINCENTY_QUIET` | No | `false` | Suppress per-point log output |

## Usage

### Local binary

```bash
vincenty-cli \
  --server=https://api.example.com \
  --token=sat_a1b2c3... \
  --file=path/to/track.gpx
```

### Docker

Mount the track file into the container and pass config via environment variables:

```bash
docker run --rm \
  -v /path/to/tracks:/data:ro \
  -e VINCENTY_SERVER=https://api.example.com \
  -e VINCENTY_TOKEN=sat_a1b2c3... \
  -e VINCENTY_FILE=/data/track.gpx \
  ghcr.io/<org>/vincenty/cli
```

Or use flags directly:

```bash
docker run --rm \
  -v /path/to/tracks:/data:ro \
  ghcr.io/<org>/vincenty/cli \
  --server=https://api.example.com \
  --token=sat_a1b2c3... \
  --file=/data/track.gpx
```

## Track File Formats

### GPX

Standard GPX files with `<trk>/<trkseg>/<trkpt>` or `<rte>/<rtept>` elements. If track points include `<time>` elements, playback is time-paced (respecting the original recording intervals). Without timestamps, points are sent at the `-interval` rate.

### GeoJSON

Supported geometry types:

- **LineString** -- interpreted as an ordered track
- **MultiLineString** -- segments concatenated in order
- **Point** -- single location update

GeoJSON can be a bare geometry, a Feature, or a FeatureCollection. Coordinates are `[longitude, latitude]` or `[longitude, latitude, altitude]`.

## Examples

### Fast playback

Stream a track at 10x speed:

```bash
vincenty-cli --token=sat_... --file=track.gpx --speed=10
```

### Continuous loop

Loop a track indefinitely (useful for demos):

```bash
vincenty-cli --token=sat_... --file=track.gpx --loop
```

### Multiple concurrent devices

Open several terminals (or use `&` for background jobs):

```bash
vincenty-cli --token=sat_... --file=route-a.gpx --device-name=vehicle-1 &
vincenty-cli --token=sat_... --file=route-b.gpx --device-name=vehicle-2 &
vincenty-cli --token=sat_... --file=route-c.gpx --device-name=vehicle-3 &
```

### Quiet mode with Docker Compose environment

```bash
docker run --rm \
  -v ./tracks:/data:ro \
  --env-file=cli.env \
  ghcr.io/<org>/vincenty/cli
```

Where `cli.env` contains:

```
VINCENTY_SERVER=https://api.example.com
VINCENTY_TOKEN=sat_a1b2c3...
VINCENTY_FILE=/data/track.gpx
VINCENTY_LOOP=true
VINCENTY_QUIET=true
```

# SitAware

A modern, lightweight situational awareness platform. Built as an alternative to TAK Server for teams that need real-time location tracking, secure messaging, and map-based coordination — deployable to the cloud or fully air-gapped environments with zero internet dependency.

## Why SitAware?

TAK Server is powerful but heavy — complex to deploy, tightly coupled to specific clients, and difficult to run in constrained environments. SitAware takes a different approach:

- **Lightweight** — Go API with minimal dependencies, distroless container images
- **Air-gap ready** — No CDN calls, no external fonts, local tile serving. Works on an isolated network with `docker compose up`
- **Modern clients** — Browser-based web UI and native iOS app with real-time maps, chat, and admin tools
- **Cloud native** — Runs on Docker Compose, Kubernetes, or AWS ECS Fargate
- **Simple operations** — All configuration via environment variables, automatic database migrations, admin bootstrap on first start

## Quick Start

Prerequisites: [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)

```bash
git clone https://github.com/sitaware/sitaware.git
cd sitaware
make dev
```

This starts the full stack:

| Service | URL | Purpose |
|---|---|---|
| Web Client | http://localhost:3000 | Browser UI |
| API | http://localhost:8080 | REST + WebSocket API |
| PostgreSQL | localhost:5432 | Database (PostGIS) |
| Redis | localhost:6379 | Pub/sub messaging |
| Minio Console | http://localhost:9001 | Object storage (dev only) |

Default admin credentials: `admin` / `changeme`

For the iOS client, see [Contributing — iOS Client Setup](CONTRIBUTING.md#ios-client-setup).

## Tech Stack

| Component | Technology |
|---|---|
| API | Go (stdlib `net/http`), no framework |
| Database | PostgreSQL 16 + PostGIS |
| Pub/Sub | Redis (pluggable — interface supports Kafka, Apache Ignite) |
| Object Storage | S3-compatible (Minio for dev, AWS S3 for production) |
| Real-time | WebSocket ([nhooyr.io/websocket](https://github.com/nhooyr/websocket)) |
| Auth | JWT access tokens + rotating opaque refresh tokens |
| Web Client | Next.js (App Router, standalone output) |
| iOS Client | SwiftUI (iOS 17+, Swift 6.0, MVVM + Observation) |
| UI (Web) | shadcn/ui, Tailwind CSS v4, Radix UI |
| Maps | MapLibre GL JS (web), MapLibre Native SDK (iOS) |
| Containers | Multi-stage Docker (distroless for API, node-slim for web) |

## Repository Structure

```
sitaware/
├── services/api/          # Go API service
│   ├── cmd/server/        # Entrypoint, DI, route registration
│   ├── internal/
│   │   ├── auth/          # JWT + password hashing
│   │   ├── config/        # Env-based configuration
│   │   ├── database/      # Migration runner + embedded SQL
│   │   ├── handler/       # HTTP handlers (one per domain)
│   │   ├── middleware/     # CORS, auth, logging, rate limit, audit
│   │   ├── model/         # Domain models + typed errors
│   │   ├── pubsub/        # Pub/sub interface + Redis impl
│   │   ├── repository/    # SQL queries (pgx)
│   │   ├── service/       # Business logic
│   │   ├── storage/       # Object storage interface + S3 impl
│   │   └── ws/            # WebSocket hub + client management
│   └── Dockerfile
├── clients/web/           # Next.js web client
│   ├── src/
│   │   ├── app/           # App Router pages
│   │   ├── components/    # UI, map, chat, audit components
│   │   ├── lib/           # API client, auth context, hooks
│   │   └── types/         # TypeScript definitions
│   └── Dockerfile
├── clients/ios/           # iOS client (SwiftUI, XcodeGen)
│   ├── SitAware/
│   │   ├── App/           # Entry point, root views
│   │   ├── Models/        # Codable API models
│   │   ├── Core/          # Services (API, auth, WebSocket, sync)
│   │   ├── Features/      # Feature modules (map, messages, settings, etc.)
│   │   └── Components/    # Shared UI components
│   └── project.yml        # XcodeGen spec (generates .xcodeproj)
├── deploy/
│   ├── caddy/             # Reverse proxy config (production)
│   ├── k8s/               # Kubernetes manifests
│   ├── helm/sitaware/     # Helm chart
│   └── ecs/               # AWS ECS Fargate task definitions
├── docker-compose.yml     # Development stack
├── docker-compose.prod.yml # Production stack (Caddy + TLS)
└── Makefile
```

## Make Targets

```bash
make dev          # Start full dev stack (Docker Compose)
make down         # Stop all services
make logs         # Tail all service logs
make infra        # Start only postgres, redis, minio
make restart s=api  # Rebuild and restart a single service
make api-dev      # Run Go API locally (requires infra)
make web-dev      # Run Next.js dev server locally
make db-shell     # Open psql shell
make api-build    # Build Go API binary
make prod         # Start production stack (Caddy + TLS)
make prod-down    # Stop production stack
make prod-logs    # Tail production logs
make clean        # Remove all containers and volumes
```

## API Endpoints

All API routes are prefixed with `/api/v1/`.

### Public

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness check |
| GET | `/readyz` | Readiness check (DB ping) |
| POST | `/api/v1/auth/login` | Login (returns JWT + refresh token) |
| POST | `/api/v1/auth/refresh` | Rotate tokens |
| POST | `/api/v1/auth/mfa/totp` | Verify TOTP code (MFA challenge) |
| POST | `/api/v1/auth/mfa/webauthn/begin` | Begin WebAuthn assertion (MFA challenge) |
| POST | `/api/v1/auth/mfa/webauthn/finish` | Finish WebAuthn assertion (MFA challenge) |
| POST | `/api/v1/auth/mfa/recovery` | Verify recovery code (MFA challenge) |
| POST | `/api/v1/auth/passkey/begin` | Begin passkey login (passwordless) |
| POST | `/api/v1/auth/passkey/finish` | Finish passkey login (passwordless) |

### Authenticated

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/auth/logout` | Revoke refresh token |
| GET | `/api/v1/users/me` | Current user profile |
| PUT | `/api/v1/users/me` | Update own profile |
| GET | `/api/v1/users/me/devices` | List own devices |
| POST | `/api/v1/users/me/devices` | Register a device |
| PUT | `/api/v1/devices/{id}` | Update a device |
| DELETE | `/api/v1/devices/{id}` | Delete a device |
| GET | `/api/v1/users/me/groups` | List groups I belong to |
| GET | `/api/v1/groups/{id}/members` | List group members |
| POST | `/api/v1/groups/{id}/members` | Add member to group |
| PUT | `/api/v1/groups/{id}/members/{userId}` | Update member permissions |
| DELETE | `/api/v1/groups/{id}/members/{userId}` | Remove member |
| GET | `/api/v1/groups/{id}/locations/history` | Group location history |
| GET | `/api/v1/map/settings` | Map configuration |
| POST | `/api/v1/messages` | Send message (group or direct) |
| GET | `/api/v1/groups/{id}/messages` | Group message history |
| GET | `/api/v1/messages/conversations` | List DM conversations |
| GET | `/api/v1/messages/direct/{userId}` | Direct message history |
| GET | `/api/v1/messages/{id}` | Get single message |
| DELETE | `/api/v1/messages/{id}` | Delete a message |
| GET | `/api/v1/attachments/{id}/download` | Download file attachment |
| GET | `/api/v1/audit-logs/me` | Own audit logs |
| GET | `/api/v1/audit-logs/me/export` | Export own audit logs |
| GET | `/api/v1/groups/{id}/audit-logs` | Group audit logs |
| GET | `/api/v1/users/me/locations/history` | Own location history |
| GET | `/api/v1/users/me/locations/export` | Export location history (GPX) |
| GET | `/api/v1/ws` | WebSocket connection |
| GET | `/api/v1/mfa/status` | Get own MFA status and methods |
| POST | `/api/v1/mfa/totp/setup` | Begin TOTP setup (returns QR code) |
| POST | `/api/v1/mfa/totp/verify` | Verify TOTP code to activate |
| DELETE | `/api/v1/mfa/totp` | Remove TOTP method |
| POST | `/api/v1/mfa/webauthn/register/begin` | Begin WebAuthn credential registration |
| POST | `/api/v1/mfa/webauthn/register/finish` | Finish WebAuthn credential registration |
| DELETE | `/api/v1/mfa/webauthn/{id}` | Remove a WebAuthn credential |
| PATCH | `/api/v1/mfa/webauthn/{id}` | Update credential (name, passwordless flag) |
| POST | `/api/v1/mfa/recovery/regenerate` | Regenerate recovery codes |

### Admin Only

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/users` | List all users |
| POST | `/api/v1/users` | Create user |
| GET | `/api/v1/users/{id}` | Get user |
| PUT | `/api/v1/users/{id}` | Update user |
| DELETE | `/api/v1/users/{id}` | Delete user |
| GET | `/api/v1/groups` | List all groups |
| POST | `/api/v1/groups` | Create group |
| GET | `/api/v1/groups/{id}` | Get group |
| PUT | `/api/v1/groups/{id}` | Update group |
| DELETE | `/api/v1/groups/{id}` | Delete group |
| GET | `/api/v1/locations` | All latest locations |
| GET | `/api/v1/map-configs` | List map configurations |
| POST | `/api/v1/map-configs` | Create map configuration |
| GET | `/api/v1/map-configs/{id}` | Get map configuration |
| PUT | `/api/v1/map-configs/{id}` | Update map configuration |
| DELETE | `/api/v1/map-configs/{id}` | Delete map configuration |
| GET | `/api/v1/audit-logs` | All audit logs |
| GET | `/api/v1/audit-logs/export` | Export all audit logs |
| DELETE | `/api/v1/users/{id}/mfa` | Reset a user's MFA |
| GET | `/api/v1/server-settings` | Get server settings |
| PUT | `/api/v1/server-settings` | Update server settings (e.g. `mfa_required`) |

## Configuration

All configuration is via environment variables. Copy `.env.example` to `.env` and edit:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|---|---|---|
| `ADMIN_USERNAME` | `admin` | Bootstrap admin username |
| `ADMIN_PASSWORD` | `changeme` | Bootstrap admin password |
| `ADMIN_EMAIL` | `admin@sitaware.local` | Bootstrap admin email |
| `API_HOST` | `0.0.0.0` | API listen address |
| `API_PORT` | `8080` | API listen port |
| `API_LOG_LEVEL` | `debug` | Log level (debug, info, warn, error) |
| `JWT_SECRET` | (insecure default) | HMAC-SHA256 signing key |
| `JWT_ACCESS_TOKEN_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TOKEN_TTL` | `168h` | Refresh token lifetime (7 days) |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `sitaware` | PostgreSQL user |
| `DB_PASSWORD` | `sitaware` | PostgreSQL password |
| `DB_NAME` | `sitaware` | PostgreSQL database |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_TLS` | `false` | Enable TLS for Redis (required for ElastiCache with transit encryption) |
| `S3_ENDPOINT` | `http://localhost:9000` | S3/Minio endpoint |
| `S3_ACCESS_KEY` | `sitaware` | S3 access key |
| `S3_SECRET_KEY` | `sitaware123` | S3 secret key |
| `S3_BUCKET` | `sitaware` | S3 bucket name |
| `S3_REGION` | `us-east-1` | S3 region |
| `S3_USE_PATH_STYLE` | `true` | Path-style S3 (true for Minio) |
| `WS_LOCATION_THROTTLE` | `1s` | Min interval between location updates |
| `WS_URL` | `ws://localhost:8080` | WebSocket URL (browser-facing, read at runtime by server) |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated allowed origins |
| `RATE_LIMIT_RPS` | `10` | Requests per second per IP |
| `RATE_LIMIT_BURST` | `20` | Rate limit burst size |
| `MAX_REQUEST_BODY_BYTES` | `10485760` | Max request body (10MB) |
| `TOKEN_CLEANUP_INTERVAL` | `1h` | Expired token purge interval |
| `WEBAUTHN_RP_ID` | `localhost` | WebAuthn Relying Party ID (your domain, no port) |
| `WEBAUTHN_RP_DISPLAY_NAME` | `SitAware` | Display name shown in browser credential prompts |
| `WEBAUTHN_RP_ORIGINS` | `http://localhost:3000` | Comma-separated allowed WebAuthn origins |
| `MFA_KMS_KEY_ARN` | (empty) | AWS KMS key ARN for TOTP secret encryption. When empty, uses AES-256-GCM derived from `JWT_SECRET` via HKDF |
| `MAP_DEFAULT_TILE_URL` | OSM tiles | Default map tile URL template |
| `MAP_DEFAULT_CENTER_LAT` | `0` | Default map center latitude |
| `MAP_DEFAULT_CENTER_LNG` | `0` | Default map center longitude |
| `MAP_DEFAULT_ZOOM` | `2` | Default map zoom level |

## Documentation

### Guides

| Guide | Description |
|---|---|
| [Getting Started](docs/guides/getting-started.md) | First login, navigation overview, key concepts |
| [Using the Map](docs/guides/map.md) | Location markers, drawing tools, replay, measurement, terrain |
| [Messaging](docs/guides/messaging.md) | Group chat, direct messages, file attachments |
| [Location Sharing](docs/guides/location-sharing.md) | Enabling location sharing, privacy, GPX export |
| [MFA Setup](docs/guides/mfa-setup.md) | TOTP, WebAuthn/passkeys, recovery codes |
| [Account Settings](docs/guides/account-settings.md) | Profile, avatar, devices, map marker customization |
| [Admin Guide](docs/guides/admin-guide.md) | User/group management, map config, security, audit logs |

### Reference

| Document | Description |
|---|---|
| [Architecture](ARCHITECTURE.md) | System design, data flow, database schema, deployment targets |
| [Features](FEATURES.md) | Complete feature reference |
| [Contributing](CONTRIBUTING.md) | Development setup, code conventions, PR process |
| [ECS Deployment](deploy/ecs/README.md) | Step-by-step AWS ECS Fargate deployment guide |

## Deployment

SitAware supports multiple deployment targets. See the [Architecture Guide](ARCHITECTURE.md) for details.

### Docker Compose (Production)

Uses Caddy for TLS termination. Place your TLS certificate and key in `deploy/caddy/certs/`, configure the Caddyfile, then:

```bash
make prod
```

### Kubernetes

Raw manifests in `deploy/k8s/`:

```bash
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/
```

### Helm

```bash
helm install sitaware deploy/helm/sitaware/ \
  --namespace sitaware \
  --create-namespace \
  -f my-values.yaml
```

### AWS ECS Fargate

Task definitions and service configs in `deploy/ecs/`. See [`deploy/ecs/README.md`](deploy/ecs/README.md) for the full walkthrough.

## Air-Gapped Deployment

SitAware is designed to run with zero internet access:

1. Pre-pull and export all container images
2. Upload map tiles to the S3-compatible object store (Minio in Docker Compose, or any S3 endpoint)
3. Configure `MAP_DEFAULT_TILE_URL` to point to the local tile source
4. All UI assets are bundled — no CDN, no external fonts, no external scripts
5. Deploy with `docker compose up`

## Security

- JWT access tokens (15min, HMAC-SHA256) with rotating opaque refresh tokens
- Refresh tokens stored as SHA-256 hashes in the database
- bcrypt password hashing (cost 12)
- **Multi-Factor Authentication (MFA)**: TOTP (authenticator apps) and WebAuthn/FIDO2 (security keys, passkeys)
- **Passkey login**: Passwordless authentication via WebAuthn discoverable credentials
- **Recovery codes**: 8 one-time backup codes (bcrypt-hashed) generated when MFA is first enabled
- **Admin MFA enforcement**: Server-wide `mfa_required` setting blocks users without MFA from all routes except MFA setup
- **TOTP secret encryption**: AES-256-GCM with HKDF-derived key from `JWT_SECRET` (default), or AWS KMS when `MFA_KMS_KEY_ARN` is set
- Per-IP rate limiting with token bucket algorithm
- Configurable CORS with origin validation
- Request body size limits
- Automatic expired token cleanup
- Distroless container images (API) with non-root execution
- All secrets configurable via environment variables (SSM Parameter Store for ECS)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code conventions, and the pull request process.

## License

[MIT](LICENSE)

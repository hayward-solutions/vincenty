# Configuration

All configuration is via environment variables. Copy `.env.example` to `.env` and edit:

```bash
cp .env.example .env
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ADMIN_USERNAME` | `admin` | Bootstrap admin username |
| `ADMIN_PASSWORD` | `changeme` | Bootstrap admin password |
| `ADMIN_EMAIL` | `admin@vincenty.local` | Bootstrap admin email |
| `API_HOST` | `0.0.0.0` | API listen address |
| `API_PORT` | `8080` | API listen port |
| `API_LOG_LEVEL` | `debug` | Log level (debug, info, warn, error) |
| `JWT_SECRET` | (insecure default) | HMAC-SHA256 signing key |
| `JWT_ACCESS_TOKEN_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TOKEN_TTL` | `168h` | Refresh token lifetime (7 days) |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `vincenty` | PostgreSQL user |
| `DB_PASSWORD` | `vincenty` | PostgreSQL password |
| `DB_NAME` | `vincenty` | PostgreSQL database |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_TLS` | `false` | Enable TLS for Redis (required for ElastiCache with transit encryption) |
| `REDIS_CLUSTER` | `false` | Enable Redis Cluster mode (required for ElastiCache with cluster mode enabled) |
| `S3_ENDPOINT` | `http://localhost:9000` | S3/Minio endpoint |
| `S3_ACCESS_KEY` | `vincenty` | S3 access key |
| `S3_SECRET_KEY` | `vincenty123` | S3 secret key |
| `S3_BUCKET` | `vincenty` | S3 bucket name |
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
| `WEBAUTHN_RP_DISPLAY_NAME` | `Vincenty` | Display name shown in browser credential prompts |
| `WEBAUTHN_RP_ORIGINS` | `http://localhost:3000` | Comma-separated allowed WebAuthn origins |
| `MFA_KMS_KEY_ARN` | (empty) | AWS KMS key ARN for TOTP secret encryption. When empty, uses AES-256-GCM derived from `JWT_SECRET` via HKDF |
| `MAP_DEFAULT_TILE_URL` | OSM tiles | Default map tile URL template |
| `MAP_DEFAULT_CENTER_LAT` | `0` | Default map center latitude |
| `MAP_DEFAULT_CENTER_LNG` | `0` | Default map center longitude |
| `MAP_DEFAULT_ZOOM` | `2` | Default map zoom level |

## Security Notes

!!! warning "Production Configuration"
    Always change the following before deploying to production:

    - `JWT_SECRET` — use a strong random string (at least 32 characters)
    - `ADMIN_PASSWORD` — change from the default `changeme`
    - `DB_PASSWORD` — use a strong database password
    - `S3_ACCESS_KEY` / `S3_SECRET_KEY` — use proper credentials
    - `CORS_ALLOWED_ORIGINS` — restrict to your actual domain(s)
    - `WEBAUTHN_RP_ID` / `WEBAUTHN_RP_ORIGINS` — set to your production domain

## Adding New Variables

When adding a new environment variable:

1. Add it to `internal/config/config.go` (struct field + parsing in `Load()`)
2. Add it to `.env.example` with a sensible default and comment
3. Document it in this page
4. If it's a secret, add it to the Kubernetes secret template, Helm values, and ECS task definition secrets

.PHONY: dev down logs api-dev web-dev cli-build cli-dev db-shell clean build prod prod-down prod-logs

# ---------------------------------------------------------------------------
# Full stack (Docker Compose)
# ---------------------------------------------------------------------------

## Start all services (rebuild if needed)
dev:
	cp -n .env.example .env 2>/dev/null || true
	docker compose up --build

## Start infrastructure only (postgres, redis, minio)
infra:
	cp -n .env.example .env 2>/dev/null || true
	docker compose up postgres redis minio minio-init --build

## Stop all services
down:
	docker compose down

## Follow logs for all services
logs:
	docker compose logs -f

## Rebuild and restart a single service (usage: make restart s=api)
restart:
	docker compose up --build -d $(s)

# ---------------------------------------------------------------------------
# Local development (outside Docker)
# ---------------------------------------------------------------------------

## Run Go API locally (requires infra running)
api-dev:
	cd services/api && go run ./cmd/server

## Run Next.js dev server locally
web-dev:
	cd clients/web && npm run dev

## Build CLI binary
cli-build:
	cd clients/cli && CGO_ENABLED=0 go build -o bin/sitaware-cli .

## Run CLI locally (pass args via ARGS, e.g. make cli-dev ARGS="--token=sat_... --file=track.gpx")
cli-dev:
	cd clients/cli && go run . $(ARGS)

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

## Open psql shell
db-shell:
	docker compose exec postgres psql -U sitaware -d sitaware

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

## Build Go API binary
api-build:
	cd services/api && CGO_ENABLED=0 go build -o bin/server ./cmd/server

# ---------------------------------------------------------------------------
# Production (Docker Compose + Caddy TLS)
# ---------------------------------------------------------------------------

## Start production stack (Caddy + TLS, no Minio)
prod:
	docker compose -f docker-compose.prod.yml up --build -d

## Stop production stack
prod-down:
	docker compose -f docker-compose.prod.yml down

## Follow production logs
prod-logs:
	docker compose -f docker-compose.prod.yml logs -f

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

## Remove all containers and volumes
clean:
	docker compose down -v --remove-orphans

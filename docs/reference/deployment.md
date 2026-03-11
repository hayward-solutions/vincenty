# Deployment

Vincenty supports multiple deployment targets. Choose the one that fits your environment.

## Docker Compose (Development)

Single command starts the full stack:

```bash
make dev
```

Runs PostgreSQL+PostGIS, Redis, Minio, Go API, and Next.js web client. Minio provides S3-compatible storage locally with automatic bucket creation.

## Docker Compose (Production)

Uses Caddy for TLS termination. Place your TLS certificate and key in `deploy/caddy/certs/`, configure the Caddyfile, then:

```bash
make prod
```

No ports are exposed except 80/443 through Caddy. Resource limits are applied to all containers.

## Kubernetes

Raw manifests in `deploy/k8s/`:

```bash
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/
```

Includes namespace, configmap, secret, StatefulSet for PostgreSQL, deployments for Redis/API/web, services, and Ingress with TLS. The API runs 2 replicas behind a Service.

## Helm

```bash
helm install vincenty deploy/helm/vincenty/ \
  --namespace vincenty \
  --create-namespace \
  -f my-values.yaml
```

Fully parameterized via `values.yaml`. Supports toggling between in-cluster and external PostgreSQL/Redis (RDS, ElastiCache).

## AWS ECS Fargate

Task definitions and service configs in `deploy/ecs/`. See the [ECS deployment guide](https://github.com/hayward-solutions/vincenty/blob/main/deploy/ecs/README.md) for the full walkthrough.

Uses ALB for TLS, RDS for PostgreSQL, ElastiCache for Redis, S3 for storage, SSM Parameter Store for secrets, and IAM task roles for S3 access.

## Air-Gapped Deployment

Vincenty is designed to run with zero internet access:

1. Pre-pull and export all container images
2. Upload map tiles to the S3-compatible object store (Minio in Docker Compose, or any S3 endpoint)
3. Configure `MAP_DEFAULT_TILE_URL` to point to the local tile source
4. All UI assets are bundled — no CDN, no external fonts, no external scripts
5. Deploy with `docker compose up`

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
make cli-build    # Build CLI binary
make prod         # Start production stack (Caddy + TLS)
make prod-down    # Stop production stack
make prod-logs    # Tail production logs
make clean        # Remove all containers and volumes
```

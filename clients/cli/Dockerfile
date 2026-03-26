# =============================================================================
# Build stage
# =============================================================================
FROM golang:alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o /build/vincenty-cli .

# =============================================================================
# Runtime stage
# =============================================================================
FROM gcr.io/distroless/static-debian12

COPY --from=builder /build/vincenty-cli /vincenty-cli

ENTRYPOINT ["/vincenty-cli"]

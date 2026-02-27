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
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/sitaware-cli .

# =============================================================================
# Runtime stage
# =============================================================================
FROM gcr.io/distroless/static-debian12

COPY --from=builder /build/sitaware-cli /sitaware-cli

ENTRYPOINT ["/sitaware-cli"]

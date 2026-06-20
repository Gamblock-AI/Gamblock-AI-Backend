# syntax=docker/dockerfile:1
# Multi-stage build for the Gamblock-AI Go backend.
# Builds the API binary in a full Go toolchain image, then ships a minimal
# alpine runtime. Production config comes from environment variables (.env in
# the deploy) — never baked into the image.

# ---- Build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache deps first.
COPY go.mod go.sum ./
RUN go mod download

# Build. CGO disabled for a static binary compatible with alpine.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api

# ---- Runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/api /app/api

# Artifact storage dir (mapped to a volume in production).
RUN mkdir -p /app/var/artifacts /app/var/exports && chown -R app:app /app
USER app

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=10s \
  CMD wget -qO- --tries=1 --spider http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/api"]

# Build arguments for version customization
ARG NODE_VERSION=22
ARG GO_VERSION=1.25
ARG ALPINE_VERSION=latest

# Stage 1: Build frontend
FROM node:${NODE_VERSION}-alpine AS frontend

WORKDIR /app

# Create directory for frontend build output
RUN mkdir -p /app/internal/ui

# Copy frontend package files
COPY web/package.json web/pnpm-lock.yaml ./web/

# Install pnpm
RUN npm install -g pnpm

# Install dependencies
RUN cd web && pnpm install --frozen-lockfile

# Copy web source code
COPY web/ ./web/

# Build frontend (outputs to ../internal/ui/dist)
RUN cd web && pnpm build

# Stage 2: Build backend
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Install build dependencies (CGO required for sqlite3, git for pseudo-version modules)
RUN apk add --no-cache gcc musl-dev make git

# Copy go mod files
COPY go.mod go.sum ./

# GOPROXY argument for faster downloads (can be overridden at build time)
ARG GOPROXY=direct
ENV GOPROXY=${GOPROXY}

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build output (frontend builds to ../internal/ui/dist)
COPY --from=frontend /app/internal/ui/dist ./internal/ui/dist/

# Enable CGO and build
ENV CGO_ENABLED=1
RUN go build -o cloud-sync ./cmd/cloud-sync

# Stage 3: Runtime
FROM alpine:${ALPINE_VERSION}

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/cloud-sync .

# Create data directory
RUN mkdir -p /app/app_data

# Expose port
EXPOSE 8080

# Set working directory for data
VOLUME ["/app/app_data"]

# Run the application
CMD ["./cloud-sync", "serve"]

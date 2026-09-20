# WordCube Application Deployment

This repository contains a full-stack WordCube application with a React frontend and Go backend.

## Quick Start

### Development Setup
```bash
./dev-setup.sh
```

### Production Deployment
```bash
./build-and-run.sh
```

## Environment Configuration

The frontend uses Vite environment variables for API configuration:

- **`.env.development`** - Used during `npm run dev`
- **`.env.production`** - Used during `npm run build` 
- **`.env.local`** - Local overrides (not committed to git)
- **`.env.example`** - Template file

### Environment Variables

- `VITE_API_BASE_URL` - Base URL for API calls
  - **Development**: Your codespace URL (e.g., `https://animated-meme-9x46vwrrpcxvv6-1323.app.github.dev`)
  - **Production**: Empty string (uses relative URLs)
  - **Local**: `http://localhost:1323`

## Deployment Options

### Using the build script (recommended)
```bash
./build-and-run.sh
```

### Using Docker Compose
```bash
docker-compose up --build
```

### Manual Docker build
```bash
# Build the image
docker build -t wordcube:latest .

# Run the container
docker run -p 1323:1323 wordcube:latest
```

## Backend Security & Networking Configuration

The Go server (`server/app/server.go`) reads the following environment variables to
control its networking/security posture. All are optional; defaults are chosen to
preserve existing dev/Codespaces behavior out of the box.

- `PORT` - the TCP port the server listens on (plain HTTP or TLS, whichever is
  active). Defaults to `1323`, matching the port `docker-compose.yml` maps and has
  always set via this same env var. Falls back to the default with a logged warning
  if set to something that isn't a valid port number (1-65535).
- `ALLOWED_ORIGINS` - comma-separated list of origins allowed by CORS
  (e.g. `https://wordcube.example.com,https://staging.wordcube.example.com`).
  Defaults to `*` (all origins allowed), since the frontend may be served from an
  arbitrary dev-environment URL. **Set this to your real frontend origin(s) in
  production** rather than relying on the wildcard default.
- `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` - per-client-IP token-bucket limits applied
  to `POST /solutions` only (the one endpoint that runs a real search against the
  in-memory index). Defaults to 5 requests/second with a burst of 10. Requests over
  the limit receive HTTP 429.
- `TRUST_PROXY_HEADERS` - set to `true` only if this server sits behind a trusted
  reverse proxy that sets `X-Forwarded-For`. Defaults to `false`, meaning the
  server (and the rate limiter above) identifies clients by the raw TCP connection
  address only. **Do not set this unless you actually have a trusted proxy in
  front of the container** - since neither `docker-compose.yml` nor the
  `Dockerfile` in this repo add one by default, trusting these headers out of the
  box would let any client bypass per-IP rate limiting by sending a different
  `X-Forwarded-For` value on every request.
- `TLS_CERT_FILE` / `TLS_KEY_FILE` - paths to a PEM certificate and private key. If
  both are set, the server terminates TLS itself (`tls.Config{MinVersion:
  tls.VersionTLS12}`) and listens with HTTPS on the configured `PORT` instead of
  plain HTTP. If unset (the default, including the current Docker/Compose setup),
  the server serves plain HTTP and expects TLS to be terminated in front of it -
  e.g. by a reverse proxy (nginx, Caddy, an ALB/Cloud Load Balancer, etc.) or a
  platform that handles TLS for you. Neither `docker-compose.yml` nor the
  `Dockerfile` in this repo currently set up such a proxy, so anyone deploying this
  image directly on the open internet must either supply
  `TLS_CERT_FILE`/`TLS_KEY_FILE` or put a TLS-terminating proxy in front of
  container port `1323` (or whatever `PORT` is set to).

These, along with connection timeouts, security headers (`X-Content-Type-Options`,
`X-Frame-Options`, HSTS), a 64K request body limit, and panic recovery, are covered
by the test suite in `server/app/server_test.go`.

## Application Structure

- **Frontend**: React application built with Vite (in `site/` directory)
- **Backend**: Go server using Echo framework (in `server/` directory)
- **Main entry point**: `server/app/server.go`

## Deployment Details

The Dockerfile uses a multi-stage build process:

1. **Stage 1**: Builds the frontend using Node.js 18
   - Installs npm dependencies
   - Runs `npm run build` to create production assets

2. **Stage 2**: Builds the Go server binary
   - Uses Go 1.25
   - Compiles the server from `server/app/server.go`

3. **Stage 3**: Creates final lightweight Alpine Linux image
   - Copies the built Go binary
   - Copies frontend build assets to `static/` directory
   - Copies required data files
   - Exposes port 1323

## API Endpoints

- `GET /solutions` - Get solutions for an empty board
- `POST /solutions` - Get solutions for a specific board state
- `GET /` - Serves the frontend application

## Environment

- **Port**: 1323
- **Frontend assets**: Served from `/static` directory
- **Data files**: Located in `app/data/`

## Development

For local development without Docker:

### Frontend (site/)
```bash
cd site
npm install
npm run dev
```

### Backend (server/)
```bash
cd server
go run app/server.go
```

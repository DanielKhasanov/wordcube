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
   - Uses Go 1.24
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

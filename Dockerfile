# Multi-stage build for WordCube application

# Stage 1: Build the frontend
FROM node:18-alpine AS frontend-builder

WORKDIR /app/site
COPY site/package*.json ./
RUN npm ci

COPY site/ ./
RUN npm run build

# Stage 2: Build the Go server
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./app/server.go

# Stage 3: Final production image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates
RUN mkdir /app

WORKDIR /app

# Copy the built Go binary
COPY --from=backend-builder /app/main .

# Copy the built frontend assets
COPY --from=frontend-builder /app/site/dist ./static

# Copy server data files
COPY --from=backend-builder /app/app/data ./app/data

# Expose port 1323 (the port your Go server runs on)
EXPOSE 1323

# Run the binary
CMD ["./main"]

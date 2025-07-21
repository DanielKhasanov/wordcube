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

# Build the CLI tool for generating solutions
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cli-tool ./cli/main.go

# Generate solutions.textpb if it doesn't exist
RUN if [ ! -f ./app/data/solutions.textpb ]; then \
        echo "Generating solutions.textpb..."; \
        NUM_CORES=$(nproc); \
        NUM_PARTITIONS=$((NUM_CORES < 4 ? NUM_CORES : 4)); \
        echo "Using $NUM_PARTITIONS partitions"; \
        ./cli-tool --mode=find_solutions --output_dir=app/data/ --word_list=app/data/all_words_5.txt --num_partitions=$NUM_PARTITIONS; \
    else \
        echo "solutions.textpb already exists, skipping generation"; \
    fi

# Build the main server binary
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

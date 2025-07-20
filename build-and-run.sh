#!/bin/bash

# Build and run the WordCube application using Docker

echo "Building WordCube application..."

# Build the Docker image
docker build -t wordcube:latest .

if [ $? -eq 0 ]; then
    echo "Build successful! Starting the application..."
    echo "The application will be available at http://localhost:1323"
    echo "Press Ctrl+C to stop the application"
    
    # Run the container
    docker run -p 1323:1323 --rm wordcube:latest
else
    echo "Build failed!"
    exit 1
fi

#!/bin/bash

# Development setup script for WordCube

echo "Setting up WordCube development environment..."

# Check if .env.local exists
if [ ! -f "site/.env.local" ]; then
    echo "Creating .env.local from template..."
    cp site/.env.example site/.env.local
    echo "Please edit site/.env.local with your development API URL"
fi

# Install frontend dependencies
echo "Installing frontend dependencies..."
cd site
npm install

echo "Setup complete!"
echo ""
echo "To start development:"
echo "1. Edit site/.env.local with your API URL"
echo "2. Frontend: cd site && npm run dev"
echo "3. Backend: cd server && go run app/server.go"
echo ""
echo "For production deployment: ./build-and-run.sh"

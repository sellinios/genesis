#!/bin/bash
set -e
cd "$(dirname "$0")"

# Install Docker if not present
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    sudo apt-get update
    sudo apt-get install -y docker.io docker-compose-v2
    sudo systemctl start docker
    sudo systemctl enable docker
    sudo usermod -aG docker $USER
    echo "Docker installed. Run this script again."
    exit 0
fi

# Start Genesis
docker compose up -d --build
echo ""
echo "Genesis: http://localhost:8090"

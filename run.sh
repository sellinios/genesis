#!/bin/bash
set -e
cd "$(dirname "$0")"

# Update system (optional, skip if locked)
echo "Updating system..."
sudo apt-get update -qq 2>/dev/null || echo "Skipping update (locked)"
sudo apt-get upgrade -y -qq 2>/dev/null || echo "Skipping upgrade (locked)"

# Install Docker if not present
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    sudo apt-get install -y docker.io docker-compose-v2 docker-buildx
    sudo systemctl start docker
    sudo systemctl enable docker
    sudo usermod -aG docker $USER
    echo "Docker installed. Run this script again."
    exit 0
fi

# Install buildx if missing
if ! docker buildx version &> /dev/null; then
    echo "Installing Docker Buildx..."
    sudo apt-get install -y docker-buildx 2>/dev/null || true
fi

# Create .env if not exists
if [ ! -f .env ]; then
    echo "Creating .env with secure random passwords..."
    DB_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)
    JWT_SECRET=$(openssl rand -base64 32)
    cat > .env << EOF
DB_USER=genesis
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=genesis
JWT_SECRET=${JWT_SECRET}
EOF
    chmod 600 .env
    echo ".env created with secure credentials"
fi

# Start Genesis
sudo docker compose up -d --build
echo ""
echo "Genesis: http://localhost:8090"

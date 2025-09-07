#!/bin/bash

# NFL Discord Bot - Docker Deployment Script
# This script automates the Docker deployment process

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}"
    exit 1
}

# Check if .env file exists
check_env_file() {
    if [[ ! -f .env ]]; then
        warning ".env file not found. Creating from template..."
        if [[ -f .env.example ]]; then
            cp .env.example .env
            warning "Please edit .env file with your Discord token and NFL API key"
            warning "Required variables: DISCORD_TOKEN, NFL_API_KEY"
            read -p "Press Enter after updating .env file..."
        else
            error ".env.example not found. Please create .env file manually."
        fi
    fi
}

# Build Docker image
build_image() {
    log "Building Docker image..."
    docker build -t nfl-discord-bot:latest .
    success "Docker image built successfully"
}

# Deploy with Docker Compose
deploy_compose() {
    log "Deploying with Docker Compose..."
    
    # Create necessary directories
    mkdir -p logs data
    
    # Stop existing container if running
    docker-compose down || true
    
    # Start the bot
    docker-compose up -d
    
    success "Bot deployed successfully with Docker Compose"
}

# Deploy standalone Docker container
deploy_standalone() {
    log "Deploying standalone Docker container..."
    
    # Create necessary directories
    mkdir -p logs data
    
    # Stop and remove existing container
    docker stop nfl-discord-bot 2>/dev/null || true
    docker rm nfl-discord-bot 2>/dev/null || true
    
    # Run the container
    docker run -d \
        --name nfl-discord-bot \
        --restart unless-stopped \
        --env-file .env \
        -v "$(pwd)/logs:/app/logs" \
        -v "$(pwd)/data:/app/data" \
        --memory="512m" \
        --cpus="0.5" \
        nfl-discord-bot:latest
    
    success "Bot deployed successfully as standalone container"
}

# Show container status
show_status() {
    log "Container status:"
    docker ps --filter "name=nfl-discord-bot" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    
    log "Recent logs:"
    docker logs --tail 20 nfl-discord-bot 2>/dev/null || true
}

# Main deployment function
main() {
    log "Starting NFL Discord Bot deployment..."
    
    # Check requirements
    command -v docker >/dev/null 2>&1 || error "Docker is required but not installed"
    
    # Check environment
    check_env_file
    
    # Build image
    build_image
    
    # Choose deployment method
    echo ""
    echo "Choose deployment method:"
    echo "1) Docker Compose (recommended)"
    echo "2) Standalone Docker container"
    read -p "Enter choice (1-2): " choice
    
    case $choice in
        1)
            command -v docker-compose >/dev/null 2>&1 || error "Docker Compose is required but not installed"
            deploy_compose
            ;;
        2)
            deploy_standalone
            ;;
        *)
            error "Invalid choice"
            ;;
    esac
    
    # Show status
    echo ""
    show_status
    
    success "Deployment completed!"
    log "Use 'docker logs -f nfl-discord-bot' to view live logs"
}

# Handle script arguments
case "${1:-}" in
    "build")
        build_image
        ;;
    "deploy")
        deploy_compose
        ;;
    "standalone")
        deploy_standalone
        ;;
    "status")
        show_status
        ;;
    *)
        main
        ;;
esac

#!/bin/bash

# NFL Discord Bot - Systemd Deployment Script
# This script automates the systemd service deployment on Linux

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="nfl-discord-bot"
INSTALL_DIR="/opt/nfl-discord-bot"
SERVICE_USER="nflbot"
SERVICE_GROUP="nflbot"

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

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (use sudo)"
    fi
}

# Check system requirements
check_requirements() {
    log "Checking system requirements..."
    
    # Check systemd
    command -v systemctl >/dev/null 2>&1 || error "systemctl not found. This script requires systemd."
    
    # Check Go installation
    command -v go >/dev/null 2>&1 || error "Go is not installed. Please install Go 1.21+ first."
    
    # Check Go version
    go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    if [[ $(echo "$go_version < 1.21" | bc -l 2>/dev/null) == 1 ]]; then
        error "Go version $go_version found. Requires Go 1.21 or higher."
    fi
    
    success "System requirements satisfied"
}

# Create service user
create_user() {
    log "Creating service user and group..."
    
    if ! getent group $SERVICE_GROUP >/dev/null 2>&1; then
        groupadd --system $SERVICE_GROUP
        success "Created group: $SERVICE_GROUP"
    else
        log "Group $SERVICE_GROUP already exists"
    fi
    
    if ! getent passwd $SERVICE_USER >/dev/null 2>&1; then
        useradd --system --gid $SERVICE_GROUP --create-home \
                --home-dir /var/lib/$SERVICE_USER --shell /bin/false \
                --comment "NFL Discord Bot Service User" $SERVICE_USER
        success "Created user: $SERVICE_USER"
    else
        log "User $SERVICE_USER already exists"
    fi
}

# Build the application
build_application() {
    log "Building NFL Discord Bot..."
    
    # Build with optimizations
    CGO_ENABLED=0 go build \
        -a -installsuffix cgo \
        -ldflags="-w -s" \
        -o nfl-bot \
        cmd/nfl-bot/main.go
    
    success "Application built successfully"
}

# Install application files
install_application() {
    log "Installing application to $INSTALL_DIR..."
    
    # Create installation directory
    mkdir -p $INSTALL_DIR/{logs,data}
    
    # Copy binary
    cp nfl-bot $INSTALL_DIR/
    chmod +x $INSTALL_DIR/nfl-bot
    
    # Copy configuration template
    cp .env.example $INSTALL_DIR/
    if [[ ! -f $INSTALL_DIR/.env ]]; then
        cp .env.example $INSTALL_DIR/.env
        warning "Created .env from template. Please edit $INSTALL_DIR/.env with your credentials."
    fi
    
    # Copy documentation
    cp README.md SLASH_COMMANDS.md WARP.md $INSTALL_DIR/ 2>/dev/null || true
    
    # Set ownership
    chown -R $SERVICE_USER:$SERVICE_GROUP $INSTALL_DIR
    chmod 600 $INSTALL_DIR/.env  # Secure the environment file
    
    success "Application installed to $INSTALL_DIR"
}

# Install systemd service
install_service() {
    log "Installing systemd service..."
    
    # Copy service file
    cp $SERVICE_NAME.service /etc/systemd/system/
    
    # Reload systemd
    systemctl daemon-reload
    
    success "Systemd service installed"
}

# Configure and start service
start_service() {
    log "Starting $SERVICE_NAME service..."
    
    # Enable service to start on boot
    systemctl enable $SERVICE_NAME
    
    # Start the service
    systemctl start $SERVICE_NAME
    
    success "Service started successfully"
}

# Show service status
show_status() {
    log "Service status:"
    systemctl status $SERVICE_NAME --no-pager || true
    
    echo ""
    log "Recent logs (last 20 lines):"
    journalctl -u $SERVICE_NAME --no-pager -n 20 || true
}

# Configure firewall (optional)
configure_firewall() {
    if command -v ufw >/dev/null 2>&1; then
        log "UFW detected. No additional firewall rules needed for Discord bot."
    elif command -v firewall-cmd >/dev/null 2>&1; then
        log "Firewalld detected. No additional firewall rules needed for Discord bot."
    else
        log "No supported firewall detected. Manual configuration may be needed."
    fi
}

# Main installation function
main() {
    log "Starting NFL Discord Bot systemd installation..."
    
    # Pre-flight checks
    check_root
    check_requirements
    
    # Create user and build
    create_user
    build_application
    
    # Install application and service
    install_application
    install_service
    
    # Configure firewall
    configure_firewall
    
    # Start service
    start_service
    
    # Show status
    echo ""
    show_status
    
    success "Installation completed!"
    echo ""
    echo "Useful commands:"
    echo "  sudo systemctl status $SERVICE_NAME    # Check service status"
    echo "  sudo systemctl restart $SERVICE_NAME   # Restart service"
    echo "  sudo systemctl stop $SERVICE_NAME      # Stop service"
    echo "  sudo journalctl -u $SERVICE_NAME -f    # View live logs"
    echo "  sudo systemctl edit $SERVICE_NAME      # Edit service configuration"
    echo ""
    warning "Don't forget to edit $INSTALL_DIR/.env with your Discord token and NFL API key!"
}

# Handle script arguments
case "${1:-}" in
    "build")
        build_application
        ;;
    "install")
        check_root
        install_application
        install_service
        ;;
    "start")
        check_root
        start_service
        ;;
    "status")
        show_status
        ;;
    "uninstall")
        check_root
        systemctl stop $SERVICE_NAME 2>/dev/null || true
        systemctl disable $SERVICE_NAME 2>/dev/null || true
        rm -f /etc/systemd/system/$SERVICE_NAME.service
        systemctl daemon-reload
        warning "Service uninstalled. $INSTALL_DIR directory left intact."
        ;;
    *)
        main
        ;;
esac

# 🚀 NFL Discord Bot - Deployment Guide

This guide covers multiple deployment options for the NFL Discord Bot, from simple local development to production-ready containerized and systemd deployments.

## 📋 Prerequisites

### Required
- **Go 1.21+** ([Download](https://golang.org/dl/))
- **Discord Bot Token** ([Setup Guide](README.md#discord-setup))
- **SportsData.io API Key** ([Get Free Key](https://sportsdata.io))

### For Docker Deployment
- **Docker** ([Install Guide](https://docs.docker.com/get-docker/))
- **Docker Compose** (optional, recommended)

### For Linux Systemd Deployment
- **Linux with systemd** (Ubuntu 18+, CentOS 7+, etc.)
- **Root access** (sudo privileges)

---

## 🐳 Docker Deployment (Recommended)

### Quick Start with Docker Compose

1. **Clone and setup environment:**
   ```bash
   git clone <your-repo-url>
   cd nfl-discord-bot
   cp .env.example .env
   # Edit .env with your tokens
   ```

2. **Deploy with one command:**
   ```bash
   chmod +x scripts/deploy-docker.sh
   ./scripts/deploy-docker.sh
   ```

3. **Or manually:**
   ```bash
   # Build and start
   docker-compose up -d
   
   # View logs
   docker-compose logs -f
   
   # Stop
   docker-compose down
   ```

### Manual Docker Deployment

1. **Build the image:**
   ```bash
   docker build -t nfl-discord-bot:latest .
   ```

2. **Run the container:**
   ```bash
   docker run -d \
     --name nfl-discord-bot \
     --restart unless-stopped \
     --env-file .env \
     -v ./logs:/app/logs \
     -v ./data:/app/data \
     nfl-discord-bot:latest
   ```

3. **Manage the container:**
   ```bash
   # View logs
   docker logs -f nfl-discord-bot
   
   # Stop
   docker stop nfl-discord-bot
   
   # Start
   docker start nfl-discord-bot
   
   # Remove
   docker rm nfl-discord-bot
   ```

### Docker Features
- ✅ **Multi-stage build** for minimal image size (~30MB)
- ✅ **Non-root user** for security
- ✅ **Health checks** for monitoring
- ✅ **Resource limits** (512MB RAM, 0.5 CPU)
- ✅ **Log rotation** (10MB max, 3 files)
- ✅ **Auto-restart** on failure

---

## 🖥️ Linux Systemd Deployment

### Automated Installation

1. **Run the deployment script:**
   ```bash
   chmod +x scripts/deploy-systemd.sh
   sudo ./scripts/deploy-systemd.sh
   ```

2. **Edit configuration:**
   ```bash
   sudo nano /opt/nfl-discord-bot/.env
   # Add your Discord token and NFL API key
   ```

3. **Restart the service:**
   ```bash
   sudo systemctl restart nfl-discord-bot
   ```

### Manual Installation

1. **Build the application:**
   ```bash
   go build -o nfl-bot cmd/nfl-bot/main.go
   ```

2. **Create service user:**
   ```bash
   sudo groupadd --system nflbot
   sudo useradd --system --gid nflbot --home-dir /var/lib/nflbot nflbot
   ```

3. **Install files:**
   ```bash
   sudo mkdir -p /opt/nfl-discord-bot/{logs,data}
   sudo cp nfl-bot /opt/nfl-discord-bot/
   sudo cp .env.example /opt/nfl-discord-bot/.env
   sudo chown -R nflbot:nflbot /opt/nfl-discord-bot
   sudo chmod 600 /opt/nfl-discord-bot/.env
   ```

4. **Install systemd service:**
   ```bash
   sudo cp nfl-discord-bot.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable nfl-discord-bot
   sudo systemctl start nfl-discord-bot
   ```

### Systemd Management Commands

```bash
# Service status
sudo systemctl status nfl-discord-bot

# Start/Stop/Restart
sudo systemctl start nfl-discord-bot
sudo systemctl stop nfl-discord-bot
sudo systemctl restart nfl-discord-bot

# Enable/Disable auto-start
sudo systemctl enable nfl-discord-bot
sudo systemctl disable nfl-discord-bot

# View logs
sudo journalctl -u nfl-discord-bot -f        # Live logs
sudo journalctl -u nfl-discord-bot -n 50     # Last 50 lines
sudo journalctl -u nfl-discord-bot --since today  # Today's logs
```

### Systemd Features
- ✅ **Auto-restart** on failure
- ✅ **Resource limits** (512MB RAM, 50% CPU)
- ✅ **Security hardening** (NoNewPrivileges, PrivateTmp)
- ✅ **Journald logging** with systemd integration
- ✅ **Boot startup** automatic start on system boot

---

## 🛠️ Development Deployment

### Local Development

1. **Setup environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your tokens
   ```

2. **Run directly:**
   ```bash
   go run cmd/nfl-bot/main.go
   ```

3. **Or build and run:**
   ```bash
   go build -o bin/nfl-bot cmd/nfl-bot/main.go
   ./bin/nfl-bot
   ```

### Development with Docker

```bash
# Development with live reload (if using air)
docker run -it --rm \
  -v $(pwd):/app \
  -w /app \
  --env-file .env \
  golang:1.21-alpine \
  go run cmd/nfl-bot/main.go
```

---

## 🔧 Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_TOKEN` | ✅ Yes | - | Discord bot token |
| `NFL_API_KEY` | ✅ Yes | - | SportsData.io API key |
| `NFL_API_BASE_URL` | ❌ No | `https://api.sportsdata.io/v3/nfl` | API base URL |
| `BOT_PREFIX` | ❌ No | `!` | Command prefix |
| `COMMAND_COOLDOWN` | ❌ No | `3` | Cooldown between commands (seconds) |
| `MAX_CONCURRENT_REQUESTS` | ❌ No | `10` | Max concurrent API requests |
| `LOG_LEVEL` | ❌ No | `info` | Logging level |
| `LOG_FILE` | ❌ No | `bot.log` | Log file path |
| `BOT_ALLOWED_ROLE` | ❌ No | - | Role required to use bot |
| `BOT_VISIBILITY_ROLE` | ❌ No | - | **Controls slash command visibility** |

### Message Visibility Control

**Important**: The `BOT_VISIBILITY_ROLE` setting controls whether slash command responses are public or private:

- **Not set** (`BOT_VISIBILITY_ROLE=`) → All commands public (everyone sees responses)
- **Set to any role** (`BOT_VISIBILITY_ROLE="Members"`) → Slash commands are ephemeral (only user sees them)

This allows strategic usage:
- Use `/stats` for private research
- Use `!stats` for sharing with the channel

### File Locations

#### Docker Deployment
- **Application**: `/app/` (inside container)
- **Logs**: `./logs/` (host) → `/app/logs` (container)
- **Data**: `./data/` (host) → `/app/data` (container)
- **Config**: `.env` file in project root

#### Systemd Deployment
- **Application**: `/opt/nfl-discord-bot/`
- **Logs**: `/opt/nfl-discord-bot/logs/`
- **Config**: `/opt/nfl-discord-bot/.env`
- **Service**: `/etc/systemd/system/nfl-discord-bot.service`

---

## 📊 Monitoring and Maintenance

### Health Checks

#### Docker
```bash
# Container health
docker ps --format "table {{.Names}}\t{{.Status}}"

# Application logs
docker logs --tail 50 nfl-discord-bot
```

#### Systemd
```bash
# Service health
systemctl status nfl-discord-bot

# Application logs
journalctl -u nfl-discord-bot --since "1 hour ago"
```

### Log Management

#### Docker Compose Logging
- **Rotation**: Automatic (10MB files, 3 files max)
- **Format**: JSON structured logging
- **Access**: `docker logs` command

#### Systemd Logging
- **Storage**: Systemd journal (`journalctl`)
- **Rotation**: Automatic systemd rotation
- **Format**: Structured logging with metadata

### Performance Monitoring

```bash
# Resource usage (Docker)
docker stats nfl-discord-bot

# Resource usage (Systemd)
systemctl show nfl-discord-bot --property=MemoryCurrent,CPUUsageNSec
```

---

## 🔒 Security Best Practices

### Container Security
- ✅ Non-root user execution
- ✅ Read-only filesystem where possible
- ✅ Resource limits enforced
- ✅ No privileged containers
- ✅ Regular base image updates

### Systemd Security
- ✅ Dedicated service user
- ✅ Restricted file permissions (600 for .env)
- ✅ NoNewPrivileges enabled
- ✅ PrivateTmp for isolation
- ✅ ProtectSystem enabled

### General Security
- ✅ Environment variables for secrets
- ✅ No hardcoded credentials
- ✅ Regular dependency updates
- ✅ Minimal attack surface

---

## 🚨 Troubleshooting

### Common Issues

#### "Discord token is invalid"
```bash
# Check environment variable
echo $DISCORD_TOKEN  # Should not be empty

# Verify token in Discord Developer Portal
# Regenerate token if necessary
```

#### "NFL API key errors"
```bash
# Test API connectivity
curl "https://api.sportsdata.io/v3/nfl/scores/json/Teams?key=YOUR_KEY"

# Check SportsData.io dashboard for usage limits
```

#### Container won't start
```bash
# Check logs for specific error
docker logs nfl-discord-bot

# Verify environment file
docker run --rm --env-file .env alpine env | grep -E "(DISCORD|NFL)"
```

#### Service fails to start
```bash
# Check service status
sudo systemctl status nfl-discord-bot

# Check detailed logs
sudo journalctl -u nfl-discord-bot -n 100

# Verify file permissions
ls -la /opt/nfl-discord-bot/
```

### Getting Help

1. **Check logs** first (most issues are logged)
2. **Verify configuration** (environment variables)
3. **Test connectivity** (Discord/NFL API)
4. **Review documentation** (README.md, SLASH_COMMANDS.md)
5. **Check GitHub issues** for similar problems

---

## 🔄 Updates and Maintenance

### Updating the Bot

#### Docker Deployment
```bash
# Pull latest code
git pull origin main

# Rebuild and redeploy
./scripts/deploy-docker.sh build
docker-compose up -d --build
```

#### Systemd Deployment
```bash
# Pull latest code and rebuild
git pull origin main
go build -o nfl-bot cmd/nfl-bot/main.go

# Update installation
sudo cp nfl-bot /opt/nfl-discord-bot/
sudo systemctl restart nfl-discord-bot
```

### Backup and Restore

#### Important Files to Backup
- `.env` (configuration)
- `logs/` directory (if needed)
- `data/` directory (if used)

#### Backup Script Example
```bash
#!/bin/bash
backup_dir="/backup/nfl-bot-$(date +%Y%m%d)"
mkdir -p "$backup_dir"
cp .env "$backup_dir/"
cp -r logs/ "$backup_dir/" 2>/dev/null || true
tar czf "$backup_dir.tar.gz" "$backup_dir"
```

---

**🎉 Your NFL Discord Bot is now ready for production deployment!**

Choose the deployment method that best fits your infrastructure and requirements. Both Docker and systemd deployments are production-ready with proper security, monitoring, and maintenance features.

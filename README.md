# 🏈 NFL Discord Bot

**A powerful, intelligent Discord bot that provides real-time NFL statistics, schedules, and live scores with smart week detection and caching.**

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Discord](https://img.shields.io/badge/Discord-API-blurple.svg)](https://discord.com/developers/docs)
[![SportsData.io](https://img.shields.io/badge/API-SportsData.io-orange.svg)](https://sportsdata.io)

## ✨ Features

- 📊 **Real-time NFL Statistics** - Current week and season totals for any player
- ⚖️ **Player Comparisons** - Side-by-side stats with visual indicators
- 🏟 **Complete Team Information** - Details, coaches, stadiums, divisions
- 📅 **Full Season Schedules** - Games, scores, dates, and BYE weeks
- 🔴 **Live Scores** - Real-time game updates and results
- 🧠 **Smart Week Detection** - Automatically detects current NFL week with intelligent day-of-week logic
- ⚡ **5-Minute Caching** - Fast responses with automatic data caching
- 📱 **Flexible Team Names** - Works with full names, cities, or abbreviations
- 🔧 **Comprehensive Logging** - Full request tracking and debugging

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+** ([Download](https://golang.org/dl/))
- **Discord Bot Token** ([Setup Guide](#discord-setup))
- **SportsData.io API Key** ([Get Free Key](https://sportsdata.io))

### 1️⃣ Clone the Repository
```bash
git clone https://github.com/your-username/nfl-discord-bot.git
cd nfl-discord-bot
```

### 2️⃣ Configure Environment
```bash
cp .env.example .env
```

Edit `.env` with your credentials:
```env
# Required
DISCORD_TOKEN=your_discord_bot_token_here
NFL_API_KEY=your_sportsdata_io_api_key_here

# Optional (defaults provided)
BOT_PREFIX=!
LOG_LEVEL=info
```

### 3️⃣ Build and Run
```bash
# Install dependencies
go mod tidy

# Build the bot
go build -o bin/nfl-bot cmd/nfl-bot/main.go

# Run the bot
./bin/nfl-bot
```

**Or run directly:**
```bash
go run cmd/nfl-bot/main.go
```

## 🤖 Discord Setup

### Create Discord Application
1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click **"New Application"** and name it (e.g., "NFL Stats Bot")
3. Go to **"Bot"** section and click **"Add Bot"**
4. Copy the **Token** and add to your `.env` file
5. Enable **"Message Content Intent"** in Privileged Gateway Intents

### Invite Bot to Server
1. Go to **"OAuth2" → "URL Generator"**
2. Select scopes: ☑️ `bot`
3. Select permissions: ☑️ `Send Messages`, ☑️ `Embed Links`, ☑️ `Read Message History`
4. Copy the generated URL and open in browser
5. Select your Discord server and authorize

## 📝 Commands Reference

### 📊 Player Statistics
```
!stats <player_name>           # Current week stats
!stats --season <player_name>  # Full season totals
!stats --week <#> <player_name> # Specific week stats
```
**Examples:**
- `!stats Josh Allen` - Current week performance
- `!stats --season Saquon Barkley` - Season totals
- `!stats --week 5 Patrick Mahomes` - Week 5 stats

### ⚖️ Player Comparisons
```
!compare <player1> vs <player2>                    # Compare current week
!compare --season <player1> vs <player2>           # Compare season stats
!compare --week <#> <player1> vs <player2>         # Compare specific week
```
**Examples:**
- `!compare Josh Allen vs Patrick Mahomes` - Current week head-to-head
- `!compare --season Saquon Barkley vs Derrick Henry` - Season comparison
- `!compare --week 5 Cooper Kupp vs Davante Adams` - Week 5 matchup

**Comparison Features:**
- 🔵🔴 Side-by-side stats with color coding
- ⬆️ Visual indicators for better performance
- 🏈 Position-specific stats (QB/RB/WR/TE)
- 📈 Advanced metrics (Completion%, YPC, YPR)

### 🏟️ Team Information
```
!team <team_name>  # Complete team details
```
**Examples:**
- `!team Bills` - Buffalo Bills info
- `!team KC` - Kansas City Chiefs info
- `!team New England` - Patriots info

### 📅 Team Schedule
```
!schedule <team_name>  # Full season schedule with BYE weeks
```
**Examples:**
- `!schedule Eagles` - Philadelphia Eagles schedule
- `!schedule Cowboys` - Dallas Cowboys schedule

### 🔴 Live Scores
```
!scores  # Current week's games and scores
```

### ❓ Help
```
!help  # Complete command guide
```

## 🧠 Smart Week Detection

The bot automatically detects the current NFL week with intelligent logic:

- ✅ **Tuesday**: New week begins
- ⚠️ **Wednesday**: Shows previous week (games recently completed)
- ✅ **Thursday-Monday**: Shows current week (upcoming/live games)

## 🛠️ Configuration

### Environment Variables
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_TOKEN` | ✅ Yes | - | Discord bot token |
| `NFL_API_KEY` | ✅ Yes | - | SportsData.io API key |
| `BOT_PREFIX` | ❌ No | `!` | Command prefix |
| `LOG_LEVEL` | ❌ No | `info` | Logging level (debug, info, warn, error) |
| `LOG_FILE` | ❌ No | `bot.log` | Log file path |
| `COMMAND_COOLDOWN` | ❌ No | `3` | Cooldown between commands (seconds) |
| `BOT_ALLOWED_ROLE` | ❌ No | - | Role required to use bot commands |
| `BOT_VISIBILITY_ROLE` | ❌ No | - | **Controls slash command visibility** |

## 🔥 Performance Features

- **⚡ 5-Minute Caching**: API responses cached for 5 minutes
- **📋 Smart Logging**: Request tracking and performance monitoring
- **🔄 Auto-Cleanup**: Expired cache entries automatically removed
- **⏱️ Rate Limiting**: Respects API rate limits

## 👁️ Message Visibility Control

### Ephemeral Slash Commands
Control who can see slash command responses with `BOT_VISIBILITY_ROLE`:

#### **Not Set (Default)**: All Public
```env
BOT_VISIBILITY_ROLE=
```
- **Traditional commands** (`!stats Josh Allen`) → **Everyone sees response**
- **Slash commands** (`/stats player:Josh Allen`) → **Everyone sees response**
- Perfect for **community servers** where sharing is encouraged

#### **Set to Role**: Slash Commands Private
```env
BOT_VISIBILITY_ROLE="VIP Members"
```
- **Traditional commands** (`!stats Josh Allen`) → **Everyone sees response** (for sharing)
- **Slash commands** (`/stats player:Josh Allen`) → **Only user sees response** (private)
- Perfect for **clean channels** – reduces spam while keeping sharing option

**💡 Pro Tip**: Use both command types strategically:
- Use `/stats` for personal research (private)
- Use `!stats` for sharing with the channel (public)

## 📝 Project Structure

```
nfl-discord-bot/
├── cmd/nfl-bot/main.go           # Application entry point
├── internal/
│   ├── bot/bot.go              # Discord bot logic and commands
│   ├── config/config.go        # Configuration management
│   └── nfl/client.go           # NFL API client with caching
├── pkg/models/models.go        # Data structures
├── .env.example                # Environment template
├── WARP.md                     # Development guide
└── README.md                   # This file
```

## 🐛 Development

### Local Development
```bash
# Run with live reload (if you have air installed)
air

# Or run directly
go run cmd/nfl-bot/main.go

# Build for production
go build -ldflags="-s -w" -o bin/nfl-bot cmd/nfl-bot/main.go
```

### Testing
```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Code Quality
```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run linter (if golangci-lint installed)
golangci-lint run
```

## 📚 API Documentation

This bot uses the [SportsData.io NFL API](https://sportsdata.io/developers/api-documentation/nfl) for real-time NFL data.

**Free Tier Includes:**
- 1,000 API calls per month
- Player statistics
- Team information
- Schedules and scores

## 🌐 Deployment

### Docker (Recommended)
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o bin/nfl-bot cmd/nfl-bot/main.go

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bin/nfl-bot .
CMD ["./nfl-bot"]
```

### Systemd Service
```ini
[Unit]
Description=NFL Discord Bot
After=network.target

[Service]
Type=simple
User=nflbot
WorkingDirectory=/opt/nfl-discord-bot
ExecStart=/opt/nfl-discord-bot/bin/nfl-bot
Restart=always
RestartSec=10
EnvironmentFile=/opt/nfl-discord-bot/.env

[Install]
WantedBy=multi-user.target
```

## 📞 Support

- **Documentation**: Check [WARP.md](WARP.md) for detailed development guide
- **Issues**: [GitHub Issues](https://github.com/your-username/nfl-discord-bot/issues)
- **API Support**: [SportsData.io Support](mailto:support@sportsdata.io)

**Made with ❤️ for NFL fans and Discord communities**

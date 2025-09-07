# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Overview

This is an NFL Discord Bot built in Go that provides NFL player statistics and team information through Discord commands. The bot uses the DiscordGo library and follows a clean architecture with separate packages for bot logic, NFL data fetching, configuration, and data models.

## Common Development Commands

### Build and Run
```bash
# Build the bot binary
go build -o bin/nfl-bot cmd/nfl-bot/main.go

# Run directly with Go
go run cmd/nfl-bot/main.go

# Run with environment variables
DISCORD_TOKEN=your_token_here go run cmd/nfl-bot/main.go
```

### Testing and Validation
```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Tidy dependencies
go mod tidy
```

## Configuration and Environment Setup

### Required Environment Variables
- `DISCORD_TOKEN` - Discord bot token (required)
- `NFL_API_KEY` - SportsData.io API key for NFL data fetching (optional for mock data)
- `NFL_API_BASE_URL` - Base URL for SportsData.io NFL API (default: "https://api.sportsdata.io/v3/nfl")

### Optional Configuration
- `BOT_PREFIX` - Command prefix (default: "!")
- `COMMAND_COOLDOWN` - Cooldown in seconds (default: 3)
- `MAX_CONCURRENT_REQUESTS` - Max concurrent API requests (default: 10)
- `STATS_UPDATE_INTERVAL` - Stats update interval in minutes (default: 30)
- `SCHEDULE_UPDATE_INTERVAL` - Schedule update interval in minutes (default: 1440)
- `LOG_LEVEL` - Logging level (default: "info")
- `LOG_FILE` - Log file path (default: "bot.log")

### Setup Steps
1. Copy `.env.example` to `.env`
2. Configure Discord bot token and API keys in `.env`
3. Run `go mod tidy` to install dependencies
4. Build and run the bot

## Bot Commands and Functionality

The bot responds to message-based commands with the configured prefix (default "!"):

### Available Commands
- `!help` - Shows all available commands with descriptions
- `!stats <player_name>` - Displays player statistics in a rich embed
- `!stats --season <player_name>` - Shows season aggregate stats
- `!stats --week <week> <player_name>` - Shows specific week stats
- `!compare <player1> vs <player2>` - Compare current week stats side-by-side
- `!compare --season <player1> vs <player2>` - Compare season stats
- `!compare --week <week> <player1> vs <player2>` - Compare specific week stats
- `!team <team_name>` - Shows comprehensive team information
- `!schedule <team_name>` - Shows full team schedule with BYE weeks
- `!scores` - Shows live scores and game results for current week

### Command Processing
- Commands are processed through the `messageCreate` handler in `internal/bot/bot.go`
- The bot ignores its own messages and only responds to messages with the correct prefix
- Rich embeds are used for formatted responses
- Error handling provides user-friendly messages

## Architecture and Structure Overview

### Core Components
```
Discord Gateway ──> Bot Handler ──> Command Router ──> NFL Client ──> External API
                           │              │                   │
                           │              └──> Models <───────┘
                           │
                           └──> Config Loader
```

### Key Architectural Patterns
- **Clean Architecture**: Separation between bot logic, business logic, and data access
- **Configuration Management**: Environment-based configuration with defaults
- **Graceful Shutdown**: Signal handling for clean bot shutdown
- **Error Handling**: Comprehensive error handling with user-friendly messages
- **Modular Design**: Separate packages for different concerns

### Main Dependencies
- `github.com/bwmarrin/discordgo v0.29.0` - Discord API client
- Go standard library for HTTP, JSON, and system operations

## Project Structure and Important Files

```
nfl-discord-bot/
├── cmd/nfl-bot/main.go         # Application entry point, handles startup and shutdown
├── internal/                   # Private application code
│   ├── bot/bot.go             # Discord bot implementation and command handling
│   ├── config/config.go       # Configuration loading and environment variable management
│   └── nfl/client.go          # NFL data client (currently using mock data)
├── pkg/models/models.go        # Data structures for players, teams, games, and stats
├── .env.example               # Environment variable template
├── go.mod                     # Go module definition and dependencies
├── go.sum                     # Dependency checksums
└── README.md                  # Project documentation and setup instructions
```

### Key Files Explained
- `cmd/nfl-bot/main.go`: Bootstrap application, load config, create bot instance, handle signals
- `internal/bot/bot.go`: Discord session management, message handling, command routing, embed creation
- `internal/config/config.go`: Environment variable parsing with validation and defaults
- `internal/nfl/client.go`: NFL data access layer (mock implementation ready for real API integration)
- `pkg/models/models.go`: Comprehensive data models with helper methods for formatting

## Tips for Development

### Adding New Commands
1. Add command handling in `bot.go`'s `messageCreate` function switch statement
2. Create a new handler function following the pattern `handleCommandName`
3. Use the existing helper functions `sendMessage` or `sendEmbed` for responses
4. Update the help command to include the new command

### NFL API Integration
- The `nfl/client.go` currently returns mock data
- **API Provider**: SportsData.io NFL API
- **Base URL**: `https://api.sportsdata.io/v3/nfl`
- **Authentication**: API key passed as query parameter `?key=YOUR_API_KEY`
- **Key Endpoints**:
  - Player Game Stats: `/stats/json/PlayerGameStatsByWeek/{season}/{week}`
  - Example: `/stats/json/PlayerGameStatsByWeek/2025REG/1?key=API_KEY`
  - Season types: `REG` (regular season weeks 1-17), `POST` (playoffs weeks 1-4)
- Replace mock implementations with actual HTTP calls to SportsData.io
- Consider rate limiting and caching for production use (API has usage limits)
- API responses should be mapped to the models in `pkg/models/`

### Common Patterns
- All command handlers follow the pattern: validate args → call service → format response → send
- Rich embeds are preferred for formatted data display
- Error messages are sent as plain text to the channel
- Configuration values are loaded once at startup

### Testing
- No test files currently exist - consider adding unit tests for command handlers
- Mock the Discord session for testing bot logic
- Test configuration loading with various environment setups

### API Testing
- Test SportsData.io API connectivity:
  ```bash
  # Test current week stats (replace with actual week number)
  curl "https://api.sportsdata.io/v3/nfl/stats/json/PlayerGameStatsByWeek/2025REG/1?key=YOUR_API_KEY"
  ```
- Verify API key is working and responses match expected model structure
- Check API documentation at: https://sportsdata.io/developers/api-documentation/nfl

### Debugging
- Increase log verbosity by setting `LOG_LEVEL=debug`
- Check `bot.log` for detailed operation logs
- Use `go run -race` to detect race conditions during development
- Monitor API usage and rate limits in SportsData.io dashboard

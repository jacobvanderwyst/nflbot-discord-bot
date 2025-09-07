package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the NFL Discord bot
type Config struct {
	// Discord settings
	DiscordToken      string
	BotPrefix         string
	CommandCooldown   time.Duration
	MaxConcurrentReqs int

	// NFL API settings
	NFLAPIKey     string
	NFLAPIBaseURL string

	// Update intervals
	StatsUpdateInterval    time.Duration
	ScheduleUpdateInterval time.Duration

	// Logging
	LogLevel string
	LogFile  string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{}

	// Discord configuration
	config.DiscordToken = os.Getenv("DISCORD_TOKEN")
	if config.DiscordToken == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN environment variable is required")
	}

	config.BotPrefix = getEnvWithDefault("BOT_PREFIX", "!")

	cooldown, err := strconv.Atoi(getEnvWithDefault("COMMAND_COOLDOWN", "3"))
	if err != nil {
		return nil, fmt.Errorf("invalid COMMAND_COOLDOWN value: %v", err)
	}
	config.CommandCooldown = time.Duration(cooldown) * time.Second

	maxReqs, err := strconv.Atoi(getEnvWithDefault("MAX_CONCURRENT_REQUESTS", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_CONCURRENT_REQUESTS value: %v", err)
	}
	config.MaxConcurrentReqs = maxReqs

	// NFL API configuration
	config.NFLAPIKey = os.Getenv("NFL_API_KEY")
	config.NFLAPIBaseURL = getEnvWithDefault("NFL_API_BASE_URL", "https://api.sportsdata.io/v3/nfl")

	// Update intervals
	statsInterval, err := strconv.Atoi(getEnvWithDefault("STATS_UPDATE_INTERVAL", "30"))
	if err != nil {
		return nil, fmt.Errorf("invalid STATS_UPDATE_INTERVAL value: %v", err)
	}
	config.StatsUpdateInterval = time.Duration(statsInterval) * time.Minute

	scheduleInterval, err := strconv.Atoi(getEnvWithDefault("SCHEDULE_UPDATE_INTERVAL", "1440"))
	if err != nil {
		return nil, fmt.Errorf("invalid SCHEDULE_UPDATE_INTERVAL value: %v", err)
	}
	config.ScheduleUpdateInterval = time.Duration(scheduleInterval) * time.Minute

	// Logging
	config.LogLevel = getEnvWithDefault("LOG_LEVEL", "info")
	config.LogFile = getEnvWithDefault("LOG_FILE", "bot.log")

	return config, nil
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

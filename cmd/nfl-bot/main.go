package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"nfl-discord-bot/internal/bot"
	"nfl-discord-bot/internal/config"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Load configuration
	cfg, err2 := config.Load()
	if err2 != nil {
		log.Fatalf("Error loading config: %v", err2)
	}

	// Create and start the bot
	discordBot, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}

	// Start the bot
	err = discordBot.Start()
	if err != nil {
		log.Fatalf("Error starting bot: %v", err)
	}

	log.Println("NFL Discord Bot is now running. Press CTRL+C to exit.")

	// Wait for interrupt signal to gracefully shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Clean up
	discordBot.Stop()
	log.Println("Bot stopped gracefully.")
}

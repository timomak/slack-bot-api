package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/slack-bot-api/config"
	"github.com/user/slack-bot-api/internal/bot"
)

func main() {
	// Set up logging
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a new bot instance
	slackBot, err := bot.New(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create bot: %v", err)
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		sig := <-sigCh
		logger.Printf("Received signal: %v, shutting down...", sig)
		cancel()
	}()

	// Start the bot
	logger.Println("Starting the Gen Alpha translation bot...")
	if err := slackBot.Start(ctx); err != nil {
		logger.Fatalf("Bot error: %v", err)
	}
} 
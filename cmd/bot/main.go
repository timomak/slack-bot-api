package main

import (
	"context"
	"log"
	"net/http"
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

	// Start a simple HTTP server for health checks and to satisfy Render's port requirements
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Gen Alpha Slack Bot is running! 🤖"))
	})
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	server := &http.Server{Addr: ":" + port}
	
	go func() {
		logger.Printf("Starting HTTP server on port %s...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Printf("HTTP server error: %v", err)
		}
	}()

	// Start the bot
	logger.Println("Starting the Gen Alpha translation bot...")
	if err := slackBot.Start(ctx); err != nil {
		logger.Fatalf("Bot error: %v", err)
	}
	
	// Shutdown the HTTP server when the bot is done
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Printf("HTTP server shutdown error: %v", err)
	}
} 
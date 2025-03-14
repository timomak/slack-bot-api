package bot

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/slack-go/slack"

	"github.com/user/slack-bot-api/config"
	"github.com/user/slack-bot-api/internal/openai"
	slackClient "github.com/user/slack-bot-api/internal/slack"
)

// Bot represents the Slack bot application
type Bot struct {
	slack  *slackClient.Client
	openai *openai.Client
	logger *log.Logger
	debug  bool
	logs   bool
	wg     sync.WaitGroup
}

// New creates a new Bot instance
func New(cfg *config.Config, logger *log.Logger) (*Bot, error) {
	// Initialize Slack client
	slack, err := slackClient.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("error initializing Slack client: %w", err)
	}

	// Initialize OpenAI client
	openai := openai.New(cfg, logger)

	if cfg.Logs {
		logger.Println("Bot initialized with configuration:")
		logger.Printf("  Debug mode: %v", cfg.Debug)
		logger.Printf("  Logs enabled: %v", cfg.Logs)
		logger.Printf("  OpenAI Model: %s", cfg.OpenAIModel)
		
		// Log detailed channel information
		logger.Println("\nConfigured Slack Channels:")
		for i, channelID := range cfg.SlackChannelIDs {
			logger.Printf("  %d. Channel ID: %s", i+1, channelID)
		}
		
		// Log detailed target user information
		logger.Println("\nConfigured Target Users:")
		for i, user := range cfg.SlackTargetUsers {
			logger.Printf("  %d. User: %s", i+1, user)
		}
	}

	return &Bot{
		slack:  slack,
		openai: openai,
		logger: logger,
		debug:  cfg.Debug,
		logs:   cfg.Logs,
	}, nil
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) error {
	if b.logs {
		b.logger.Println("Starting Gen Alpha translation bot...")
	}
	
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Track active goroutines
	b.wg.Add(1)

	// Start processing messages
	go func() {
		defer b.wg.Done()
		b.processMessages(ctx)
	}()

	if b.logs {
		b.logger.Println("Message processing routine started")
	}

	// Start the Slack client
	if err := b.slack.Start(ctx); err != nil {
		return err
	}

	// Wait for all goroutines to finish
	b.wg.Wait()
	if b.logs {
		b.logger.Println("All bot goroutines have completed")
	}
	
	return nil
}

// processMessages handles incoming Slack messages
func (b *Bot) processMessages(ctx context.Context) {
	b.logger.Println("Starting to process messages")

	// Process events from Slack
	b.slack.ProcessEvents(ctx, func(ctx context.Context, event *slack.MessageEvent) error {
		if b.logs {
			b.logger.Printf("Processing new message event - Channel: %s, User: %s", 
				event.Channel, event.User)
		}
		
		// Get user info
		user, err := b.slack.GetUserInfo(ctx, event.User)
		if err != nil {
			return fmt.Errorf("error getting user info: %w", err)
		}

		// Log the message we're about to process
		if b.logs {
			b.logger.Printf("Received message from %s (%s):", user.RealName, user.Name)
			b.logger.Printf("  Message text: %s", event.Text)
			b.logger.Printf("  Channel: %s", event.Channel)
			b.logger.Printf("  Timestamp: %s", event.Timestamp)
		} else {
			b.logger.Printf("Processing message from user %s (%s): %s", user.Name, user.ID, event.Text)
		}

		// Translate the message
		if b.logs {
			b.logger.Printf("Sending message to OpenAI for Gen Alpha translation")
		}
		
		translatedText, err := b.openai.TranslateToGenAlpha(ctx, event.Text, user.Profile.DisplayName)
		if err != nil {
			return fmt.Errorf("error translating message: %w", err)
		}

		if b.logs {
			b.logger.Printf("Received translation from OpenAI:")
			b.logger.Printf("  Original: %s", event.Text)
			b.logger.Printf("  Translated: %s", translatedText)
		}

		// Format the response
		response := fmt.Sprintf("*%s's message in Gen Alpha:*\n%s", user.Profile.DisplayName, translatedText)

		if b.logs {
			b.logger.Printf("Posting translation as channel message")
		}

		// Post the translated message directly to the channel
		_, _, err = b.slack.PostMessage(ctx, event.Channel, response)
		if err != nil {
			return fmt.Errorf("error posting message: %w", err)
		}

		if b.logs {
			b.logger.Printf("Successfully posted translation in channel %s", event.Channel)
		} else {
			b.logger.Printf("Posted translated message for %s", user.Name)
		}
		
		return nil
	})
} 
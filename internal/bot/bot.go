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

	return &Bot{
		slack:  slack,
		openai: openai,
		logger: logger,
		debug:  cfg.Debug,
	}, nil
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) error {
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

	// Start the Slack client
	if err := b.slack.Start(ctx); err != nil {
		return err
	}

	// Wait for all goroutines to finish
	b.wg.Wait()
	return nil
}

// processMessages handles incoming Slack messages
func (b *Bot) processMessages(ctx context.Context) {
	b.logger.Println("Starting to process messages")

	// Process events from Slack
	b.slack.ProcessEvents(ctx, func(ctx context.Context, event *slack.MessageEvent) error {
		// Get user info
		user, err := b.slack.GetUserInfo(ctx, event.User)
		if err != nil {
			return fmt.Errorf("error getting user info: %w", err)
		}

		// Log the message we're about to process
		b.logger.Printf("Processing message from user %s (%s): %s", user.Name, user.ID, event.Text)

		// Translate the message
		translatedText, err := b.openai.TranslateToGenAlpha(ctx, event.Text, user.RealName)
		if err != nil {
			return fmt.Errorf("error translating message: %w", err)
		}

		// Format the response
		response := fmt.Sprintf("*%s's message in Gen Alpha:*\n%s", user.RealName, translatedText)

		// Post the translated message as a thread
		_, _, err = b.slack.CreateThread(ctx, event.Channel, event.TimeStamp, response)
		if err != nil {
			return fmt.Errorf("error posting message: %w", err)
		}

		b.logger.Printf("Posted translated message for %s", user.Name)
		return nil
	})
} 
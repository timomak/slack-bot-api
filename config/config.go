package config

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Slack configuration
	SlackBotToken     string
	SlackAppToken     string
	SlackChannelIDs   []string
	SlackTargetUsers  []string
	
	// OpenAI configuration
	OpenAIAPIKey      string
	OpenAIModel       string
	OpenAIMaxTokens   int

	// App configuration
	Debug             bool
	Logs              bool
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Check for required env variables
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackBotToken == "" {
		return nil, errors.New("SLACK_BOT_TOKEN environment variable is required")
	}

	slackAppToken := os.Getenv("SLACK_APP_TOKEN")
	if slackAppToken == "" {
		return nil, errors.New("SLACK_APP_TOKEN environment variable is required")
	}

	channelIDs := os.Getenv("SLACK_CHANNEL_IDS")
	// No longer required, will monitor all channels if not specified
	// if channelIDs == "" {
	// 	return nil, errors.New("SLACK_CHANNEL_IDS environment variable is required")
	// }

	targetUsers := os.Getenv("SLACK_TARGET_USERS")
	if targetUsers == "" {
		return nil, errors.New("SLACK_TARGET_USERS environment variable is required")
	}

	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable is required")
	}

	// Set defaults for optional values
	openAIModel := os.Getenv("OPENAI_MODEL")
	if openAIModel == "" {
		openAIModel = "gpt-4"
	}

	// Debug flag
	debug := os.Getenv("DEBUG") == "true"
	
	// Logs flag
	logs := os.Getenv("LOGS") == "true"

	// Maximum tokens for OpenAI response
	openAIMaxTokens := 1024

	return &Config{
		SlackBotToken:    slackBotToken,
		SlackAppToken:    slackAppToken,
		SlackChannelIDs:  strings.Split(channelIDs, ","),
		SlackTargetUsers: strings.Split(targetUsers, ","),
		OpenAIAPIKey:     openAIKey,
		OpenAIModel:      openAIModel,
		OpenAIMaxTokens:  openAIMaxTokens,
		Debug:            debug,
		Logs:             logs,
	}, nil
} 
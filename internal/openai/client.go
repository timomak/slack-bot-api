package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/user/slack-bot-api/config"
)

// Client handles communication with the OpenAI API
type Client struct {
	apiKey    string
	model     string
	maxTokens int
	baseURL   string
	client    *http.Client
	logger    *log.Logger
}

// Message represents a single message in the OpenAI chat completion request
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents the request to the OpenAI API
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

// ChatCompletionResponse represents the response from the OpenAI API
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Index        int `json:"index"`
		Message      Message `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// New creates a new OpenAI client
func New(cfg *config.Config, logger *log.Logger) *Client {
	return &Client{
		apiKey:    cfg.OpenAIAPIKey,
		model:     cfg.OpenAIModel,
		maxTokens: cfg.OpenAIMaxTokens,
		baseURL:   "https://api.openai.com/v1/chat/completions",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// TranslateToGenAlpha translates a message to Gen Alpha slang
func (c *Client) TranslateToGenAlpha(ctx context.Context, message, username string) (string, error) {
	// Create the request to OpenAI
	prompt := fmt.Sprintf(
		"Translate the following message to Gen Alpha slang/language (TikTok style, with emojis, internet abbreviations, and current youth trends). " +
		"Make it humorous but keep the original meaning. The message is from %s: \"%s\"", 
		username, message)
	
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a Gen Alpha language translator. Your job is to translate normal messages into Gen Alpha slang and expressions. Be creative, use current youth trends, emojis, and make it funny but still understandable.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	requestBody := ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: 0.7, // Slightly creative
	}

	// Convert request to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	// Unmarshal the response
	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(body, &completionResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Check if we got any choices
	if len(completionResponse.Choices) == 0 {
		return "", fmt.Errorf("no completion choices returned from OpenAI")
	}

	// Return the translated text
	return completionResponse.Choices[0].Message.Content, nil
} 
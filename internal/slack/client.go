package slack

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/user/slack-bot-api/config"
)

// Client handles communication with the Slack API
type Client struct {
	api          *slack.Client
	socketClient *socketmode.Client
	channelIDs   map[string]bool
	targetUsers  map[string]bool
	logger       *log.Logger
	debug        bool
	logs         bool
}

// New creates a new Slack client
func New(cfg *config.Config, logger *log.Logger) (*Client, error) {
	// Initialize Slack API client
	api := slack.New(
		cfg.SlackBotToken,
		slack.OptionAppLevelToken(cfg.SlackAppToken),
		slack.OptionDebug(cfg.Debug),
	)

	// Create socket mode client
	socketClient := socketmode.New(
		api,
		socketmode.OptionDebug(cfg.Debug),
		socketmode.OptionLog(log.New(logger.Writer(), "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	// Convert channel IDs to a map for faster lookup
	channelIDs := make(map[string]bool)
	for _, id := range cfg.SlackChannelIDs {
		// Strip any whitespace
		id = strings.TrimSpace(id)
		if id != "" {
			channelIDs[id] = true
		}
	}

	if cfg.Logs {
		logger.Println("=== Slack Channel Configuration ===")
		logger.Printf("Number of monitored channels: %d", len(cfg.SlackChannelIDs))
		for i, id := range cfg.SlackChannelIDs {
			logger.Printf("  Channel #%d: %s", i+1, id)
			// Try to get channel info if possible
			if channel, err := api.GetConversationInfo(&slack.GetConversationInfoInput{ChannelID: id}); err == nil {
				logger.Printf("    Name: %s", channel.Name)
				logger.Printf("    Is Channel: %v, Is Private: %v", channel.IsChannel, channel.IsPrivate)
			}
		}
	}

	// Convert target users to a map for faster lookup
	targetUsers := make(map[string]bool)
	for _, user := range cfg.SlackTargetUsers {
		// Strip any whitespace
		user = strings.TrimSpace(user)
		if user != "" {
			targetUsers[user] = true
		}
	}

	if cfg.Logs {
		logger.Println("=== Slack User Configuration ===")
		logger.Printf("Number of target users: %d", len(cfg.SlackTargetUsers))
		for i, user := range cfg.SlackTargetUsers {
			logger.Printf("  User #%d: %s", i+1, user)
			// Try to get user info if the user ID format is detected
			if strings.HasPrefix(user, "U") && len(user) > 8 {
				if userInfo, err := api.GetUserInfo(user); err == nil {
					logger.Printf("    Name: %s", userInfo.Name)
					logger.Printf("    Real Name: %s", userInfo.RealName)
					logger.Printf("    Email: %s", userInfo.Profile.Email)
				}
			}
		}
	}

	return &Client{
		api:          api,
		socketClient: socketClient,
		channelIDs:   channelIDs,
		targetUsers:  targetUsers,
		logger:       logger,
		debug:        cfg.Debug,
		logs:         cfg.Logs,
	}, nil
}

// Start listens for Slack events
func (c *Client) Start(ctx context.Context) error {
	if c.logs {
		c.logger.Println("Starting Slack client with Socket Mode...")
	}
	
	// Run the socket mode client in a goroutine
	go func() {
		if err := c.socketClient.Run(); err != nil {
			c.logger.Printf("Error running socket mode client: %v", err)
		}
	}()

	// Run until context is canceled
	<-ctx.Done()
	c.logger.Println("Shutting down Slack client...")
	return nil
}

// ProcessEvents processes Slack events
func (c *Client) ProcessEvents(ctx context.Context, processor func(ctx context.Context, event *slack.MessageEvent) error) {
	if c.logs {
		c.logger.Println("\n===============================================")
		c.logger.Println("🤖 GEN ALPHA BOT READY TO PROCESS MESSAGES 🤖")
		c.logger.Println("===============================================")
		c.logger.Printf("Bot is monitoring %d channels for messages from %d target users", 
			len(c.channelIDs), len(c.targetUsers))
		c.logger.Println("===============================================\n")
	}
	
	for evt := range c.socketClient.Events {
		// Handle events by type
		switch evt.Type {
		case socketmode.EventTypeConnecting:
			c.logger.Println("Connecting to Slack with Socket Mode...")
		case socketmode.EventTypeConnectionError:
			c.logger.Println("Connection failed. Retrying later...")
		case socketmode.EventTypeConnected:
			c.logger.Println("Connected to Slack with Socket Mode.")
		case socketmode.EventTypeEventsAPI:
			// Acknowledge the event
			c.socketClient.Ack(*evt.Request)

			// Parse the event
			eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
			if !ok {
				c.logger.Printf("Error: Events API event expected but got %T", evt.Data)
				continue
			}

			if c.logs {
				c.logger.Printf("Event received: %s (type: %s)", eventsAPIEvent.Type, eventsAPIEvent.InnerEvent.Type)
			}

			// Handle message events
			switch eventsAPIEvent.Type {
			case "message":
				// Convert to message event
				messageEvent, ok := eventsAPIEvent.InnerEvent.Data.(*slack.MessageEvent)
				if !ok {
					c.logger.Printf("Error: Message event expected but got %T", eventsAPIEvent.InnerEvent.Data)
					continue
				}

				if c.logs {
					c.logger.Printf("Message event received from channel: %s, user: %s", 
						messageEvent.Channel, messageEvent.User)
				}

				// Process only messages from monitored channels
				if !c.channelIDs[messageEvent.Channel] {
					if c.debug || c.logs {
						c.logger.Printf("Ignoring message from non-monitored channel: %s", messageEvent.Channel)
					}
					continue
				}

				// Process only messages from target users
				user, err := c.GetUserInfo(ctx, messageEvent.User)
				if err != nil {
					c.logger.Printf("Error getting user info: %v", err)
					continue
				}

				if c.logs {
					c.logger.Printf("User info retrieved: %s (%s)", user.Name, user.ID)
				}

				if !c.targetUsers[user.Name] && !c.targetUsers[messageEvent.User] {
					if c.debug || c.logs {
						c.logger.Printf("Ignoring message from non-target user: %s (%s)", user.Name, messageEvent.User)
					}
					continue
				}

				if c.logs {
					c.logger.Printf("Processing message from target user: %s, text: %s", user.Name, messageEvent.Text)
				}

				// Skip bot messages, including our own replies to avoid loops
				if messageEvent.BotID != "" || messageEvent.SubType == "bot_message" {
					if c.debug || c.logs {
						c.logger.Println("Ignoring bot message")
					}
					continue
				}

				// Process the message
				if err := processor(ctx, messageEvent); err != nil {
					c.logger.Printf("Error processing message: %v", err)
				} else if c.logs {
					c.logger.Printf("Successfully processed message from user: %s", user.Name)
				}
			}
		}
	}
}

// GetUserInfo gets information about a Slack user
func (c *Client) GetUserInfo(ctx context.Context, userID string) (*slack.User, error) {
	if c.logs {
		c.logger.Printf("Getting user info for userID: %s", userID)
	}
	
	user, err := c.api.GetUserInfoContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting user info: %w", err)
	}
	
	if c.logs {
		c.logger.Printf("User info retrieved: %s (%s)", user.Name, user.ID)
	}
	
	return user, nil
}

// PostMessage posts a message to a Slack channel
func (c *Client) PostMessage(ctx context.Context, channelID, text string, options ...slack.MsgOption) (string, string, error) {
	if c.logs {
		c.logger.Printf("Posting message to channel: %s", channelID)
	}
	
	return c.api.PostMessageContext(ctx, channelID, append([]slack.MsgOption{slack.MsgOptionText(text, false)}, options...)...)
}

// CreateThread posts a message to a thread
func (c *Client) CreateThread(ctx context.Context, channelID, threadTS, text string) (string, string, error) {
	if c.logs {
		c.logger.Printf("Creating thread reply in channel: %s, thread: %s", channelID, threadTS)
	}
	
	channelID, threadTS, err := c.api.PostMessageContext(
		ctx,
		channelID,
		slack.MsgOptionText(text, false),
		slack.MsgOptionTS(threadTS),
	)
	
	if err == nil && c.logs {
		c.logger.Printf("Thread reply created successfully in channel: %s, thread: %s", channelID, threadTS)
	}
	
	return channelID, threadTS, err
} 
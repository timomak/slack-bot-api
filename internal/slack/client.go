package slack

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/user/slack-bot-api/config"
	"github.com/user/slack-bot-api/maps"
)

// Client handles communication with the Slack API
type Client struct {
	api          *slack.Client
	socketClient *socketmode.Client
	channelIDs   map[string]bool // Will be nil if we're monitoring all channels
	targetUsers  map[string]bool
	logger       *log.Logger
	debug        bool
	logs         bool
	monitorAllChannels bool
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

	// Check if we should monitor all channels
	monitorAllChannels := len(cfg.SlackChannelIDs) == 0 || (len(cfg.SlackChannelIDs) == 1 && cfg.SlackChannelIDs[0] == "")
	
	var channelIDs map[string]bool
	
	if !monitorAllChannels {
		// Convert channel IDs to a map for faster lookup
		channelIDs = make(map[string]bool)
		for _, id := range cfg.SlackChannelIDs {
			// Strip any whitespace
			id = strings.TrimSpace(id)
			if id != "" {
				channelIDs[id] = true
			}
		}
	}

	if cfg.Logs {
		if monitorAllChannels {
			logger.Println("=== Slack Channel Configuration ===")
			logger.Println("🔍 Bot will monitor ALL channels it has been added to")
		} else {
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
		monitorAllChannels: monitorAllChannels,
	}, nil
}

// Start listens for Slack events
func (c *Client) Start(ctx context.Context) error {
	if c.logs {
		c.logger.Println("Starting Slack client with Socket Mode...")
		
		// Only run setup verification when logs are enabled
		if err := c.VerifySetup(ctx); err != nil {
			c.logger.Printf("WARNING: Setup verification found issues: %v", err)
		}
	} else {
		// Simple startup message when logs are disabled
		c.logger.Println("Starting Slack client...")
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

// VerifySetup checks that everything is correctly configured
func (c *Client) VerifySetup(ctx context.Context) error {
	c.logger.Println("Verifying Slack bot setup...")
	
	// Check authentication
	authTest, err := c.api.AuthTestContext(ctx)
	if err != nil {
		return fmt.Errorf("authentication test failed: %w", err)
	}
	
	c.logger.Printf("✅ Connected as: %s (UserID: %s, TeamName: %s)", 
		authTest.User, authTest.UserID, authTest.Team)
	
	// Check each channel
	c.logger.Println("Verifying channel access...")
	channelErrors := false

	if c.monitorAllChannels {
		c.logger.Println("🔍 Bot is configured to monitor ALL channels it has been added to")
		
		// Get all conversations the bot is a member of
		channels, nextCursor, err := c.api.GetConversationsForUserContext(ctx, &slack.GetConversationsForUserParameters{
			Types: []string{"public_channel", "private_channel"},
			Limit: 100,
		})
		
		if err != nil {
			c.logger.Printf("❌ Error fetching channels: %v", err)
			channelErrors = true
		} else {
			if len(channels) == 0 {
				c.logger.Println("⚠️ Bot is not a member of any channels. Please add the bot to channels using /invite @BotName")
				channelErrors = true
			} else {
				c.logger.Printf("✅ Bot is a member of %d channels:", len(channels))
				for _, channel := range channels {
					c.logger.Printf("   - %s (%s)", channel.Name, channel.ID)
				}
				
				if nextCursor != "" {
					c.logger.Println("⚠️ Bot is in more than 100 channels. Only showing the first 100.")
				}
			}
		}
	} else {
		for channelID := range c.channelIDs {
			channelInfo, err := c.api.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
				ChannelID: channelID,
			})
			
			if err != nil {
				c.logger.Printf("❌ Channel access error for %s: %v", channelID, err)
				channelErrors = true
				continue
			}
			
			// Check if bot is a member of the channel
			members, _, err := c.api.GetUsersInConversationContext(ctx, &slack.GetUsersInConversationParameters{
				ChannelID: channelID,
			})
			
			if err != nil {
				c.logger.Printf("❌ Cannot verify membership for channel %s (%s): %v", 
					channelInfo.Name, channelID, err)
				channelErrors = true
				continue
			}
			
			botInChannel := false
			for _, memberID := range members {
				if memberID == authTest.UserID {
					botInChannel = true
					break
				}
			}
			
			if !botInChannel {
				c.logger.Printf("❌ Bot is NOT a member of channel %s (%s). Please add the bot using /invite @%s", 
					channelInfo.Name, channelID, authTest.User)
				channelErrors = true
				continue
			}
			
			c.logger.Printf("✅ Channel verified: %s (%s)", channelInfo.Name, channelID)
		}
	}
	
	// Check user access
	c.logger.Println("Verifying user access...")
	userErrors := false
	
	for targetUser := range c.targetUsers {
		// Skip IDs that look like user IDs as they don't need username verification
		if strings.HasPrefix(targetUser, "U") && len(targetUser) > 8 {
			user, err := c.api.GetUserInfoContext(ctx, targetUser)
			if err != nil {
				c.logger.Printf("❌ Cannot get info for user ID %s: %v", targetUser, err)
				userErrors = true
			} else {
				c.logger.Printf("✅ User ID verified: %s (%s)", user.Name, targetUser)
			}
			continue
		}
		
		// Try to find user by username
		users, err := c.api.GetUsersContext(ctx)
		if err != nil {
			c.logger.Printf("❌ Cannot retrieve users list: %v", err)
			userErrors = true
			continue
		}
		
		foundUser := false
		for _, user := range users {
			if user.Name == targetUser {
				foundUser = true
				c.logger.Printf("✅ Username verified: %s (%s)", user.Name, user.ID)
				break
			}
		}
		
		if !foundUser {
			c.logger.Printf("❌ Username '%s' not found in workspace. Check for typos or use the user ID instead.", 
				targetUser)
			userErrors = true
		}
	}
	
	// Test if we can listen for events
	c.logger.Println("Checking event subscriptions...")
	c.logger.Println("⚠️ To verify event reception, please send a test message in one of the monitored channels.")
	
	// Send a test message to verify if Slack events are set up properly
	c.testEventSubscription(ctx)

	if channelErrors || userErrors {
		return fmt.Errorf("setup verification found issues with channels and/or users")
	}
	
	c.logger.Println("✅ Slack setup verification completed successfully!")
	return nil
}

// testEventSubscription sends a test message to verify event subscriptions
func (c *Client) testEventSubscription(ctx context.Context) {
	// For all-channels mode, we need to find a channel to test
	if c.monitorAllChannels {
		c.logger.Println("🔍 Finding a channel to send test message...")
		
		// Get channels the bot is a member of
		channels, _, err := c.api.GetConversationsForUserContext(ctx, &slack.GetConversationsForUserParameters{
			Types: []string{"public_channel", "private_channel"},
			Limit: 1,
		})
		
		if err != nil {
			c.logger.Printf("❌ Error fetching channels for test: %v", err)
			c.logger.Println("⚠️ Skipping event subscription test")
			return
		}
		
		if len(channels) == 0 {
			c.logger.Println("⚠️ Bot is not a member of any channels. Please add the bot to channels using /invite @BotName")
			c.logger.Println("⚠️ Skipping event subscription test")
			return
		}
		
		// Skip sending test message if DEBUG mode is not enabled
		if !c.debug {
			c.logger.Println("ℹ️ Skipping self-test message (enable DEBUG=true to send test messages)")
			c.logger.Println("⚠️ If you're not receiving events, check your Event Subscriptions in Slack API settings")
			return
		}
		
		// Use the first channel we find
		channelID := channels[0].ID
		c.logger.Printf("🧪 Sending a self-test message to channel %s (%s) to verify event subscriptions...", 
			channels[0].Name, channelID)
		
		// Create a unique message so we can identify it
		testMsg := fmt.Sprintf("🔍 Bot self-test message (timestamp: %s) - If you see this message but no events are logged, check your Event Subscriptions in Slack API", 
			time.Now().Format(time.RFC3339))
		
		// Send the message
		_, _, err = c.api.PostMessageContext(
			ctx,
			channelID,
			slack.MsgOptionText(testMsg, false),
		)
		
		if err != nil {
			c.logger.Printf("❌ Failed to send test message: %v", err)
			c.logger.Println("⚠️ This may indicate the bot lacks permissions to post in this channel")
			return
		}
		
		c.logger.Println("✅ Test message sent successfully")
		c.logger.Println("⚠️ If you don't see any event logs after this, your Slack app's Event Subscriptions may not be set up correctly")
		c.logger.Println("⚠️ Check that Socket Mode is enabled AND you've subscribed to message events in your Slack app settings")
		return
	}
	
	// Only try to send a test message if we have at least one channel
	if len(c.channelIDs) == 0 {
		c.logger.Println("⚠️ No channels configured, skipping event subscription test")
		return
	}
	
	// Skip sending test message if DEBUG mode is not enabled
	if !c.debug {
		c.logger.Println("ℹ️ Skipping self-test message (enable DEBUG=true to send test messages)")
		c.logger.Println("⚠️ If you're not receiving events, check your Event Subscriptions in Slack API settings")
		return
	}
	
	// Get the first channel ID
	var channelID string
	for id := range c.channelIDs {
		channelID = id
		break
	}
	
	c.logger.Printf("🧪 Sending a self-test message to channel %s to verify event subscriptions...", channelID)
	
	// Create a unique message so we can identify it
	testMsg := fmt.Sprintf("🔍 Bot self-test message (timestamp: %s) - If you see this message but no events are logged, check your Event Subscriptions in Slack API", 
		time.Now().Format(time.RFC3339))
	
	// Send the message
	_, _, err := c.api.PostMessageContext(
		ctx,
		channelID,
		slack.MsgOptionText(testMsg, false),
	)
	
	if err != nil {
		c.logger.Printf("❌ Failed to send test message: %v", err)
		c.logger.Println("⚠️ This may indicate the bot lacks permissions to post in this channel")
		return
	}
	
	c.logger.Println("✅ Test message sent successfully")
	c.logger.Println("⚠️ If you don't see any event logs after this, your Slack app's Event Subscriptions may not be set up correctly")
	c.logger.Println("⚠️ Check that Socket Mode is enabled AND you've subscribed to message events in your Slack app settings")
}

// ProcessEvents processes Slack events
func (c *Client) ProcessEvents(ctx context.Context, processor func(ctx context.Context, event *slack.MessageEvent) error) {
	if c.logs {
		c.logger.Println("\n===============================================")
		c.logger.Println("🤖 GEN ALPHA BOT READY TO PROCESS MESSAGES 🤖")
		c.logger.Println("===============================================")
		c.logger.Printf("Bot is monitoring %d channels for messages from %d target users", 
			len(c.channelIDs), len(c.targetUsers))
		c.logger.Println("Channels monitored:", strings.Join(maps.Keys(c.channelIDs), ", "))
		c.logger.Println("Target users:", strings.Join(maps.Keys(c.targetUsers), ", "))
		c.logger.Println("===============================================\n")
		c.logger.Println("⚠️ WAITING FOR EVENTS - If no events appear below when you send messages, check your Slack app configuration")
	}
	
	// Create a ticker to log periodic heartbeats
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	go func() {
		for {
			select {
			case <-ticker.C:
				c.logger.Println("❤️ Bot is still alive and listening for events...")
			case <-ctx.Done():
				return
			}
		}
	}()
	
	for evt := range c.socketClient.Events {
		// Debug log for ALL events received from Slack
		c.logger.Printf("🔍 DEBUG - Received event from Slack: Type=%s", evt.Type)
		
		// Handle events by type
		switch evt.Type {
		case socketmode.EventTypeConnecting:
			c.logger.Println("Connecting to Slack with Socket Mode...")
		case socketmode.EventTypeConnectionError:
			c.logger.Println("Connection failed. Retrying later...")
		case socketmode.EventTypeConnected:
			c.logger.Println("Connected to Slack with Socket Mode.")
		case socketmode.EventTypeHello:
			c.logger.Println("🎉 Received Hello from Slack - connection fully established")
		case socketmode.EventTypeDisconnect:
			c.logger.Println("⚠️ Disconnected from Slack")
		case socketmode.EventTypeEventsAPI:
			// Acknowledge the event immediately
			c.socketClient.Ack(*evt.Request)

			// Log raw event for troubleshooting
			c.logger.Printf("📨 Received event from Slack Events API: %+v", evt)

			// Parse the event
			eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
			if !ok {
				c.logger.Printf("❌ Error: Events API event expected but got %T", evt.Data)
				continue
			}

			// Log the complete event structure
			c.logger.Printf("📨 Event details - Type: %s, InnerEvent Type: %s", 
				eventsAPIEvent.Type, eventsAPIEvent.InnerEvent.Type)

			// Handle message events
			if eventsAPIEvent.Type == slackevents.CallbackEvent {
				innerEvent := eventsAPIEvent.InnerEvent
				
				// Log inner event type for troubleshooting
				c.logger.Printf("🔍 Inner event type: %s", innerEvent.Type)
				
				// Check for message type
				if innerEvent.Type == string(slackevents.Message) {
					// First, get the event as a slackevents.MessageEvent
					slackEventsMessageEvent, ok := innerEvent.Data.(*slackevents.MessageEvent)
					if !ok {
						c.logger.Printf("❌ Error: slackevents.MessageEvent expected but got %T", innerEvent.Data)
						continue
					}
					
					// Create a compatible MessageEvent structure
					// Using only the fields we need for our application to avoid field name mismatches
					messageEvent := &slack.MessageEvent{
						Msg: slack.Msg{
							Channel:   slackEventsMessageEvent.Channel,
							User:      slackEventsMessageEvent.User,
							Text:      slackEventsMessageEvent.Text,
							Timestamp: slackEventsMessageEvent.TimeStamp,
							ThreadTimestamp: slackEventsMessageEvent.ThreadTimeStamp,
							BotID:     slackEventsMessageEvent.BotID,
							SubType:   slackEventsMessageEvent.SubType,
						},
					}

					c.logger.Printf("📝 Message received - Channel: %s, User: %s, Text: %s", 
						messageEvent.Channel, messageEvent.User, messageEvent.Text)

					// Skip bot messages, including our own replies to avoid loops
					if messageEvent.BotID != "" || messageEvent.SubType == "bot_message" {
						c.logger.Printf("⏩ Ignoring bot message from: %s", messageEvent.BotID)
						continue
					}

					// Debug all channel IDs
					c.logger.Printf("🔍 Checking channel access - Message channel: %s, Monitored channels: %v", 
						messageEvent.Channel, c.channelIDs)
						
					// Process only messages from monitored channels if we're not monitoring all channels
					if !c.monitorAllChannels && !c.channelIDs[messageEvent.Channel] {
						c.logger.Printf("⏩ Ignoring message from non-monitored channel: %s", messageEvent.Channel)
						continue
					}

					if c.monitorAllChannels {
						c.logger.Printf("✅ Processing message from channel: %s (monitoring all channels)", messageEvent.Channel)
					} else {
						c.logger.Printf("✅ Channel match found: %s", messageEvent.Channel)
					}

					// Process only messages from target users
					user, err := c.GetUserInfo(ctx, messageEvent.User)
					if err != nil {
						c.logger.Printf("❌ Error getting user info: %v", err)
						continue
					}

					c.logger.Printf("👤 User info retrieved: %s (%s)", user.Name, user.ID)

					// Debug all target users
					c.logger.Printf("🔍 Checking user match - Message user: %s (%s), Target users: %v", 
						user.Name, messageEvent.User, c.targetUsers)
						
					if !c.targetUsers[user.Name] && !c.targetUsers[messageEvent.User] {
						c.logger.Printf("⏩ Ignoring message from non-target user: %s (%s)", user.Name, messageEvent.User)
						continue
					}

					c.logger.Printf("✅ User match found: %s", user.Name)
					c.logger.Printf("🎯 Processing message: '%s'", messageEvent.Text)

					// Process the message
					if err := processor(ctx, messageEvent); err != nil {
						c.logger.Printf("❌ Error processing message: %v", err)
					} else {
						c.logger.Printf("✅ Successfully processed message from user: %s", user.Name)
					}
				} else {
					c.logger.Printf("ℹ️ Received non-message event type: %s", innerEvent.Type)
				}
			} else {
				c.logger.Printf("ℹ️ Received non-callback event type: %s", eventsAPIEvent.Type)
			}
		default:
			c.logger.Printf("ℹ️ Received unhandled event type: %s", evt.Type)
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
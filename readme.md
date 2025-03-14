# Gen Alpha Slack Bot

A Slack bot that listens to messages from specific users in designated channels and "translates" them into Gen Alpha slang using ChatGPT, adding a humorous spin while preserving the original meaning.

## Features

- 🤖 Monitors specific Slack channels for messages
- 👥 Targets only messages from designated users
- 🔄 Translates messages to Gen Alpha slang using ChatGPT/OpenAI
- 🧵 Posts translations as threaded replies to original messages
- 🛠️ Easily configurable through environment variables
- 🐳 Docker support for containerized deployment

## Prerequisites

- Go 1.18 or later
- Slack API credentials (Bot Token and App Token)
- OpenAI API key
- Docker (optional, for containerized deployment)

## Setup

### 1. Create a Slack App

1. Go to [Slack API Apps](https://api.slack.com/apps) and click "Create New App"
2. Choose "From scratch"
3. Name your app and select your workspace

#### Enable Socket Mode

4. Navigate to "Socket Mode" in the sidebar and enable it
5. Create an app-level token with the `connections:write` scope (save this token as your `SLACK_APP_TOKEN`)

#### Configure OAuth & Permissions

6. Under "OAuth & Permissions", add the following Bot Token Scopes:
   - `channels:history` - to read messages in public channels
   - `channels:read` - to get information about public channels
   - `groups:history` - to read messages in private channels
   - `groups:read` - to get information about private channels
   - `chat:write` - to post messages
   - `users:read` - to get information about users

   **Note:** If you plan to monitor direct messages or group DMs, also add:
   - `im:history` - for direct messages
   - `mpim:history` - for group direct messages

7. Install the app to your workspace (save the Bot User OAuth Token as your `SLACK_BOT_TOKEN`)
   
   **Important:** If you add scopes after initially installing the app, you'll need to reinstall the app for the new scopes to take effect.

#### Set Up Event Subscriptions

8. Under "Event Subscriptions", toggle "Enable Events" to On
9. Subscribe to the following bot events:
   - `message.channels` - to receive messages from public channels
   - `message.groups` - to receive messages from private channels
   - `message.im` - to receive direct messages (if needed)
   - `message.mpim` - to receive group direct messages (if needed)

10. Save your changes

#### Add Bot to Channels

11. Invite your bot to the channels you want it to monitor by typing `/invite @YourBotName` in each channel
    - This step is **required** for both public and private channels
    - The bot can only monitor channels it has been invited to

### 2. Set Up Environment Variables

Copy the example environment file and fill in your credentials:

```bash
cp .env.example .env
```

Edit the `.env` file with your:
- Slack Bot Token (starts with `xoxb-`)
- Slack App Token (starts with `xapp-`)
- Channel IDs to monitor - Get these by right-clicking on channels in Slack and selecting "Copy Link" (the ID is the part after the last slash)
- Target usernames or user IDs - Use exact usernames (case-sensitive) or user IDs (starts with U...)
- OpenAI API key

Example `.env` configuration:
```
SLACK_BOT_TOKEN=xoxb-your-token-here
SLACK_APP_TOKEN=xapp-your-token-here
SLACK_CHANNEL_IDS=C12345678,C87654321
SLACK_TARGET_USERS=john,jane,U12345678
OPENAI_API_KEY=sk-your-key-here
OPENAI_MODEL=gpt-4
DEBUG=false
LOGS=true
```

#### Channel Monitoring Configuration

You have two options for channel monitoring:

1. **Monitor specific channels**: Set `SLACK_CHANNEL_IDS` to a comma-separated list of channel IDs
2. **Monitor all channels**: Leave `SLACK_CHANNEL_IDS` empty or remove it from the `.env` file

When `SLACK_CHANNEL_IDS` is not specified, the bot will automatically monitor all channels it has been added to.

### 3. Install and Run

#### Using Go

```bash
# Download dependencies
go mod download

# Build the application
go build -o slack-bot-api ./cmd/bot

# Run the application
./slack-bot-api
```

#### Using Docker

```bash
# Build the Docker image
docker build -t gen-alpha-slack-bot .

# Run the container
docker run --env-file .env gen-alpha-slack-bot
```

## Troubleshooting

If your bot isn't responding to messages, check the following common issues:

### 1. Missing OAuth Scopes

If you see `Channel access error: missing_scope` in the logs:
- Go to your [Slack App's configuration page](https://api.slack.com/apps)
- Verify you've added all the required scopes listed in the setup instructions
- Pay attention to channel types:
  - Public channels need `channels:history` and `channels:read`
  - Private channels need `groups:history` and `groups:read`
- After adding new scopes, reinstall the app to your workspace

### Critical Issue: Not Receiving ANY Events

If your bot seems to connect to Slack but doesn't show any events at all when messages are sent (no logs appear), this indicates a fundamental issue with event subscriptions:

1. **Verify Event Subscriptions in Slack App**:
   - Go to your [Slack App](https://api.slack.com/apps)
   - Select your app
   - Click on "Event Subscriptions" in the sidebar
   - Make sure "Enable Events" is toggled ON
   - Under "Subscribe to bot events", verify you have:
     - `message.channels` 
     - `message.groups` (if you're using private channels)

2. **Verify Socket Mode**:
   - In the Slack App UI, click "Socket Mode" in the sidebar
   - Ensure "Enable Socket Mode" is toggled ON
   - Verify you have an App-Level Token with the `connections:write` scope

3. **Test with Debug Mode**:
   - Set both `DEBUG=true` and `LOGS=true` in your `.env` file
   - Run the bot
   - Look for heart-beat logs ("Bot is still alive and listening for events...")
   - Try sending a message in the monitored channel as a monitored user
   - If you see NO response to your message at all, it confirms event subscription issues

4. **Subscription Activation Delay**:
   - Sometimes Slack can take a few minutes to activate event subscriptions
   - Try restarting your bot and wait 5-10 minutes

5. **Reinstall App Completely**:
   - Remove the app from your workspace
   - Delete the app entirely from your Slack Apps list
   - Create a new app with the same name and required scopes
   - Reinstall to your workspace
   - Rebuild and restart the bot

6. **Network Configuration**:
   - Ensure your network allows outbound WebSocket connections
   - No firewall or proxy is blocking connections to Slack's APIs

### 2. Bot Not in Channels

- The bot must be explicitly invited to each channel it will monitor
- Use `/invite @YourBotName` in each channel
- Verify the bot is a member by checking the member list in the channel

### 3. User Configuration Issues

- Usernames in `SLACK_TARGET_USERS` are case-sensitive and must match exactly
- If a username fails verification, try using the user ID instead (starts with U...)
- Get user IDs from your logs or from the Slack profile (click on profile picture → "Copy member ID")

### 4. Channel ID Verification

- Double-check your channel IDs are correct
- Get the correct ID by right-clicking on the channel name → "Copy Link"
- The ID is the part after the last slash in the URL and starts with C...

### 5. Event Subscription Issues

- Make sure you've enabled Event Subscriptions in your Slack app
- Subscribe to the correct event types based on channel types:
  - `message.channels` for public channels
  - `message.groups` for private channels

### 6. Debug Mode

Turn on debug mode and detailed logs to see what's happening:
```
DEBUG=true
LOGS=true
```

When debug mode is enabled:
- The bot will send a self-test message to the first monitored channel at startup
- You'll see more detailed logs about event processing
- Heartbeat logs will appear every minute to confirm the bot is still running

This is particularly useful for diagnosing event subscription issues, as the self-test message helps verify if:
1. The bot can send messages to the channel
2. The bot can receive the event for its own message
3. Socket Mode is working correctly

### 7. Complete Reset

If all else fails, try:
1. Reinstalling the Slack app with all the required scopes
2. Reinviting the bot to all monitored channels
3. Restarting the bot application

## Configuration Options

| Environment Variable | Description | Required | Default |
|----------------------|-------------|----------|---------|
| `SLACK_BOT_TOKEN` | Slack Bot token starting with `xoxb-` | Yes | - |
| `SLACK_APP_TOKEN` | Slack App token starting with `xapp-` | Yes | - |
| `SLACK_CHANNEL_IDS` | Comma-separated list of channel IDs to monitor (if empty, monitors all channels the bot is in) | No | - |
| `SLACK_TARGET_USERS` | Comma-separated list of usernames or user IDs | Yes | - |
| `OPENAI_API_KEY` | OpenAI API key | Yes | - |
| `OPENAI_MODEL` | OpenAI model to use | No | `gpt-4` |
| `DEBUG` | Enable debug logging and self-test messages | No | `false` |
| `LOGS` | Enable detailed logging and setup verification | No | `false` |

### Config Behavior Details

- **DEBUG=true**: Enables diagnostic messages and sends self-test messages at startup
- **LOGS=true**: Enables:
  - Detailed logging throughout the application
  - Startup verification of channels and users
  - Channel and user info display
  - Heartbeat logs (every 60 seconds)
  
For normal operation, you can disable both. For troubleshooting, enabling both provides the most information.

## Deployment

For production deployment, you can:

1. Build the Docker container and deploy to your container orchestration platform
2. Use a service like AWS ECS, GCP Cloud Run, or Kubernetes
3. Set up proper monitoring and logging

Example Docker Compose file for simple deployment:

```yaml
version: '3'
services:
  slack-bot:
    build: .
    restart: always
    env_file:
      - .env
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### Deploying to Render.com

To deploy this bot to Render.com:

1. Push your code to a GitHub repository
2. Log in to your Render account
3. Click "New" and select "Web Service"
4. Connect your GitHub repository
5. Configure the service:
   - **Name**: Choose a name for your service
   - **Environment**: Select "Go"
   - **Region**: Choose the region closest to you
   - **Branch**: Select your default branch
   - **Build Command**: `go build -o app ./cmd/bot`
   - **Start Command**: `./app`

6. Under "Environment Variables", add all required environment variables:
   - `SLACK_BOT_TOKEN`
   - `SLACK_APP_TOKEN`
   - `SLACK_TARGET_USERS`
   - `OPENAI_API_KEY`
   - And any optional variables you want to use (`OPENAI_MODEL`, `DEBUG`, `LOGS`)

7. Click "Create Web Service"

Render will automatically detect the PORT environment variable and route traffic to your service.

## Security Considerations

- Never commit your `.env` file with actual credentials
- Consider using a secrets manager for production deployments
- Rotate your API keys periodically
- Keep your dependencies updated

## How It Works

1. The bot connects to Slack using Socket Mode
2. It listens for messages in all channels it has been added to (or specific configured channels)
3. When a message from a target user is detected, it's sent to OpenAI for "translation"
4. The translated version is posted directly in the channel

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
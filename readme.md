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
4. Under "OAuth & Permissions", add the following Bot Token Scopes:
   - `channels:history`
   - `channels:read`
   - `chat:write`
   - `users:read`
5. Install the app to your workspace
6. Under "Basic Information", look for "App-Level Tokens" and create a new token with the `connections:write` scope
7. Enable Socket Mode in the "Socket Mode" section

### 2. Set Up Environment Variables

Copy the example environment file and fill in your credentials:

```bash
cp .env.example .env
```

Edit the `.env` file with your:
- Slack Bot Token (starts with `xoxb-`)
- Slack App Token (starts with `xapp-`)
- Channel IDs to monitor
- Target usernames or user IDs
- OpenAI API key

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

## Configuration Options

| Environment Variable | Description | Required | Default |
|----------------------|-------------|----------|---------|
| `SLACK_BOT_TOKEN` | Slack Bot token starting with `xoxb-` | Yes | - |
| `SLACK_APP_TOKEN` | Slack App token starting with `xapp-` | Yes | - |
| `SLACK_CHANNEL_IDS` | Comma-separated list of channel IDs to monitor | Yes | - |
| `SLACK_TARGET_USERS` | Comma-separated list of usernames or user IDs | Yes | - |
| `OPENAI_API_KEY` | OpenAI API key | Yes | - |
| `OPENAI_MODEL` | OpenAI model to use | No | `gpt-4` |
| `DEBUG` | Enable debug logging | No | `false` |

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

## Security Considerations

- Never commit your `.env` file with actual credentials
- Consider using a secrets manager for production deployments
- Rotate your API keys periodically
- Keep your dependencies updated

## How It Works

1. The bot connects to Slack using Socket Mode
2. It listens for messages in the configured channels
3. When a message from a target user is detected, it's sent to OpenAI for "translation"
4. The translated version is posted as a thread reply to the original message

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

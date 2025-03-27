# Slack Kudos Bot

A Slack bot that helps team members recognize each other by giving kudos. When someone mentions a user with a "++" suffix (e.g., "@user ++"), the bot records the kudos. Users can view the top kudos recipients using the "/kudos" slash command.

## Features

- Track and manage kudos given to users in Slack workspaces
- Maintain a leaderboard of kudos recipients
- Support for multiple Slack workspaces
- Easy installation via OAuth 2.0 flow

## Running the Application

### Build the Binary

```bash
go build -o kudosbot
```

### Configuration

Export the following environment variables:

```bash
# Required configuration
export KUDOS_SLACK_CLIENT_ID='your_client_id'
export KUDOS_SLACK_CLIENT_SECRET='your_client_secret'
export KUDOS_SLACK_APP_TOKEN='xapp-...'  # App-level token for Socket Mode

# Optional configuration
export KUDOS_SQLITE_FILENAME='kudos.db'  # Default: kudos.db
export KUDOS_SERVER_PORT='8080'          # Default: 8080
export KUDOS_BASE_URL='https://your-domain.com'  # Default: http://localhost:8080
export KUDOS_DEBUG='true'                # Enable debug mode with HTTPS self-signed cert
```

### Debug Mode Notes

When running with `KUDOS_DEBUG=true`:

1. The application automatically generates a self-signed certificate
2. HTTPS is used instead of HTTP to meet Slack's OAuth requirements
3. You must manually accept the self-signed certificate in your browser:
   - Open the app URL (e.g., https://localhost:8080) in your browser
   - Click 'Advanced' and then 'Proceed anyway' to accept the certificate
   - Then try the Slack OAuth flow again

This is only required for local development. In production, you should run behind a reverse proxy that handles HTTPS termination.

The redirect URI is automatically set to `$KUDOS_BASE_URL/oauth/callback` unless overridden by:

```bash
export KUDOS_SLACK_REDIRECT_URI='https://your-domain.com/oauth/callback'
```

### Run the Binary

```bash
./kudosbot
```

## Publishing Your Slack App

### 1. Create Your Slack App

1. Go to [Slack API](https://api.slack.com/apps) and click "Create New App"
2. Choose "From scratch" and give your app a name
3. Select the workspace where you'll develop your app

### 2. Configure Basic Information

1. Under "Basic Information", customize your app's name, description, and icon
2. Note your Client ID and Client Secret for your environment variables

### 3. Set Up Bot Permissions

1. Go to "OAuth & Permissions" in the sidebar
2. Under "Scopes" > "Bot Token Scopes", add the following permissions:
   - `channels:history`
   - `channels:read`
   - `chat:write`
   - `commands`
   - `groups:history`
   - `im:history`
   - `users:read`

### 4. Configure Socket Mode

1. Go to "Socket Mode" in the sidebar and enable it
2. Generate an app-level token with the `connections:write` scope:
   - In "Basic Information" > "App-Level Tokens" click "Generate Token and Scopes"
   - Add the `connections:write` scope
   - Give your token a name (e.g., "socket-mode")
   - Copy the generated token (starts with `xapp-`) and set it as `KUDOS_SLACK_APP_TOKEN`
3. This token is used by your server for Socket Mode connections to Slack


### 5. Configure Slash Commands

1. Go to "Slash Commands" in the sidebar
2. Create a new command called `/kudos`
3. Set the Request URL to your server's endpoint (during development, this can be a placeholder)
4. Add a description: "View kudos leaderboard"

### 6. Configure Event Subscriptions

1. Go to "Event Subscriptions" in the sidebar
2. Enable events
3. Subscribe to bot events:
   - `message.channels`
   - `message.groups`
   - `message.im`

### 7. Configure OAuth & Distribution

1. Go to "OAuth & Permissions" in the sidebar
2. Add your Redirect URL (e.g., `https://your-domain.com/oauth/callback`)
3. Go to "Manage Distribution" in the sidebar
4. Enable public distribution
5. Fill out the required information:
   - App description
   - Application website
   - Verification info for Slack's review
   - Terms of service and privacy policy URLs

### 9. Submit for Review

1. Complete the Slack App Submission Checklist
2. Submit your app for review by Slack
3. Once approved, your app will be available in the Slack App Directory

### 10. Deploy Your Application

1. Deploy your application to a server with a public IP address
2. Ensure your server is accessible via HTTPS
3. Configure your environment variables with your Client ID, Client Secret, and Redirect URI
4. Start your application

## Installation

After your app is published:

1. Users can install the app from the Slack App Directory or your landing page
2. They'll click the "Install on Slack" button
3. They'll authorize the permissions
4. The bot will be added to their workspace
5. They should add the bot to channels where they want to use it

## Usage

After installation, you need to:

1. **Invite the bot to channels** where you want it to work:
   - In Slack, go to the channel where you want to use Kudos
   - Type `/invite @kudos-bot` (replace with your actual bot name)
   - The bot needs to be in a channel to detect kudos mentions and respond to commands
   
2. **Using the bot**:
   - To give kudos: mention a user followed by `++` (e.g., `@user ++`)
   - To view the kudos leaderboard: use the `/kudos` slash command
   - By default, the leaderboard shows the top 5 users

If you see an error like "The app is not in this channel" or "Cannot find app" when using commands, you need to invite the bot to the channel first.
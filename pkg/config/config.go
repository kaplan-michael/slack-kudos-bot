package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	SQLiteFilename    string
	SlackClientID     string
	SlackClientSecret string
	SlackAppToken     string // App-level token for Socket Mode
	SlackRedirectURI  string
	ServerPort        int
	Debug             bool
	BaseURL           string // Base URL where the application is running
}

var AppConfig = &Config{}

func Init() {
	var missingVars []string

	// Database config
	AppConfig.SQLiteFilename = os.Getenv("KUDOS_SQLITE_FILENAME")
	if AppConfig.SQLiteFilename == "" {
		AppConfig.SQLiteFilename = "kudos.db"
	}

	// OAuth2 configuration (required)
	AppConfig.SlackClientID = os.Getenv("KUDOS_SLACK_CLIENT_ID")
	if AppConfig.SlackClientID == "" {
		missingVars = append(missingVars, "KUDOS_SLACK_CLIENT_ID")
	}

	AppConfig.SlackClientSecret = os.Getenv("KUDOS_SLACK_CLIENT_SECRET")
	if AppConfig.SlackClientSecret == "" {
		missingVars = append(missingVars, "KUDOS_SLACK_CLIENT_SECRET")
	}
	
	// App-level token for Socket Mode
	AppConfig.SlackAppToken = os.Getenv("KUDOS_SLACK_APP_TOKEN")
	if AppConfig.SlackAppToken == "" {
		missingVars = append(missingVars, "KUDOS_SLACK_APP_TOKEN")
	}

	// Base URL where the application is running
	AppConfig.BaseURL = os.Getenv("KUDOS_BASE_URL")
	if AppConfig.BaseURL == "" {
		AppConfig.BaseURL = "http://localhost:8080"
	}
	
	// Slack redirect URI
	AppConfig.SlackRedirectURI = os.Getenv("KUDOS_SLACK_REDIRECT_URI")
	if AppConfig.SlackRedirectURI == "" {
		// Build the redirect URI from the base URL
		AppConfig.SlackRedirectURI = AppConfig.BaseURL + "/oauth/callback"
	}

	// Server port for HTTP server
	portStr := os.Getenv("KUDOS_SERVER_PORT")
	if portStr == "" {
		AppConfig.ServerPort = 8080
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Printf("Invalid server port %s, using default 8080", portStr)
			AppConfig.ServerPort = 8080
		} else {
			AppConfig.ServerPort = port
		}
	}
	
	// Debug mode
	debugEnv := os.Getenv("KUDOS_DEBUG")
	AppConfig.Debug = debugEnv == "true" || debugEnv == "1" || debugEnv == "yes"

	if len(missingVars) > 0 {
		log.Fatalf("Missing required environment variables: %v", missingVars)
	}
}
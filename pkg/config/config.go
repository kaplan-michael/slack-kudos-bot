package config

import (
	"log"
	"os"
)

type Config struct {
	SQLiteFilename string
	SlackToken     string
	SlackAppToken  string
}

var AppConfig = &Config{}

func Init() {
	var missingVars []string

	AppConfig.SQLiteFilename = os.Getenv("KUDOS_SQLITE_FILENAME")
	if AppConfig.SQLiteFilename == "" {
		AppConfig.SQLiteFilename = "kudos.db"
	}

	AppConfig.SlackToken = os.Getenv("KUDOS_SLACK_BOT_TOKEN")
	if AppConfig.SlackToken == "" {
		missingVars = append(missingVars, "KUDOS_SLACK_BOT_TOKEN")
	}

	AppConfig.SlackAppToken = os.Getenv("KUDOS_SLACK_APP_TOKEN")
	if AppConfig.SlackAppToken == "" {
		missingVars = append(missingVars, "KUDOS_SLACK_APP_TOKEN")
	}

	if len(missingVars) > 0 {
		log.Fatalf("Missing required environment variables: %v", missingVars)
	}
}

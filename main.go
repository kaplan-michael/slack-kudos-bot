package main

import (
	"github.com/charmbracelet/log"
	"github.com/kaplan-michael/slack-kudos/pkg/config"
	"github.com/kaplan-michael/slack-kudos/pkg/database"
	"github.com/kaplan-michael/slack-kudos/pkg/dispatcher"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	l "log"
	"os"
)

func main() {
	config.Init()
	err := database.InitDB()
	if err != nil {
		log.Fatalf("Error initializing database: %s\n", err)
	}

	// Initialize the central dispatcher
	disp := dispatcher.NewDispatcher()

	token := config.AppConfig.SlackToken
	appToken := config.AppConfig.SlackAppToken

	api := slack.New(token, slack.OptionDebug(true), slack.OptionDebug(false), slack.OptionAppLevelToken(appToken))

	client := socketmode.New(
		api,
		socketmode.OptionDebug(false),
		socketmode.OptionLog(l.New(os.Stdout, "socketmode: ", l.Lshortfile|l.LstdFlags)),
	)

	// Start listening for Slack events
	go func() {
		for evt := range client.Events {
			// Dispatch all events to the central dispatcher
			if err := disp.Dispatch(&evt, client); err != nil {
				log.Warnf("Error processing event: %s\n", err)
			}
		}
	}()

	log.Info("Bot is running...")
	// Run the Socket Mode client, which listens and responds to events from Slack
	if err := client.Run(); err != nil {
		log.Fatalf("Error running client: %v", err)
	}
}

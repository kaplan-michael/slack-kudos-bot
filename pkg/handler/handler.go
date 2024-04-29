package handler

import (
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// EventHandler handles generic events.
type EventHandler interface {
	HandleEvent(client *socketmode.Client, evt interface{}) error
}

// MessageEventHandler handles message events specifically.
type MessageEventHandler interface {
	HandleMessage(client *socketmode.Client, msgEvent *slackevents.MessageEvent) error
}

// CommandHandler handles slash commands.
type CommandHandler interface {
	HandleCommand(client *socketmode.Client, cmd string, event *socketmode.Event) error
}

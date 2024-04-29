package events

import (
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"regexp"
)

// MessageHandler defines the interface for all message handlers.
type MessageHandler interface {
	Matches(text string) bool
	Handle(client *socketmode.Client, msgEvent *slackevents.MessageEvent) error
}

// RegexMessageHandler implements the MessageHandler interface with a regex pattern.
type RegexMessageHandler struct {
	Pattern    *regexp.Regexp
	HandleFunc func(client *socketmode.Client, msgEvent *slackevents.MessageEvent) error
}

func (h *RegexMessageHandler) Matches(text string) bool {
	return h.Pattern.MatchString(text)
}

func (h *RegexMessageHandler) Handle(client *socketmode.Client, msgEvent *slackevents.MessageEvent) error {
	return h.HandleFunc(client, msgEvent)
}

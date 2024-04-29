package commands

import (
	"github.com/slack-go/slack/socketmode"
	"regexp"
)

// CommandHandler defines the interface for all message handlers.
type CommandHandler interface {
	Matches(text string) bool
	Handle(client *socketmode.Client, event *socketmode.Event) error
}

// RegexCommandHandler implements the MessageHandler interface with a regex pattern.
type RegexCommandHandler struct {
	Pattern    *regexp.Regexp
	HandleFunc func(client *socketmode.Client, event *socketmode.Event) error
}

func (h *RegexCommandHandler) Matches(text string) bool {
	return h.Pattern.MatchString(text)
}

func (h *RegexCommandHandler) Handle(client *socketmode.Client, event *socketmode.Event) error {
	return h.HandleFunc(client, event)
}

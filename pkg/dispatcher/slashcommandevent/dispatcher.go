package slashcommandevent

import (
	"fmt"

	"github.com/kaplan-michael/slack-kudos/pkg/handler/commands"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// Dispatcher for handling Slash Command events.

type Dispatcher struct {
	handlers []commands.CommandHandler
}

// NewDispatcher creates a new dispatcher for slash commands.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: []commands.CommandHandler{
			commands.NewKudosHandler(),
		},
	}
}

func (d *Dispatcher) Dispatch(evt *socketmode.Event, client *socketmode.Client) error {
	command, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		return fmt.Errorf("invalid command event type")
	}

	for _, handler := range d.handlers {
		if handler.Matches(command.Command) {
			return handler.Handle(client, evt)
		}
	}
	return nil
}

package eventsapievent

import (
	"fmt"
	"github.com/kaplan-michael/slack-kudos/pkg/handler/events"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type Dispatcher struct {
	handlers []events.MessageHandler
}

// NewDispatcher constructs a new Events API event dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: []events.MessageHandler{
			events.NewKudosHandler(),
		},
	}
}

func (d *Dispatcher) Dispatch(evt *socketmode.Event, client *socketmode.Client) error {
	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		return fmt.Errorf("unexpected event type: %s", evt.Type)
	}
	if messageEvent, ok := eventsAPIEvent.InnerEvent.Data.(*slackevents.MessageEvent); ok {
		for _, handler := range d.handlers {
			if handler.Matches(messageEvent.Text) {
				return handler.Handle(client, messageEvent)
			}
		}
	}
	return nil
}

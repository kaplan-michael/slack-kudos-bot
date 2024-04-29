package dispatcher

import (
	"github.com/kaplan-michael/slack-kudos/pkg/dispatcher/eventsapievent"
	"github.com/kaplan-michael/slack-kudos/pkg/dispatcher/slashcommandevent"
	"github.com/slack-go/slack/socketmode"
)

type Dispatcher struct {
	eventAPIEventDispatcher     *eventsapievent.Dispatcher
	slashCommandEventDispatcher *slashcommandevent.Dispatcher
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		eventAPIEventDispatcher:     eventsapievent.NewDispatcher(),
		slashCommandEventDispatcher: slashcommandevent.NewDispatcher(),
	}
}

func (d *Dispatcher) Dispatch(evt *socketmode.Event, client *socketmode.Client) error {
	//dispatch the event
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		client.Ack(*evt.Request)
		return d.eventAPIEventDispatcher.Dispatch(evt, client)
	case socketmode.EventTypeSlashCommand:
		client.Ack(*evt.Request)
		return d.slashCommandEventDispatcher.Dispatch(evt, client)
	}
	return nil
}

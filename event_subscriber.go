
package evs

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/golang/protobuf/proto"
)

// Users of the EventAnalytic API implement the Handler interface.
type EventHandler interface {
	Event(*Event, map[string]string) error
}

// EventAnalytic API wraps Pulsar communication and cyberprobe event decoding
type EventSubscriber struct {
	*Subscriber
	handler EventHandler
}

// Initialise the analyitc
func NewEventSubscriber(name string, binding string, e EventHandler) (*EventSubscriber, error) {
	s := &EventSubscriber{
	}

	var err error
	s.Subscriber, err = NewSubscriber(name, binding, s)
	if err != nil {
		return nil, err
	}

	s.handler = e

	return s, nil
}

// Internal Handler implementation of EventAnalytic, decodes messages as cyberprobe events
// and delegates to the EventHandler interface for processing.
func (s *EventSubscriber) Handle(msg pulsar.Message) error {
`	ev := &Event{}
	err := proto.Unmarshal(msg.Payload(), ev)
	if err != nil {
		return err
	}
	return s.handler.Event(ev, msg.Properties())
}


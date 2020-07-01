package evs

import (
	"github.com/apache/pulsar-client-go/pulsar"
	pb "github.com/cybermaggedon/evs-golang-api/protos"
	"github.com/golang/protobuf/proto"
)

// Wraps Pulsar communication and cyberprobe event encoding
type EventProducer struct {
	*Producer
}

// Initialise the analyitc
func NewEventProducer(name string, c HasOutputTopics) (*EventProducer, error) {

	p, err := NewProducer(name, c)
	if err != nil {
		return nil, err
	}

	ep := &EventProducer{
		Producer: p,
	}
	return ep, nil
}

// Output an event by iterating over all outputs
func (a *EventProducer) Output(ev *pb.Event, properties map[string]string) error {

	// Marshal event to protobuf
	b, err := proto.Marshal(ev)
	if err != nil {
		return err
	}

	// Create a ProducerMessage
	msg := pulsar.ProducerMessage{
		Payload:    b,
		Properties: properties,
		Key:        ev.Id,
	}

	// Delegate to Analytic.Output to output
	a.Producer.Output(msg)

	return nil

}

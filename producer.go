
package evs

import (
	"context"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/golang/protobuf/proto"
	"log"
	"os"
	"time"
)

// Describes  Users of the Analytic API implement the Handler interface.
type Producer struct {

	// Pulsar client
	// FIXME: Should be shared across publisher/subscriber
	client pulsar.Client

	// Output producers, is a map from output name to producer.
	outputs map[string]pulsar.Producer

}

// Initialise the Analytic.
func NewProducer(outputs []string) (*Producer, error) {

	p := &Producer{}

	// Get Pulsar broker location
	svc_addr, ok := os.LookupEnv("PULSAR_BROKER")
	if !ok {
		svc_addr = "pulsar://localhost:6650"
	}

	// Create Pulsar client
	var err error
	p.client, err = pulsar.NewClient(pulsar.ClientOptions{
		URL: svc_addr,
	})
	if err != nil {
		return nil, err
	}

	// Initialise outputs map, and create producers for each output
	p.outputs = make(map[string]pulsar.Producer)
	for _, output := range outputs {

		topic := fmt.Sprintf("%s://%s/%s/%s", persistence, tenant,
			namespace, output)

		producer, err := p.client.CreateProducer(pulsar.ProducerOptions{
			Topic: topic,
		})

		if err != nil {
			return nil, err
		}
		
		p.outputs[output] = producer
	}

	return p, nil

}

// Output a message by iterating over all outputs.  Retries until message is sent.
func (a *Producer) Output(msg pulsar.ProducerMessage) {

	for _, producer := range a.outputs {

		for {
			_, err := producer.Send(context.Background(), &msg)
			if err != nil {
				log.Printf("Pulsar Send: %v (will retry)", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

	}

}

// Wraps Pulsar communication and cyberprobe event encoding
type EventProducer struct {
	*Producer
}

// Initialise the analyitc
func NewEventProducer(outputs []string) (*EventProducer, error) {

	p, err := NewProducer(outputs)
	if err != nil {
		return nil, err
	}

	ep := &EventProducer{
		Producer: p,
	}
	return ep, nil
}

// Output an event by iterating over all outputs
func (a *EventProducer) Output(ev *Event, properties map[string]string) error {

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

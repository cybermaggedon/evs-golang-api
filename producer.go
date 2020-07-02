package evs

import (
	"context"
	"github.com/apache/pulsar-client-go/pulsar"
	"log"
	"os"
	"time"
)

// Describes  Users of the Analytic API implement the Handler interface.
type Producer struct {
	name string

	// Pulsar client
	// FIXME: Should be shared across publisher/subscriber
	client pulsar.Client

	// Output producers, is a map from output name to producer.
	producers map[string]pulsar.Producer

	// Tracking current state of delivery
	isWorking bool
}

// Initialise the Analytic.
func NewProducer(c HasOutputTopics) (*Producer, error) {

	name := c.GetName()
	topics := c.GetOutputTopics()
	p := &Producer{
		name: name,
		isWorking: true,
	}

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
	p.producers = make(map[string]pulsar.Producer)
	for _, topic := range topics {

		prod, err := p.client.CreateProducer(pulsar.ProducerOptions{
			Topic: topic,
		})

		if err != nil {
			return nil, err
		}

		p.producers[topic] = prod
	}

	return p, nil

}

func (a *Producer) sendCallback(id pulsar.MessageID, m *pulsar.ProducerMessage,
	err error) {
	if err != nil {

		if a.isWorking {
			log.Printf("Pulsar Send: %v", err)
			log.Print("Degraded, delivery failures will be retried")
			a.isWorking = false
		}
		
		time.Sleep(1000 * time.Millisecond)

		// FIXME: The message only failed on one output, not all of
		// them.  Should only retry the failed producer.

		// Retry message...
		a.Output(m)
		
	} else {
		if !a.isWorking {
			log.Print("Working again, delivery successful")
			a.isWorking = true
		}
	}
	
}

// Output a message by iterating over all outputs.  Retries until message is
// sent.
func (a *Producer) Output(msg *pulsar.ProducerMessage) {

	for _, producer := range a.producers {

		for {
			producer.SendAsync(context.Background(), msg,
				a.sendCallback)
		}

	}

}

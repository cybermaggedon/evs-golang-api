
package cyberprobe

import (
	"os"
	"github.com/google/uuid"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/golang/protobuf/proto"
	"log"
)

type Handler interface {
	Handle(msg pulsar.Message)
}

type Analytic struct {
	handler Handler
	ch chan pulsar.ConsumerMessage
	consumer pulsar.Consumer
}

func (a *Analytic) Init(binding string, outputs []string, h Handler) {

	a.handler = h
	
	svc_addr, ok := os.LookupEnv("PULSAR_BROKER"); if !ok {
		svc_addr = "pulsar://localhost:6650"
	}

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:                     svc_addr,
	})
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
	}

	a.ch = make(chan pulsar.ConsumerMessage, 1000)
	subs := uuid.New().String()

	consumerOpts := pulsar.ConsumerOptions{
		Topic:            "persistent://public/default/" + binding,
		SubscriptionName: subs,
		Type:             pulsar.Exclusive,
		MessageChannel:   a.ch,
	}

	a.consumer, err = client.Subscribe(consumerOpts)
	if err != nil {
		log.Fatalf("Could not establish subscription: %v", err)
	}

}

func (a *Analytic) Run() {

	defer a.consumer.Close()

	for cm := range a.ch {
		msg := cm.Message
		a.handler.Handle(msg)
		a.consumer.Ack(msg)
	}

}

type EventHandler interface {
	Event(*Event, map[string]string)
}

type EventAnalytic struct {
	Analytic
	handler EventHandler
}

func (a *EventAnalytic) Init(binding string, outputs []string, e EventHandler) {
	a.Analytic.Init(binding, outputs, a)
	a.handler = e
}

func (a *EventAnalytic) Handle(msg pulsar.Message) {
	ev := &Event{}
	err := proto.Unmarshal(msg.Payload(), ev)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	a.handler.Event(ev, msg.Properties())
}

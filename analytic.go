// There are two levels of analytic API here: Analytic wraps Pulsar and Prometheus client
// in this API, messages are passed around as Pulsar messages.  EventAnalytic wraps
// Analytic, and expects messages to be protobuf-encoded cyberprobe events, protobuf
// encode/decode is taken care of for the caller.
package cyberprobe

import (
	"context"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"time"
)

type Stoppable interface {
	Stop()
}

// Describes  Users of the Analytic API implement the Handler interface.
type Analytic struct {

	// Handler, interface is invoked when events are received.
	handler Handler

	// Communication channel used to deliver messages.
	ch chan pulsar.ConsumerMessage

	// Pulsar Consumer
	consumer pulsar.Consumer

	// Output producers, is a map from output name to producer.
	outputs map[string]pulsar.Producer

	// Prometheus metrics
	request_time prometheus.Summary
	event_size   prometheus.Histogram
	events       *prometheus.CounterVec

	// Am I running?
	running      bool

	// Cancellable context
	context      context.Context
	cancel       context.CancelFunc
	stoppable    Stoppable
}

func (a *Analytic) RegisterStop(stoppable Stoppable) {
	a.stoppable = stoppable
}

// Initialise the Analytic.
func (a *Analytic) Init(binding string, outputs []string, h Handler) {

	a.running = true

	a.context, a.cancel = context.WithCancel(context.Background())

	// Summary metric keeps track of request duration
	a.request_time = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "event_processing_time",
		Help: "Time spent processing event",
	})

	// Histogram metric keeps track of event sizes
	a.event_size = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "event_size",
		Help: "Size of event message",
		Buckets: []float64{25, 50, 100, 250, 500, 1000, 2500, 5000,
			10000, 25000, 50000, 100000, 250000, 500000,
			1000000, 2500000},
	})

	// Counter metric, keeps track of success/failure counts.
	a.events = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "events_total",
		Help: "Events processed total",
	}, []string{"state"})

	// Register prometheus metrics
	prometheus.MustRegister(a.request_time)
	prometheus.MustRegister(a.event_size)
	prometheus.MustRegister(a.events)

	a.handler = h

	// Launch prometheus web server
	metric_port, ok := os.LookupEnv("METRICS_PORT")
	if ok {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			log.Fatal(http.ListenAndServe(":"+metric_port, nil))
		}()
	}

	// Get Pulsar broker location
	svc_addr, ok := os.LookupEnv("PULSAR_BROKER")
	if !ok {
		svc_addr = "pulsar://localhost:6650"
	}

	// Create Pulsar client
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: svc_addr,
	})
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
	}

	// Create Pulsar incoming message queue, 1000 messages backlog
	a.ch = make(chan pulsar.ConsumerMessage, 1000)

	// Subscriber name is a new UUID
	subs := uuid.New().String()

	// Create topic name
	topic := fmt.Sprintf("%s://%s/%s/%s", persistence, tenant, namespace, binding)

	// Consumer options
	consumerOpts := pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subs,
		Type:             pulsar.Exclusive,
		MessageChannel:   a.ch,
	}

	// Create consumer by subscribing to the topic
	a.consumer, err = client.Subscribe(consumerOpts)
	if err != nil {
		log.Fatalf("Could not establish subscription: %v", err)
	}

	// Initialise outputs map, and create producers for each output
	a.outputs = make(map[string]pulsar.Producer)
	for _, output := range outputs {
		topic = fmt.Sprintf("%s://%s/%s/%s", persistence, tenant, namespace, output)
		producer, err := client.CreateProducer(pulsar.ProducerOptions{
			Topic: topic,
		})
		if err != nil {
			log.Fatalf("Could not instantiate Pulsar producer: %v", err)
		}
		a.outputs[output] = producer
	}

}

// Output a message by iterating over all outputs.  Retries until message is sent.
func (a *Analytic) Output(msg pulsar.ProducerMessage) {

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

// Go into the 'run' state getting messages from the consumer and delivering to Handler.
func (a *Analytic) Run() {

	defer a.consumer.Close()

	for a.running {
		select {
		case cm := <- a.ch:
			
			// Message from queue
			msg := cm.Message

			// Update metric with payload length
			a.event_size.Observe(float64(len(cm.Message.Payload())))

			// Create a timer to time request duration
			timer := prometheus.NewTimer(a.request_time)

			// Delegate message handling to the Handler interface.
			err := a.handler.Handle(msg)

			// Update metric with duration
			timer.ObserveDuration()

			// Record error state
			if err == nil {
				lbls := prometheus.Labels{"state": "success"}
				a.events.With(lbls).Inc()
			} else {
				lbls := prometheus.Labels{"state": "failure"}
				a.events.With(lbls).Inc()
				log.Printf("Error: %v\n", err)
			}
			a.consumer.Ack(msg)

		case <-time.After(500 * time.Millisecond):

			// Will return immediately and leave a set of unack'd
			// messages in the queue.

		}
			
	}

}

func (a *Analytic) Stop() {
	a.running = false
}

// Users of the EventAnalytic API implement the Handler interface.
/*
type EventHandler interface {
	Event(*Event, map[string]string) error
}
*/

// EventAnalytic API wraps Pulsar communication and cyberprobe event decoding

type EventAnalytic struct {
	Analytic
	handler EventHandler
}

// Initialise the analyitc
func (a *EventAnalytic) Init(binding string, outputs []string, e EventHandler) {
	a.Analytic.Init(binding, outputs, a)
	a.handler = e
}

// Internal Handler implementation of EventAnalytic, decodes messages as cyberprobe events
// and delegates to the EventHandler interface for processing.
func (a *EventAnalytic) Handle(msg pulsar.Message) error {
	ev := &Event{}
	err := proto.Unmarshal(msg.Payload(), ev)
	if err != nil {
		return err
	}
	return a.handler.Event(ev, msg.Properties())
}

// Output an event by iterating over all outputs
func (a *EventAnalytic) OutputEvent(ev *Event, properties map[string]string) error {

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
	a.Output(msg)

	return nil

}

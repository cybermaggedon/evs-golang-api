package evs

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"time"
)

// Users of the Analytic API implement the Handler interface.
type Handler interface {
	Handle(msg pulsar.Message) error
}

// Describes  Users of the Analytic API implement the Handler interface.
type Subscriber struct {

	// Analytic name
	name string

	// Handler, interface is invoked when events are received.
	handler Handler

	// Communication channel used to deliver messages.
	ch chan pulsar.ConsumerMessage

	// Pulsar client
	// FIXME: Should be shared across publisher/subscriber
	client pulsar.Client

	// Pulsar Consumer
	consumer pulsar.Consumer

	// Prometheus metrics
	request_time *prometheus.SummaryVec
	event_size   *prometheus.HistogramVec
	events       *prometheus.CounterVec

	// Am I running?
	running bool
}

// Initialise the Analytic.
func NewSubscriber(name string, c HasInputTopics, h Handler) (*Subscriber, error) {

	s := &Subscriber{name: name}

	s.running = true

	// Summary metric keeps track of request duration
	s.request_time = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "event_processing_time",
		Help: "Time spent processing event",
	}, []string{"analytic"})

	// Histogram metric keeps track of event sizes
	s.event_size = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "event_size",
		Help: "Size of event message",
		Buckets: []float64{25, 50, 100, 250, 500, 1000, 2500, 5000,
			10000, 25000, 50000, 100000, 250000, 500000,
			1000000, 2500000},
	}, []string{"analytic"})

	// Counter metric, keeps track of success/failure counts.
	s.events = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "event_total",
		Help: "Events processed total",
	}, []string{"analytic", "state"})

	// Register prometheus metrics
	prometheus.MustRegister(s.request_time)
	prometheus.MustRegister(s.event_size)
	prometheus.MustRegister(s.events)

	s.handler = h

	CheckAndStartMetricsService()

	// Get Pulsar broker location
	svc_addr, ok := os.LookupEnv("PULSAR_BROKER")
	if !ok {
		svc_addr = "pulsar://localhost:6650"
	}

	// Create Pulsar client
	var err error
	s.client, err = pulsar.NewClient(pulsar.ClientOptions{
		URL: svc_addr,
	})
	if err != nil {
		return nil, err
	}

	// Create Pulsar incoming message queue, 1000 messages backlog
	s.ch = make(chan pulsar.ConsumerMessage, 1000)

	// Subscriber name is a new UUID
	subs := name + "-" + uuid.New().String()

	// Consumer options
	consumerOpts := pulsar.ConsumerOptions{
		Topic:            c.GetInputTopic(),
		SubscriptionName: subs,
		Type:             pulsar.Shared,
		MessageChannel:   s.ch,
	}

	// Create consumer by subscribing to the topic
	s.consumer, err = s.client.Subscribe(consumerOpts)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Subscriber) Close() {
	s.consumer.Close()
	s.client.Close()

}

// Go into the 'run' state getting messages from the consumer and delivering
// to Handler.
func (s *Subscriber) Run() {

	for s.running {
		select {
		case cm := <-s.ch:

			// Message from queue
			msg := cm.Message

			// Update metric with payload length
			lbls := prometheus.Labels{"analytic": s.name}
			s.event_size.With(lbls).
				Observe(float64(len(cm.Message.Payload())))

			// Create a timer to time request duration
			lbls = prometheus.Labels{"analytic": s.name}
			timer := prometheus.NewTimer(s.request_time.With(lbls))

			// Delegate message handling to the Handler interface.
			err := s.handler.Handle(msg)

			// Update metric with duration
			timer.ObserveDuration()

			// Record error state
			if err == nil {
				lbls := prometheus.Labels{
					"analytic": s.name,
					"state":    "success",
				}
				s.events.With(lbls).Inc()
			} else {
				lbls := prometheus.Labels{
					"analytic": s.name,
					"state":    "failure",
				}
				s.events.With(lbls).Inc()
				log.Printf("Error: %v\n", err)
			}
			s.consumer.Ack(msg)

		case <-time.After(100 * time.Millisecond):

		}

	}

	// Will return immediately and leave a set of unack'd
	// messages in the queue.

}

func (s *Subscriber) Stop() {
	s.running = false
}

var metrics_started = false

func CheckAndStartMetricsService() {

	if metrics_started {
		return
	}

	// Check if metrics port is specified.  If not, do nothing.
	metric_port, ok := os.LookupEnv("METRICS_PORT")
	if !ok {
		return
	}

	// Launch prometheus web server
	metric_port = ":" + metric_port

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(metric_port, nil)
		log.Fatal(err)
	}()

	// Minor naming error maybe.
	metrics_started = true

}

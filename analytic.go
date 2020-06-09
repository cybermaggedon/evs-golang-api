package cyberprobe

import (
	"context"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"time"
	"fmt"
)

const (
	persistence = "persistent"
	tenant = "public"
	namespace = "default"
)

var (
	request_time = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "event_processing_time",
		Help: "Time spent processing event",
	})
	event_size = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "event_size",
		Help: "Size of event message",
		Buckets: []float64{25, 50, 100, 250, 500, 1000, 2500, 5000,
			10000, 25000, 50000, 100000, 250000, 500000,
			1000000, 2500000},
	})
	events = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "events_total",
		Help: "Events processed total",
	}, []string{"state"})
)

type Handler interface {
	Handle(msg pulsar.Message) error
}

type Analytic struct {
	handler  Handler
	ch       chan pulsar.ConsumerMessage
	consumer pulsar.Consumer
	outputs  map[string]pulsar.Producer
}

func (a *Analytic) Init(binding string, outputs []string, h Handler) {

	prometheus.MustRegister(request_time)
	prometheus.MustRegister(event_size)
	prometheus.MustRegister(events)

	a.handler = h

	metric_port, ok := os.LookupEnv("METRICS_PORT")
	if ok {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			log.Fatal(http.ListenAndServe(":"+metric_port, nil))
		}()
	}

	svc_addr, ok := os.LookupEnv("PULSAR_BROKER")
	if !ok {
		svc_addr = "pulsar://localhost:6650"
	}

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: svc_addr,
	})
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
	}

	a.ch = make(chan pulsar.ConsumerMessage, 1000)
	subs := uuid.New().String()

	topic := fmt.Sprintf("%s://%s/%s/%s", persistence, tenant, namespace, binding)

	consumerOpts := pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subs,
		Type:             pulsar.Exclusive,
		MessageChannel:   a.ch,
	}

	a.consumer, err = client.Subscribe(consumerOpts)
	if err != nil {
		log.Fatalf("Could not establish subscription: %v", err)
	}

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

func (a *Analytic) Run() {

	defer a.consumer.Close()

	for cm := range a.ch {
		msg := cm.Message

		event_size.Observe(float64(len(cm.Message.Payload())))

		timer := prometheus.NewTimer(request_time)

		err := a.handler.Handle(msg)

		timer.ObserveDuration()

		if err == nil {
			events.With(prometheus.Labels{"state": "success"}).Inc()
		} else {
			events.With(prometheus.Labels{"state": "failure"}).Inc()
		}
		a.consumer.Ack(msg)
	}

}

type EventHandler interface {
	Event(*Event, map[string]string) error
}

type EventAnalytic struct {
	Analytic
	handler EventHandler
}

func (a *EventAnalytic) Init(binding string, outputs []string, e EventHandler) {
	a.Analytic.Init(binding, outputs, a)
	a.handler = e
}

func (a *EventAnalytic) Handle(msg pulsar.Message) error {
	ev := &Event{}
	err := proto.Unmarshal(msg.Payload(), ev)
	if err != nil {
		return err
	}
	return a.handler.Event(ev, msg.Properties())
}

func (a *EventAnalytic) OutputEvent(ev Event, properties map[string]string) error {

	b, err := proto.Marshal(&ev)
	if err != nil {
		return err
	}

	msg := pulsar.ProducerMessage{
		Payload:    b,
		Properties: properties,
		Key:        ev.Id,
	}

	a.Output(msg)

	return nil

}

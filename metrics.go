
package evs

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
)

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

// Package functions contains GCF implementations for updating Cloud Monitoring
// metrics.
package functions

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	// import otel sdk libraries for instrumentation.
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	otelmetric "go.opentelemetry.io/otel/metric"
	otelglobal "go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/api/idtoken"
)

var meter otelmetric.Meter
var metricController *controller.Controller

func init() {
	var err error
	ctx := context.Background()

	// text exporter, reports metrics on stdout.
	textporter, _ := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
		stdoutmetric.WithWriter(log.Writer()),
	)
	// otlp exporter.
	// Create an auth token source, authenticating as the Service Account that this function runs as.
	// audience claim must match the url of the Cloud Run service we're calling.
	audienceClaim := "https://otel-collector-ridqe6ysba-uw.a.run.app/"
	tokensrc, err := idtoken.NewTokenSource(ctx, audienceClaim)
	if err != nil {
		log.Fatalf("failed create auth token source: %s", err)
	}
	t, err := tokensrc.Token()
	if err != nil {
		log.Fatalf("failed get auth token: %s", err)
	}
	// These lines can help debug by printing the extracted token details
	// p, err := idtoken.Validate(ctx, t.AccessToken, audienceClaim)
	// log.Printf("payload of token: %#v or error: %s", p, err)

	exporter, err := otlpmetrichttp.New(
		ctx, otlpmetrichttp.WithMaxAttempts(1),
		otlpmetrichttp.WithTimeout(30*time.Second),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithHeaders(
			map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", t.AccessToken),
			}),
		// NB: this is a cloud run hostname, which is non-deterministic.
		otlpmetrichttp.WithEndpoint("otel-collector-ridqe6ysba-uw.a.run.app"),
	)
	if err != nil {
		log.Fatalf("Failed to create otlp exporter: %s", err)
	}

	// The controller handles periodic collection and exporting of metrics.
	metricController = controller.New(
		processor.NewFactory(
			simple.NewWithInexpensiveDistribution(),
			textporter,
		),
		// Note: only one of these exporters can be used at a time.
		// TODO: consider a multi-exporter, which will call interface methods on both.
		// controller.WithExporter(textporter),
		controller.WithExporter(exporter),
		controller.WithResource(resource.Empty()),
	)
	err = metricController.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start metric controller: %s", err)
	}
	// Set our controller as the global MeterProvider, so instrumented libraries
	// will report through this controller.
	otelglobal.SetMeterProvider(metricController)

	// Make a new meter, which instruments our beancounter library.
	meter = otelglobal.Meter("beancounter", otelmetric.WithInstrumentationVersion("v0.1.0"))
}

// otelPush creates an otel counter with the given metric name, and records the value.
func otelPush(ctx context.Context, w http.ResponseWriter, metric string, value int64) {
	counter, err := meter.NewInt64UpDownCounter(metric)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to create metric: %v", err)))
		log.Fatalf("Failed to create metric: %v", err)
	}
	counter.Add(ctx, 1)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("Data point recorded for metric: '%s'.", metric)))
}

// ChangeEvent pushes a timeseries point with a "change" event, happening at the current time.
func ChangeEvent(w http.ResponseWriter, r *http.Request) {
	otelPush(r.Context(), w, "changes", 1)
}

// FeedEvent records a metric point for feeding.
func FeedEvent(w http.ResponseWriter, r *http.Request) {
	otelPush(r.Context(), w, "feedings", 1)
}

// StatusEvent records an integer point, labeled with the 'status' query parameter.
func MoodEvent(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("status")
	if s == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("could not read status from query string."))
		return
	}

	counter, err := meter.NewInt64UpDownCounter("status")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to create metric: %v", err)))
		log.Fatalf("Failed to create metric: %v", err)
	}
	counter.Add(r.Context(), 1, attribute.KeyValue{
		Key:   "status",
		Value: attribute.StringValue(s),
	})
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("Data point recorded for metric: '%s'.", "status")))
}

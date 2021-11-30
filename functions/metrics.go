// Package functions contains GCF implementations for updating Cloud Monitoring
// metrics.
package functions

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	googlepb "github.com/golang/protobuf/ptypes/timestamp"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	// import otel sdk libraries for instrumentation.
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	otelmetric "go.opentelemetry.io/otel/metric"
	otelglobal "go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Note: be sure to configure this as a Runtime Variable in GCF.
var projectId string = os.Getenv("GOOGLE_PROJECT_ID")
var client *monitoring.MetricClient
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

	// The controller handles periodic collection and exporting of metrics.
	metricController = controller.New(
		processor.NewFactory(
			simple.NewWithInexpensiveDistribution(),
			textporter,
		),
		controller.WithExporter(textporter),
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
	meter = otelglobal.Meter("beancounter", metric.WithInstrumentationVersion("v0.1.0"))

	// Use google's Monitoring API directly.
	client, err = monitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create Metrics Client: %v\n", err)
	}
}

// newIntPoint creates a Point with the given value, at the current time.
func newIntPoint(value int64, ts int64) *monitoringpb.Point {
	if ts == 0 {
		ts = time.Now().Unix()
	}
	return &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			EndTime: &googlepb.Timestamp{
				Seconds: ts,
			},
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_Int64Value{Int64Value: value},
		},
	}

}

func newDoublePoint(value float64, ts int64) *monitoringpb.Point {
	if ts == 0 {
		ts = time.Now().Unix()
	}
	return &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			EndTime: &googlepb.Timestamp{
				Seconds: ts,
			},
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DoubleValue{DoubleValue: value},
		},
	}

}

// metricPush creates a single new point in a timeseries.
func metricPush(ctx context.Context, w http.ResponseWriter, metric string, labels map[string]string, value *monitoringpb.Point) {
	err := client.CreateTimeSeries(ctx, &monitoringpb.CreateTimeSeriesRequest{
		// Note: code snippet recommends a deprecated function here.
		Name: fmt.Sprintf("projects/%s", projectId),
		TimeSeries: []*monitoringpb.TimeSeries{
			{
				Metric: &metricpb.Metric{
					Type:   path.Join("custom.googleapis.com", metric),
					Labels: labels,
				},
				Points: []*monitoringpb.Point{value},
			},
		},
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to create timeseries: %v", err)))
		log.Fatalf("Failed to create timeseries: %v", err)
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Data point recorded."))

}

// ChangeEvent pushes a timeseries point with a "change" event, happening at the current time.
func ChangeEvent(w http.ResponseWriter, r *http.Request) {
	counter, err := meter.NewInt64Counter("changes")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to create metric: %v", err)))
		log.Fatalf("Failed to create metric: %v", err)
	}
	counter.Add(r.Context(), 1)

	metricPush(r.Context(), w, "/beancounter/changes", nil, newIntPoint(1, 0))
}

// FeedEvent records a metric point for feeding.
func FeedEvent(w http.ResponseWriter, r *http.Request) {
	counter, err := meter.NewInt64Counter("feedings")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to create metric: %v", err)))
		log.Fatalf("Failed to create metric: %v", err)
	}
	counter.Add(r.Context(), 1)
	metricPush(r.Context(), w, "/beancounter/feedings", nil, newIntPoint(1, 0))
}

// StatusEvent records an integer point, labeled with the 'status' query parameter.
func MoodEvent(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("status")
	if s == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("could not read status from query string."))
		return
	}

	counter, err := meter.NewInt64Counter("status")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to create metric: %v", err)))
		log.Fatalf("Failed to create metric: %v", err)
	}
	counter.Add(r.Context(), 1, attribute.KeyValue{
		Key:   "status",
		Value: attribute.StringValue(s),
	})

	labels := map[string]string{"status": s}
	metricPush(r.Context(), w, "/beancounter/status-label", labels, newIntPoint(1, 0))
}

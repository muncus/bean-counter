# Bean Counter

This is an exploration project for me to learn how to use Cloud Functions and
Cloud Run with Cloud Monitoring. Particularly, how does one get metrics out of
CF/CR and into Cloud Monitoring tools, using a minimum amount of vendor-specific
solutions.

## Current architecture:

☺ -> ☁️ (GCF) -> 🔭 (otel collector) -> :bar_chart: (cloud monitoring)

Cloud Functions create OpenTelemetry metrics, and push them to an OTel collector
backend.  The Collector backend is a Cloud Run service, running a version of the
[otel-collector-contrib](http://github.com/open-telemetry/opentelemetry-collector-contrib)
container with the config found in `otel-collector/collector-config.yaml`. This
config sends all received metrics to the Google Cloud Monitoring service.

These metrics are then displayed with a dashboard, created and managed by 
terraform configs.

## Stage 1: Custom work

To get a functional base, I started with calling the vendor-specific apis
directly from GCF. This is not the desired end state, but it does provide a
"base case" to build from.

This stage also includes terraform configurations for *most* of the required
components (e.g. Monitoring Services and Metric Descriptors).

## Stage 2: Explore OpenTelemetry

OpenTelemetry ("otel") recently announced the Metrics API spec has become
Stable. With this step forward, it is now possible to instrument libraries with
a common SDK, and allow the binary to determine how to export those metrics.

The first stage used the
[stdoutmetric](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/stdout/stdoutmetric)
exporter, which simply prints the metrics to stdout. The next step was to push
the metrics to a collector, over otel's wire protocol, OTLP.

I investigated the use of `otel-collector` to receive prometheus metrics, but
for serverless platforms like GCF, and Run, this is not a feasible solution,
because the collector does not accept push metrics at this time. (there is an
[issue for pushgateway
support](https://github.com/open-telemetry/opentelemetry-go/issues/522), but it
is not currently a high priority.

### Pending work:
- [X] Convert existing metrics to use the otel SDK
  - First step is to use `exporter/stdout`
  - [X] (later) export over OTLP to collector.
- [X] Create a OTLP collector instance that can be run in Cloud Run.
  - This collector is `otel/opentelemetry-collector-contrib` with a config found
  in the `otel-collector` directory. 
- [X] make Functions return sensible error codes, with text body.
- [ ] decide on whether or not we need the terraform for metric descriptors, now
that otel-collector-contrib can do it for us.
- [ ] add memory limiter to the otel collector.


## Unplanned, future investigations
- Prometheus Push metric support in otel-collector-contrib?
- What does the prometheus -> otel conversion look like for a more persistent service?
- add tracing support in gcf functions, and configure tail sampling with the collector.
- [-] find a way to stop depending on the Cloud Run hostname for exporting.
  - Auth through an LB still requires the token audience match the cloud run
  hostname, so this seems infeasible for now.


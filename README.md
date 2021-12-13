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

## Stage 2: OpenTelemetry

OpenTelemetry ("otel") recently announced the Metrics API spec has become
Stable. With this step forward, it is now possible to instrument libraries with
a common SDK, and allow the binary to determine how to export those metrics.

#### OTel Terms

There is more complete coverage of [terms for
Metrics](https://opentelemetry.io/docs/reference/specification/metrics/api/#overview),
and [terms for
Traces](https://opentelemetry.io/docs/reference/specification/trace/api/), but
here's a quick refresher.

* Exporter - the component that exposes your metrics/traces to external systems.
  For example, a Prometheus exporter, or an OTLP exporter.
* MeterProvider or TraceProvider - These provide a way to create Meters or
  Tracers, and are usually accessed from a global context by way of
  `GetMeterProvider()`.
* Meter - Meter is a collection of Metrics. It is common practice to create a
  Meter for logical sets of metrics. For example, a library can create its own
  Meter, containing all the metrics it creates (also called Instruments)
* Instruments - Instruments are individual metrics that you want to export. they
  can be of various types, and have additional key/value labels on them.

### OTel first steps

The first step with OTel is laying in the basic components of the OTel SDK. In addition to the parts listed above, we also need a `Controller` to manage collection from the various `Meter`s.
The initial `Exporter` I used was
[stdoutmetric](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/stdout/stdoutmetric)
exporter, which simply prints the metrics to stdout. 

Once that was working as expected, the next step was to push the metrics to a
collector, over otel's wire protocol, OTLP.

I investigated the use of `otel-collector` to receive prometheus metrics, but
for serverless platforms like GCF, and Run, this is not a feasible solution,
because the collector does not accept prometheus push metrics at this time. (there is an
[issue for pushgateway
support](https://github.com/open-telemetry/opentelemetry-go/issues/522), but it
is not currently a high priority.

### Collector in Cloud Run

The OTel Collector is a sort of proxy: it receives telemetry , and forwards it
on to another backend (optionally, doing other processing like batching and
protocol translation).
For this use case, we've configured it to receive OTLP data, and emit data 
directly to google cloud monitoring (and some log output, for verification).

As seen in `otel-collector/Dockerfile`, our use case is quite simple. We build a
container from the otel-published contrib image (which contains the
`googlecloudexporter`), and replace the config file with our own.

The only tricky bit here is that the opened port must meet the [Cloud Run
container
contract](https://cloud.google.com/run/docs/reference/container-contract).
Specifically, it must listen on port 8080 - using the `$PORT` environment
variable would require extra indirection, because port numbers are included in
the config file directly.

### OTLP Exports

To change from stdout to the OTLP exporter was straightforward. 

With our Collector service running in Cloud Run, we needed to create an
[`otlpmetrichttp`](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp)
exporter, and point it at our Cloud Run hostname.

The code authenticates to the collector service using the GCF service account,
which must be granted the `cloud run invoker` permission in IAM.

#### Reliable Exports

The first attempt at OTLP exports created a number of "context deadline
exceeded" errors in the GCF logs, where the Cloud Function failed to export data
to the Collector service. To address this, we are now explicitly calling
`Start()`, `Collect()`, and `Stop()` on the metric controller *in the request
flow*. the collector can be started multiple times, so concurrent requests do
not create problems here.

## Pending work:
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
- Consider creating a multi-exporter, which can contain more than one exporter,
  like a Prometheus and an OTLP exporter.


# Bean Counter

This is an exploration project for me to learn how to use Cloud Functions and
Cloud Run with Cloud Monitoring. Particularly, how does one get metrics out of
CF/CR and into Cloud Monitoring tools, using a minimum amount of vendor-specific
solutions.

## Stage 1: Custom work

To get a functional base, I started with calling the vendor-specific apis
directly from GCF. This is not the desired end state, but it does provide a
"base case" to build from.

This stage also includes terraform configurations for *most* of the required
components (e.g. Monitoring Services and Metric Descriptors).

## Stage 2: Explore OpenTelemetry

OpenTelemetry ("otel") recently announced the Metrics API spec has become
Stable. With this step forward, it is now possible to instrument libraries with
a common SDK, and allow the binary to determine how to export those metrics
(over either otel's own protocol (OTLP), or as Prometheus metrics, for example).

I investigated the use of `otel-collector` to convert exported prometheus
metrics into Cloud Monitoring data, but for serverless platforms like GCF, and
Run, this is not a feasible solution, because the collector does not accept push
metrics at this time. (there is an [issue for pushgateway
support](https://github.com/open-telemetry/opentelemetry-go/issues/522), but it
is not currently a high priority.

### Pending work:
- [ ] Convert existing metrics to use the otel SDK (prom format first)
  - [ ] (later) export over OTLP to collector.
- [ ] Create a OTLP collector instance that can be run in Cloud Run.
  - This collector should use the `otel/otel-collector-contrib` base, which has
    a
    [`googlecloudexporter`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/googlecloudexporter)
    to get our metrics into cloud monitoring.


## Unplanned, future investigations
- Push metric support in otel-collector-contrib?
  - or, find alternate way to enable custom metric exports from GCF/CR

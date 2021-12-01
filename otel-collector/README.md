# OpenTelemetry collector configuration

The goal is to create an Otel collector suitable for running in Cloud Run, with
the intention of proxying otel (otlp) metrics into Google Cloud Monitoring.

To use:

* `docker build . -t $YOURREPO/otel-collector:0.1`
* `docker push $YOURREPO/otel-collector:0.1`
* `gcloud run deploy --region us-west1 otel-collector --image $YOURREPO/otel-collector:0.1`

Note that the MetricDescriptor of the Cloud Monitoring metric is influenced by the type of Instrument used.
For example, and `Int64Counter` must have type `CUMULATIVE`. An `Int64UpDownCounter` becomes a `GAUGE`

### Reference:
- [otel collector](https://github.com/open-telemetry/opentelemetry-collector)
- [googlecloud exporter docs](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/googlecloudexporter)
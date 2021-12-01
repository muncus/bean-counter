# Build an image based on otel/otel-collector-contrib, that receives otlp, and
# forwards on to the google monitoring api.

FROM otel/opentelemetry-collector-contrib
COPY collector-config.yaml /etc/otel/config.yaml

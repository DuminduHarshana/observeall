# =============================================================
# ObserveAll — Custom OTel Collector Distribution
# Based on otelcol-contrib with ObserveAll defaults baked in
# =============================================================
FROM otel/opentelemetry-collector-contrib:0.95.0 AS base

# Copy default collector config (can be overridden at runtime)
COPY collector-config.yaml /etc/otelcol/config.yaml

# Expose OTLP gRPC, OTLP HTTP, metrics, and health check ports
EXPOSE 4317 4318 8888 13133

ENTRYPOINT ["/otelcol-contrib"]
CMD ["--config=/etc/otelcol/config.yaml"]

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:13133/ || exit 1

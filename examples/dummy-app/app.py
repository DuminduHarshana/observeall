#!/usr/bin/env python3
"""
ObserveAll Dummy Application
Demonstrates Traces, Metrics, and Error Reporting via standard OpenTelemetry (OTLP)
streaming to ObserveAll Collector (localhost:4317 / 4318).
"""

import time
import random
import logging
from flask import Flask, request, jsonify

# OpenTelemetry Tracing
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.trace import Status, StatusCode

# OpenTelemetry Metrics
from opentelemetry import metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.exporter.otlp.proto.http.metric_exporter import OTLPMetricExporter

# Configure Resource
resource = Resource.create({
    "service.name": "dummy-shop-service",
    "service.version": "1.0.0",
    "deployment.environment": "local-dev",
})

# 1. Setup Tracing Provider & OTLP HTTP Exporter (points to ObserveAll on port 4318)
tracer_provider = TracerProvider(resource=resource)
otlp_span_exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
tracer_provider.add_span_processor(BatchSpanProcessor(otlp_span_exporter, max_export_batch_size=10, schedule_delay_millis=500))
trace.set_tracer_provider(tracer_provider)
tracer = trace.get_tracer("dummy-shop-tracer")

# 2. Setup Metrics Provider & OTLP HTTP Exporter
metric_reader = PeriodicExportingMetricReader(
    OTLPMetricExporter(endpoint="http://localhost:4318/v1/metrics"),
    export_interval_millis=1000,
)
meter_provider = MeterProvider(resource=resource, metric_readers=[metric_reader])
metrics.set_meter_provider(meter_provider)
meter = metrics.get_meter("dummy-shop-meter")

# Create sample metrics
request_counter = meter.create_counter(
    name="dummy_requests_total",
    description="Total HTTP requests processed by dummy app",
    unit="1",
)
order_counter = meter.create_counter(
    name="dummy_orders_total",
    description="Total completed orders",
    unit="1",
)
order_value_histogram = meter.create_histogram(
    name="dummy_order_amount_usd",
    description="Order transaction values",
    unit="USD",
)

# Setup Flask App
app = Flask(__name__)
logging.basicConfig(level=logging.INFO)

@app.route("/")
def home():
    request_counter.add(1, {"endpoint": "/", "status": "200"})
    with tracer.start_as_current_span("home_handler") as span:
        span.set_attribute("http.method", "GET")
        span.set_attribute("http.route", "/")
        return jsonify({
            "service": "dummy-shop-service",
            "status": "healthy",
            "message": "ObserveAll telemetry active",
        })

@app.route("/order", methods=["GET", "POST"])
def place_order():
    request_counter.add(1, {"endpoint": "/order", "status": "200"})
    items_count = int(request.args.get("items", random.randint(1, 5)))
    amount = round(items_count * random.uniform(15.0, 45.0), 2)

    with tracer.start_as_current_span("process_order") as root_span:
        root_span.set_attribute("order.items_count", items_count)
        root_span.set_attribute("order.amount_usd", amount)

        # Step 1: Validate Cart
        with tracer.start_as_current_span("validate_cart") as span:
            time.sleep(0.02)
            span.set_attribute("cart.valid", True)

        # Step 2: Check Inventory
        with tracer.start_as_current_span("check_inventory") as span:
            time.sleep(0.03)
            span.set_attribute("inventory.warehouse", "us-east-1")
            span.set_attribute("inventory.available", True)

        # Step 3: Process Payment
        with tracer.start_as_current_span("charge_payment") as span:
            time.sleep(0.04)
            span.set_attribute("payment.gateway", "stripe-mock")
            span.set_attribute("payment.status", "succeeded")

        order_counter.add(1, {"status": "success"})
        order_value_histogram.record(amount, {"currency": "USD"})

        return jsonify({
            "order_id": f"ord-{random.randint(1000, 9999)}",
            "items": items_count,
            "amount_usd": amount,
            "status": "confirmed",
            "trace_id": format(root_span.get_span_context().trace_id, "032x"),
        })

@app.route("/slow")
def slow_endpoint():
    """Triggers ObserveAll tail-sampling latency threshold (> 500ms)."""
    request_counter.add(1, {"endpoint": "/slow", "status": "200"})
    with tracer.start_as_current_span("slow_operation") as span:
        span.set_attribute("operation.type", "heavy_computation")
        time.sleep(0.55)  # 550ms -> triggers tail_sampling slow-traces-policy
        return jsonify({
            "status": "completed",
            "duration_ms": 550,
            "sampled_by": "tail_sampling: slow-traces-policy (>500ms)",
        })

@app.route("/error")
def trigger_error():
    """Triggers an error to verify error-catching & tail-sampling errors-policy."""
    request_counter.add(1, {"endpoint": "/error", "status": "500"})
    with tracer.start_as_current_span("failing_operation") as span:
        try:
            span.set_attribute("endpoint", "/error")
            raise ValueError("Simulated database connection failure")
        except Exception as ex:
            span.record_exception(ex)
            span.set_status(Status(StatusCode.ERROR, str(ex)))
            return jsonify({
                "error": True,
                "message": str(ex),
                "tail_sampling": "preserved 100% by errors-policy",
            }), 500

if __name__ == "__main__":
    print("🚀 Starting Dummy Shop Service on http://0.0.0.0:5050")
    print("📡 Streaming traces & metrics to ObserveAll Collector at http://localhost:4318")
    app.run(host="0.0.0.0", port=5050)


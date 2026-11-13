# 🛍️ ObserveAll Dummy Application

A practical reference application instrumented with **OpenTelemetry (OTel)** to validate end-to-end telemetry streaming into the **ObserveAll Collector**.

---

## 🎯 What This Tests

1. **Distributed Tracing (OTLP/HTTP & gRPC)**:
   - Root spans and nested child spans (`process_order` ➔ `validate_cart`, `check_inventory`, `charge_payment`).
   - W3C Trace IDs and span attributes.
2. **Metrics Ingestion**:
   - Monotonic request counters (`dummy_requests_total`).
   - Business metric counters (`dummy_orders_total`).
   - Value distribution histograms (`dummy_order_amount_usd`).
3. **Tail-Based Sampling**:
   - **Error Preservation (`/error`)**: Simulates 500 exceptions, verifying ObserveAll's `errors-policy` preserves 100% of failed traces.
   - **Latency Threshold (`/slow`)**: Simulates > 500ms operations, verifying ObserveAll's `slow-traces-policy`.
4. **Enrichment & Pipeline**:
   - Verifies the collector applies global attributes (`deployment.environment=production`, `observeall.version=1.0`) and batches records.

---

## 🚀 How to Run

### 1. Ensure the ObserveAll Collector is Running
```bash
# In the repository root
docker compose up -d
./bin/observeall status
```

### 2. Start the Dummy Application
```bash
# Using the installed Python environment
./examples/dummy-app/.venv/bin/python examples/dummy-app/app.py
```
App starts on `http://localhost:5050`.

### 3. Run the Automated Test Suite
In a separate terminal:
```bash
./examples/dummy-app/.venv/bin/python examples/dummy-app/test_app.py
```

### 4. Observe the Signals in Collector Logs
View real-time trace flame data, metrics, and enriched attributes being processed by the collector:
```bash
docker logs -f otel-collector
```


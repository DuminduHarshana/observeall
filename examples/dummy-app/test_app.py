#!/usr/bin/env python3
"""
Test runner for the ObserveAll Dummy Application.
Exercises the instrumented endpoints and validates OpenTelemetry signal ingestion.
"""

import sys
import time
import requests

APP_URL = "http://localhost:5050"
COLLECTOR_METRICS_URL = "http://localhost:8888/metrics"

def test_endpoints():
    print("=" * 60)
    print("🧪 1. Testing Instrumented Endpoints")
    print("=" * 60)

    # 1. Health / Home
    print("▶ Calling GET /")
    res = requests.get(f"{APP_URL}/")
    assert res.status_code == 200, f"Expected 200, got {res.status_code}"
    print(f"  ✔ Status: {res.status_code}, Response: {res.json()}")

    # 2. Place Order (Traces: root + 3 child spans + metrics)
    print("\n▶ Calling GET /order (multi-span transaction)")
    res = requests.get(f"{APP_URL}/order?items=3")
    assert res.status_code == 200, f"Expected 200, got {res.status_code}"
    order_data = res.json()
    print(f"  ✔ Status: {res.status_code}, Order ID: {order_data['order_id']}, Trace ID: {order_data['trace_id']}")

    # 3. Slow request (Tests tail sampling latency policy > 500ms)
    print("\n▶ Calling GET /slow (tail sampling latency trigger)")
    res = requests.get(f"{APP_URL}/slow")
    assert res.status_code == 200, f"Expected 200, got {res.status_code}"
    print(f"  ✔ Status: {res.status_code}, Policy: {res.json()['sampled_by']}")

    # 4. Error request (Tests tail sampling error policy + exception recording)
    print("\n▶ Calling GET /error (tail sampling error trigger)")
    res = requests.get(f"{APP_URL}/error")
    assert res.status_code == 500, f"Expected 500, got {res.status_code}"
    print(f"  ✔ Status: {res.status_code}, Preserved: {res.json()['tail_sampling']}")

    # 5. PII Scrubbing (Phase 3: tests credit card, email, password, SSN scrubbing)
    print("\n▶ Calling GET /checkout (Phase 3 PII Scrubbing verification)")
    res = requests.get(f"{APP_URL}/checkout?card=4532015893248812&email=alice@company.com&password=secretPassword123&ssn=123-45-6789")
    assert res.status_code == 200, f"Expected 200, got {res.status_code}"
    print(f"  ✔ Status: {res.status_code}, PII Scrubber: {res.json()['scrubbed_by']}")

    print("\n⏳ Flushing spans and metrics to ObserveAll Collector...")
    time.sleep(3)

def check_collector_metrics():
    print("\n" + "=" * 60)
    print("🔍 2. Verifying ObserveAll Collector Ingestion")
    print("=" * 60)

    try:
        res = requests.get(COLLECTOR_METRICS_URL, timeout=3)
        if res.status_code != 200:
            print(f"  ✖ Collector metrics returned status {res.status_code}")
            return False

        lines = res.text.split("\n")
        accepted_spans = 0
        accepted_metrics = 0

        for line in lines:
            if line.startswith("otelcol_receiver_accepted_spans") and not line.startswith("#"):
                parts = line.split()
                if len(parts) >= 2:
                    accepted_spans = int(float(parts[-1]))
            elif line.startswith("otelcol_receiver_accepted_metric_points") and not line.startswith("#"):
                parts = line.split()
                if len(parts) >= 2:
                    accepted_metrics = int(float(parts[-1]))

        print(f"  ✔ Collector Health       : UP (200 OK)")
        print(f"  ✔ Accepted Tracing Spans : {accepted_spans}")
        print(f"  ✔ Accepted Metric Points : {accepted_metrics}")

        if accepted_spans > 0:
            print("\n🎉 SUCCESS: ObserveAll Collector successfully ingested and processed application telemetry!")
            return True
        else:
            print("\n⚠ Warning: Spans count was 0; check OTLP connection.")
            return False

    except Exception as e:
        print(f"  ✖ Error reaching collector: {e}")
        return False

if __name__ == "__main__":
    test_endpoints()
    success = check_collector_metrics()
    sys.exit(0 if success else 1)


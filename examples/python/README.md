# Python Auto-Instrumentation with ObserveAll

## Prerequisites
- Python 3.8+ application
- ObserveAll collector running locally

## Quick Start

### 1. Install OTel packages
```bash
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install   # auto-detects and installs library instrumentations
```

### 2. Run your app
```bash
OTEL_SERVICE_NAME=my-python-service \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
OTEL_EXPORTER_OTLP_PROTOCOL=grpc \
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=dev \
opentelemetry-instrument python app.py
```

### 3. For FastAPI / uvicorn
```bash
OTEL_SERVICE_NAME=my-fastapi-service \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
opentelemetry-instrument uvicorn app:app --host 0.0.0.0 --port 8080
```

### 4. Manual span example
```python
from opentelemetry import trace

tracer = trace.get_tracer("my-instrumentation")

with tracer.start_as_current_span("my-operation") as span:
    span.set_attribute("user.id", user_id)
    span.set_attribute("operation.type", "purchase")
    # your business logic
```

### 5. Custom metric example
```python
from opentelemetry import metrics

meter = metrics.get_meter("my-app")
request_counter = meter.create_counter(
    "http.requests.total",
    description="Total HTTP requests",
)
request_counter.add(1, {"method": "GET", "status": "200"})
```

## What gets auto-instrumented?
- Django, Flask, FastAPI, Starlette
- SQLAlchemy, psycopg2, pymysql
- Redis, MongoDB, Elasticsearch
- Celery, Kafka, boto3 (AWS)
- httpx, requests, urllib3, aiohttp

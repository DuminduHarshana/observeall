# Node.js Auto-Instrumentation with ObserveAll

## Prerequisites
- Node.js 14+ application
- ObserveAll collector running locally

## Quick Start

### 1. Install packages
```bash
npm install @opentelemetry/auto-instrumentations-node \
            @opentelemetry/sdk-node \
            @opentelemetry/exporter-trace-otlp-grpc \
            @opentelemetry/exporter-metrics-otlp-grpc
```

### 2. Zero-code auto-instrument
```bash
OTEL_SERVICE_NAME=my-node-service \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=dev \
node --require @opentelemetry/auto-instrumentations-node/register app.js
```

### 3. SDK setup (tracing.js — recommended for production)
```javascript
// tracing.js
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-grpc');
const { OTLPMetricExporter } = require('@opentelemetry/exporter-metrics-otlp-grpc');
const { PeriodicExportingMetricReader } = require('@opentelemetry/sdk-metrics');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { Resource } = require('@opentelemetry/resources');
const { SemanticResourceAttributes } = require('@opentelemetry/semantic-conventions');

const sdk = new NodeSDK({
  resource: new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: 'my-node-service',
    [SemanticResourceAttributes.DEPLOYMENT_ENVIRONMENT]: process.env.NODE_ENV || 'dev',
  }),
  traceExporter: new OTLPTraceExporter({
    url: 'http://localhost:4317',
  }),
  metricReader: new PeriodicExportingMetricReader({
    exporter: new OTLPMetricExporter({ url: 'http://localhost:4317' }),
    exportIntervalMillis: 10000,
  }),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();
process.on('SIGTERM', () => sdk.shutdown());
```

```bash
node --require ./tracing.js app.js
```

### 4. Manual span example
```javascript
const { trace } = require('@opentelemetry/api');

const tracer = trace.getTracer('my-instrumentation');
const span = tracer.startSpan('my-operation');
span.setAttribute('user.id', userId);
// ... your logic ...
span.end();
```

## What gets auto-instrumented?
- Express, Fastify, Koa, Hapi, NestJS
- HTTP/HTTPS, fetch
- MongoDB, PostgreSQL (pg), MySQL (mysql2)
- Redis (ioredis)
- gRPC, Kafka.js
- AWS SDK v3

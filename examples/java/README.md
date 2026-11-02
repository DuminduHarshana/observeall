# Java Auto-Instrumentation with ObserveAll

## Prerequisites
- Java 8+ application
- ObserveAll collector running locally (`docker compose up -d`)

## Quick Start

### 1. Download the OTel Java Agent
```bash
wget https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
```

### 2. Run your app with the agent attached
```bash
java \
  -javaagent:opentelemetry-javaagent.jar \
  -Dotel.service.name=my-java-service \
  -Dotel.exporter.otlp.endpoint=http://localhost:4317 \
  -Dotel.exporter.otlp.protocol=grpc \
  -Dotel.resource.attributes=deployment.environment=dev,team=platform \
  -jar your-app.jar
```

### 3. For Spring Boot (application.properties)
```properties
otel.service.name=my-spring-service
otel.exporter.otlp.endpoint=http://localhost:4317
otel.exporter.otlp.protocol=grpc
otel.resource.attributes=deployment.environment=dev
```

### 4. Maven dependency (for manual spans)
```xml
<dependency>
  <groupId>io.opentelemetry</groupId>
  <artifactId>opentelemetry-api</artifactId>
  <version>1.36.0</version>
</dependency>
```

### 5. Adding a custom span (Java)
```java
import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.api.trace.Span;

Tracer tracer = GlobalOpenTelemetry.getTracer("my-instrumentation");
Span span = tracer.spanBuilder("my-custom-operation").startSpan();
try (var scope = span.makeCurrent()) {
    // your business logic
} finally {
    span.end();
}
```

## What gets auto-instrumented?
- HTTP clients & servers (Spring MVC, Jetty, Tomcat, Netty)
- JDBC (MySQL, PostgreSQL, Oracle, etc.)
- gRPC
- Kafka, RabbitMQ
- Redis (Jedis, Lettuce)
- MongoDB
- AWS SDK, Google Cloud SDK
- And 100+ more libraries

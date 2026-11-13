<div align="center">

# 🔭 ObserveAll

**Universal OpenTelemetry Observability Platform**

*One agent. Any stack. Any backend.*

[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Standard-blue?logo=opentelemetry)](https://opentelemetry.io)
[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev)
[![Helm](https://img.shields.io/badge/Helm-v3_Chart-0F1689?logo=helm)](./helm/observeall)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)
[![Build & Tests](https://img.shields.io/badge/Tests-84%25_Coverage_Passing-success)](./internal/config)

</div>

---

## 📑 Table of Contents
- [What is ObserveAll?](#what-is-observeall)
- [How Does It Work?](#how-does-it-work)
  - [1. High-Level Architecture](#1-high-level-architecture)
  - [2. End-to-End Signal Lifecycle](#2-end-to-end-signal-lifecycle)
  - [3. The Collector Pipeline Anatomy](#3-the-collector-pipeline-anatomy)
- [Key Features](#-key-features)
- [Supported Environments & Vendors](#-supported-environments--vendors)
- [Project Layout](#-project-layout)
- [Quick Start Guide](#-quick-start-guide)
  - [1. Build or Install the CLI](#1-build-or-install-the-cli)
  - [2. Initialize Your Project](#2-initialize-your-project)
  - [3. Start the Local Stack](#3-start-the-local-dev-stack)
  - [4. Instrument Your Application](#4-instrument-your-application)
  - [5. Connect Vendor Backends](#5-connect-vendor-backends)
  - [6. Inspect Health & Metrics](#6-inspect-health--metrics)
- [Kubernetes Deployment](#-kubernetes-deployment)
  - [Helm Chart Installation](#helm-chart-installation)
  - [Zero-Code Injection via OTel Operator](#zero-code-injection-via-otel-operator)
- [Configuration Reference](#-configuration-reference)
- [Development & Testing](#-development--testing)
- [Roadmap](#-roadmap)

---

## What is ObserveAll?

Every observability vendor (Datadog, Dynatrace, New Relic, Grafana Cloud) traditionally requires installing their proprietary agent inside your servers or containers. This causes:
1. **Vendor Lock-in**: Switching vendors means rewriting application code and replacing daemonsets across entire clusters.
2. **Redundant Agent Overhead**: Running multiple agents wastes memory and CPU.
3. **Double Instrumentation**: Teams wanting open Prometheus metrics and commercial APM traces must instrument their code twice.

**ObserveAll solves this by adopting OpenTelemetry (OTel) as the universal standard**:
- Applications are instrumented **once** using open-source OTel SDKs or zero-code auto-agents.
- All signals (Traces, Metrics, Logs) stream into the **ObserveAll Collector**.
- The Collector enriches, batches, and intelligently filters the data before **fanning out to any combination of backends** (Prometheus, Grafana, Datadog, Dynatrace, New Relic, etc.) simultaneously.

```
       ┌────────────────────────────────────────────────────────┐
       │                   Your Applications                    │
       │     (Java, Python, Node.js, Go, .NET, Ruby, etc.)      │
       └──────────────────────────┬─────────────────────────────┘
                                  │  Zero-Code Auto-Agent
                                  │  (OTLP standard / gRPC / HTTP)
                                  ▼
       ┌────────────────────────────────────────────────────────┐
       │             ObserveAll Collector Pipeline              │
       │                                                        │
       │   [Receivers] ──► [Processors] ──► [Fan-Out Exporters] │
       │    • OTLP          • Memory Limiter   • Prometheus     │
       │    • Prometheus    • Tail-Sampler     • Datadog        │
       │                      Batch / Enrich   • Dynatrace      │
       │                                       • Grafana Cloud  │
       │                                       • New Relic      │
       └──────────────────────────┬─────────────────────────────┘
                                  │
         ┌────────────────────────┼────────────────────────┐
         ▼                        ▼                        ▼
  ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
  │ Prometheus  │          │   Datadog   │          │  Dynatrace  │
  │  + Grafana  │          │   APM/Logs  │          │    Cloud    │
  └─────────────┘          └─────────────┘          └─────────────┘
```

---

## How Does It Work?

### 1. High-Level Architecture

ObserveAll is comprised of three core layers:

1. **Instrumentation Layer (Zero-Code / SDKs)**:
   - Injected into your processes via language runtime flags (e.g. `-javaagent`, `python -m opentelemetry-instrument`, `node --require`).
   - Hooks into HTTP routers, gRPC clients, database drivers (PostgreSQL, MySQL, MongoDB), caches (Redis), and message queues (Kafka, RabbitMQ) automatically.
   - Emits signals using the standard **OTLP** (OpenTelemetry Protocol) over port `4317` (gRPC) or `4318` (HTTP/Protobuf).

2. **Control & Pipeline Layer (ObserveAll Collector)**:
   - A high-performance daemon built on the OpenTelemetry Collector architecture.
   - Configured dynamically by the `observeall` CLI or Helm charts.
   - Implements **Tail-Based Sampling**: waits for a trace to complete across all microservices before deciding whether to keep it (ensuring 100% of errors and slow requests are captured while dropping repetitive 200 OK traces to save costs).

3. **Routing & Fan-out Exporters**:
   - Converts standard OTLP data models on the fly into vendor-native APIs:
     - Formats metrics into Prometheus Remote-Write protobuf.
     - Authenticates and packages traces for Datadog Intake API.
     - Forwards traces, metrics, and logs to Dynatrace via secure OTLP tokens.
     - Exports to Grafana Cloud Loki, Tempo, and Mimir via unified OTLP endpoints.

---

### 2. End-to-End Signal Lifecycle

Here is what happens when a user calls an API endpoint:

```
[User Browser]
      │
      ▼ (HTTP GET /checkout)
[Spring Boot / Express / FastAPI Application]
      │
      ├── 1. OTel Agent creates a root span: 'HTTP GET /checkout'
      ├── 2. App queries Database (e.g., PostgreSQL):
      │        Agent injects child span: 'SELECT * FROM cart' with duration & SQL statement
      ├── 3. App makes external HTTP call to payment service:
      │        Agent injects W3C TraceContext headers ('traceparent: 00-4bf92f...') into request
      └── 4. Request finishes (e.g., HTTP 500 error):
               Span marked as ERROR, exception stacktrace attached
      │
      ▼ OTLP Export (gRPC port 4317)
[ObserveAll Collector]
      │
      ├── Receiver: Validates incoming OTLP payload
      ├── Memory Limiter: Ensures collector stays within memory boundaries (e.g., 512 MiB)
      ├── Resource Enricher: Adds cluster, environment, and version tags
      ├── Tail Sampler:
      │     Inspects trace status ──► Trace has ERROR ──► KEEP 100%
      │     (If trace was 200 OK < 50ms ──► Sample at 10% rate)
      └── Batch Processor: Aggregates records into 5-second or 1024-span batches
      │
      ▼ Multi-Backend Fan-Out
   ┌──┴────────────────────────┬─────────────────────────┐
   ▼                           ▼                         ▼
[Prometheus / Grafana]     [Datadog]                [Dynatrace]
• Rate, error, duration    • Full flame-graph trace  • Root-cause analysis
• Service health dashboard • Correlated host metrics • Davis AI anomaly detection
```

---

### 3. The Collector Pipeline Anatomy

The collector executes a pipeline composed of **Receivers**, **Processors**, and **Exporters**:

```yaml
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, tail_sampling, batch, resource]
      exporters: [datadog, otlp/dynatrace, otlp/jaeger, debug]
    metrics:
      receivers: [otlp, prometheus]
      processors: [memory_limiter, batch, resource]
      exporters: [prometheusremotewrite, otlp/grafana, debug]
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch, resource]
      exporters: [datadog, debug]
```

- **Receivers**: Listen for signals (`otlp` on 4317/4318, Prometheus pull on 8888).
- **Processors**: Operate in order — protect memory, sample smart, batch signals, inject global attributes.
- **Exporters**: Dispatch data over TLS to your chosen backend APIs in parallel.

---

## ✨ Key Features

- **Vendor Freedom**: Switch from Datadog to Grafana Cloud, or run both concurrently during a migration with zero app downtime or code edits.
- **Zero-Code Instrumentation**: Native agent support for Java, Python, Node.js, Go, and .NET.
- **Tail-Based Sampling**: Stop paying for millions of unneeded health-check traces; preserve 100% of errors and latency spikes.
- **Kubernetes Auto-Injection**: One annotation on a K8s namespace automatically injects and configures OTel agents in all application pods.
- **Cost Reduction**: Scrub unwanted high-cardinality tags and filter signals before sending to billable SaaS backends.
- **Unified Developer CLI**: Scaffold projects, add backends, and verify pipeline telemetry in seconds.

---

## 🌐 Supported Environments & Vendors

### Runtimes & Environments
| Runtime / Host | Instrumentation Method | Supported Signals |
|---|---|---|
| **Java 8+ / Spring Boot** | `opentelemetry-javaagent.jar` | Traces, Metrics, Logs |
| **Python 3.8+ (FastAPI, Django, Flask)** | `opentelemetry-instrument` | Traces, Metrics, Logs |
| **Node.js (Express, Nest, Fastify)** | `@opentelemetry/auto-instrumentations-node` | Traces, Metrics, Logs |
| **Go** | OTel Go SDK | Traces, Metrics |
| **.NET / C#** | OTel .NET Automatic Instrumentation | Traces, Metrics, Logs |
| **Kubernetes (EKS, GKE, AKS, Kind)** | ObserveAll Helm Chart + OTel Operator | All signals |
| **Docker / Bare Metal** | ObserveAll Docker Compose / Standalone Binary | All signals |

### Supported Observability Backends
| Backend | Protocol Used | Telemetry Supported |
|---|---|---|
| **Prometheus** | Prometheus Remote-Write | Metrics |
| **Grafana Cloud (Mimir, Tempo, Loki)** | OTLP (HTTP/gRPC) | Traces, Metrics, Logs |
| **Datadog** | Datadog Intake API | Traces, Metrics, Logs |
| **Dynatrace** | Dynatrace OTLP API v2 | Traces, Metrics, Logs |
| **New Relic** | New Relic OTLP API | Traces, Metrics, Logs |
| **Jaeger** | OTLP / gRPC | Traces |
| **Custom / Internal Gateway** | Standard OTLP (gRPC / HTTP) | Traces, Metrics, Logs |

---

## 📁 Project Layout

```
observeall/
├── cmd/observeall/                 # ObserveAll CLI application
│   └── main.go                     # Entrypoint
├── internal/
│   ├── cli/                        # Cobra commands (init, connect, generate, status)
│   │   ├── root.go                 # Root command & banner
│   │   ├── init_cmd.go             # Interactive setup wizard
│   │   ├── connect.go              # Backend credential connector
│   │   ├── generate.go             # Re-generates collector-config.yaml
│   │   └── status.go               # Collector health probe
│   └── config/                     # Configuration engine & OTel YAML generator
│       ├── types.go                # Project and Backend data structures
│       ├── generator.go            # Pipeline YAML generator
│       └── generator_test.go       # 84%+ statement coverage test suite
├── helm/observeall/                # Production-grade Kubernetes Helm Chart
│   ├── Chart.yaml                  # Chart metadata & OTel subchart dependencies
│   ├── values.yaml                 # Configuration knobs (sampling, backends, memory)
│   └── templates/
│       ├── deployment.yaml         # Gateway Collector Deployment
│       ├── daemonset.yaml          # Edge Node Collector DaemonSet
│       ├── configmap.yaml          # Collector YAML pipeline ConfigMap
│       ├── service.yaml            # ClusterIP Service for OTLP gRPC/HTTP
│       ├── hpa.yaml                # Horizontal Pod Autoscaler (CPU based)
│       ├── rbac.yaml               # ClusterRole for Kubernetes metadata enrichment
│       ├── serviceaccount.yaml     # Dedicated ServiceAccount
│       ├── servicemonitor.yaml     # Prometheus Operator ServiceMonitor
│       └── instrumentation.yaml    # OTel Operator Auto-Injection CRD
├── deploy/
│   ├── prometheus.yml              # Scrape configuration for local Prometheus
│   ├── grafana/provisioning/       # Auto-provisioned datasources (Prometheus, Jaeger)
│   └── k8s/                        # Standalone K8s secret templates & manifests
├── examples/                       # Step-by-step guides for instrumenting apps
│   ├── java/README.md              # Java auto-instrumentation guide
│   ├── python/README.md            # Python auto-instrumentation guide
│   └── node/README.md              # Node.js auto-instrumentation guide
├── .github/workflows/ci.yml        # CI Pipeline (Go tests, Helm lint, Docker build)
├── Dockerfile                      # Custom OTel Collector distribution
├── Makefile                        # Build, test, cross-compile, dev automation
└── README.md                       # Documentation
```

---

## 🚀 Quick Start Guide

### 1. Build or Install the CLI

```bash
cd observeall

# Build binary
make build

# Optional: Install to system PATH
make install
```

The compiled binary will be placed at `./bin/observeall`.

---

### 2. Initialize Your Project

Run the interactive wizard:
```bash
./bin/observeall init
```

The wizard prompts you for:
1. **Project Name** (e.g. `ecommerce-core`)
2. **Environment** (`kubernetes`, `docker-compose`, `bare-metal`, or `serverless`)
3. **Primary Language** (`java`, `python`, `nodejs`, `go`, etc.)
4. **Target Backend** (`datadog`, `dynatrace`, `prometheus+grafana`, `newrelic`, etc.)
5. **Signals** (`traces+metrics+logs`, `traces+metrics`, etc.)
6. **Sampling Strategy** (`tail-sampling`, `head-sampling`, `always-on`)

It outputs:
- `observeall.yaml`: The high-level configuration manifest.
- `collector-config.yaml`: The production-ready OpenTelemetry Collector YAML pipeline.
- `docker-compose.yaml`: Local dev stack (if Docker Compose was chosen).

---

### 3. Start the Local Dev Stack

Launch the local observability environment (Collector + Prometheus + Grafana + Jaeger):
```bash
make local-dev
# or: docker compose up -d
```

| Service | Access URL | Default Credentials |
|---|---|---|
| **Grafana Dashboard** | http://localhost:3000 | Anonymous Admin enabled |
| **Prometheus UI** | http://localhost:9090 | *None* |
| **Jaeger Trace UI** | http://localhost:16686 | *None* |
| **OTel Collector (gRPC)** | `localhost:4317` | Standard OTLP endpoint |
| **OTel Collector (HTTP)** | `localhost:4318` | Standard OTLP endpoint |
| **Collector Health/Metrics** | http://localhost:8888/metrics | Internal stats |

---

### 4. Instrument Your Application

Point your application to `localhost:4317` without modifying source code:

#### Java / Spring Boot
```bash
wget https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar

java -javaagent:opentelemetry-javaagent.jar \
     -Dotel.service.name=my-java-service \
     -Dotel.exporter.otlp.endpoint=http://localhost:4317 \
     -jar target/app.jar
```

#### Python (FastAPI, Flask, Django)
```bash
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install

OTEL_SERVICE_NAME=my-python-service \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
opentelemetry-instrument python app.py
```

#### Node.js (Express, NestJS)
```bash
npm install @opentelemetry/auto-instrumentations-node

OTEL_SERVICE_NAME=my-node-service \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
node --require @opentelemetry/auto-instrumentations-node/register app.js
```

---

### 5. Connect Vendor Backends

Want to export the same telemetry data to Datadog or Dynatrace alongside Prometheus? Simply run:

```bash
# Connect Datadog
./bin/observeall connect --backend datadog

# Connect Dynatrace
./bin/observeall connect --backend dynatrace

# Connect Grafana Cloud
./bin/observeall connect --backend grafana
```

The CLI securely updates `observeall.yaml` and regenerates `collector-config.yaml` with the proper exporter authentication and pipeline wiring.

---

### 6. Inspect Health & Metrics

Verify your collector pipeline health at any time:
```bash
./bin/observeall status
```

Output:
```
ObserveAll Pipeline Status
─────────────────────────────
  Project   : ecommerce-core
  Env       : production
  Language  : java
  Backends  : [datadog prometheus dynatrace]
  Signals   : [traces metrics logs]
  Sampling  : tail
─────────────────────────────
  Collector : ✔ healthy (http://localhost:8888/metrics)
```

---

## ☸️ Kubernetes Deployment

### Helm Chart Installation

The included Helm chart provides a resilient, auto-scaling deployment in Kubernetes:

```bash
# 1. Create secret with vendor credentials
kubectl create namespace observeall
kubectl create secret generic observeall-backend-secrets \
  --from-literal=DD_API_KEY="your-datadog-key" \
  --from-literal=DT_API_TOKEN="your-dynatrace-token" \
  --from-literal=DT_ENDPOINT="https://abc.live.dynatrace.com" \
  -n observeall

# 2. Deploy ObserveAll
helm upgrade --install observeall ./helm/observeall \
  --namespace observeall \
  --set global.projectName="cloud-production" \
  --set backends.datadog.enabled=true \
  --set backends.dynatrace.enabled=true \
  --set collector.autoscaling.enabled=true
```

### Zero-Code Injection via OTel Operator

If your cluster has the [OpenTelemetry Operator](https://github.com/open-telemetry/opentelemetry-operator) installed, enable the operator integration in `values.yaml`:
```yaml
operator:
  enabled: true
```

Then annotate any target application namespace:
```bash
# Auto-inject Java agent into all pods in the namespace
kubectl annotate namespace payments-ns \
  instrumentation.opentelemetry.io/inject-java=observeall/observeall-instrumentation

# Auto-inject Python agent into all pods in the namespace
kubectl annotate namespace recommendation-ns \
  instrumentation.opentelemetry.io/inject-python=observeall/observeall-instrumentation
```

The operator mutates the pods on startup, attaches the agent in memory, and connects them directly to the ObserveAll collector service. **Zero application changes required.**

---

## ⚙️ Configuration Reference

### `observeall.yaml`
```yaml
name: my-app                      # Project identifier added to all telemetry tags
environment: production           # Environment tag (prod, staging, dev)
language: java                    # Primary language
signals:                          # Which pipelines to activate
  - traces
  - metrics
  - logs
sampling: tail                    # tail | head | always_on | none
backends:                         # Active backend exporters
  - prometheus
  - datadog
backend_configs:
  datadog:
    api_key: ${DD_API_KEY}
    site: datadoghq.com
```

### Tail-Sampling Rules
When `sampling: tail` is specified, the collector enforces three rules out of the box:
1. **Error Traces**: 100% retained (HTTP 5xx, uncaught exceptions).
2. **Slow Traces**: 100% retained when request latency exceeds 500ms.
3. **Normal Traces**: 10% probabilistic fallback to prevent metric skew while optimizing storage costs.

---

## 🛠️ Development & Testing

```bash
# Run tests with race detection and coverage
make test

# Build multi-platform binaries (Linux, macOS, Windows)
make build-all

# Lint the Helm chart
make helm-lint

# Dry-run template generation
make helm-template
```

---

## 🗺️ Roadmap

- [x] **Phase 1: Foundation**
  - [x] OpenTelemetry Collector baseline with dynamic YAML generation
  - [x] Multi-backend exporter fan-out (Prometheus, Grafana, Datadog, Dynatrace, New Relic, Jaeger)
  - [x] Developer CLI (`init`, `connect`, `status`, `generate`)
  - [x] Production Helm chart with HPA, RBAC, DaemonSet, and OTel Operator CRDs
  - [x] Docker Compose local stack with Prometheus, Grafana, and Jaeger
  - [x] Unit test suite with 84%+ statement coverage & GitHub Actions CI
- [x] **Phase 2: SaaS Control Plane**
  - [x] Web UI for visual drag-and-drop routing pipelines (`observeall serve`)
  - [x] Real-time ingestion rate & cost breakdown per service (`observeall analytics`)
  - [x] Dynamic GitOps push for collector configuration updates (`observeall watch` / `sync`)
- [x] **Phase 3: Security & Governance**
  - [x] Automatic PII scrubbing processor (`observeall pii --enable`)
  - [x] Zero-code eBPF network instrumentation manifests (`observeall ebpf`)
- [x] **Phase 4: AIOps**
  - [x] Cross-signal automated root-cause analysis (LLM assisted) (`observeall aiops`)

---

## 📜 License

ObserveAll is licensed under the **Apache License 2.0**. See the [LICENSE](LICENSE) file for details.

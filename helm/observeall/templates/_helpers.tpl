{{/*
Expand the name of the chart.
*/}}
{{- define "observeall.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "observeall.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "observeall.labels" -}}
helm.sh/chart: {{ include "observeall.chart" . }}
{{ include "observeall.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: observability
observeall.io/project: {{ .Values.global.projectName | quote }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "observeall.selectorLabels" -}}
app.kubernetes.io/name: {{ include "observeall.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Chart label
*/}}
{{- define "observeall.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
ServiceAccount name
*/}}
{{- define "observeall.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "observeall.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Collector config — builds the full OTel Collector YAML
*/}}
{{- define "observeall.collectorConfig" -}}
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
  prometheus:
    config:
      scrape_configs:
        - job_name: otel-collector
          scrape_interval: 10s
          static_configs:
            - targets: ['0.0.0.0:8888']

processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 512
    spike_limit_mib: 128
  batch:
    timeout: 5s
    send_batch_size: 1024
  resource:
    attributes:
      - key: deployment.environment
        value: {{ .Values.global.environment | quote }}
        action: upsert
      - key: observeall.project
        value: {{ .Values.global.projectName | quote }}
        action: upsert
{{- if eq .Values.collector.sampling.strategy "tail" }}
  tail_sampling:
    decision_wait: {{ .Values.collector.sampling.tailSampling.decisionWait }}
    num_traces: {{ .Values.collector.sampling.tailSampling.numTraces }}
    policies:
{{- if .Values.collector.sampling.tailSampling.policies.errorPolicy }}
      - name: errors-policy
        type: status_code
        status_code:
          status_codes: [ERROR]
{{- end }}
{{- if .Values.collector.sampling.tailSampling.policies.slowTraceMs }}
      - name: slow-traces-policy
        type: latency
        latency:
          threshold_ms: {{ .Values.collector.sampling.tailSampling.policies.slowTraceMs }}
{{- end }}
      - name: probabilistic-policy
        type: probabilistic
        probabilistic:
          sampling_percentage: {{ .Values.collector.sampling.tailSampling.policies.probabilisticPct }}
{{- end }}

exporters:
{{- if .Values.backends.prometheus.enabled }}
  prometheusremotewrite:
    endpoint: {{ .Values.backends.prometheus.endpoint }}
    tls:
      insecure: {{ .Values.backends.prometheus.tls.insecure }}
{{- end }}
{{- if .Values.backends.grafana.enabled }}
  otlp/grafana:
    endpoint: {{ .Values.backends.grafana.endpoint }}
    headers:
      Authorization: "Basic ${GRAFANA_TOKEN}"
{{- end }}
{{- if .Values.backends.datadog.enabled }}
  datadog:
    api:
      key: "${DD_API_KEY}"
      site: {{ .Values.backends.datadog.site }}
{{- end }}
{{- if .Values.backends.dynatrace.enabled }}
  otlp/dynatrace:
    endpoint: {{ .Values.backends.dynatrace.endpoint }}
    headers:
      Authorization: "Api-Token ${DT_API_TOKEN}"
{{- end }}
{{- if .Values.backends.newrelic.enabled }}
  otlp/newrelic:
    endpoint: {{ .Values.backends.newrelic.endpoint }}
    headers:
      api-key: "${NR_LICENSE_KEY}"
{{- end }}
{{- if .Values.backends.jaeger.enabled }}
  otlp/jaeger:
    endpoint: {{ .Values.backends.jaeger.endpoint }}
    tls:
      insecure: {{ .Values.backends.jaeger.tls.insecure }}
{{- end }}
{{- if .Values.backends.customOtlp.enabled }}
  otlp/custom:
    endpoint: {{ .Values.backends.customOtlp.endpoint }}
    tls:
      insecure: true
{{- end }}
  debug:
    verbosity: basic

service:
  pipelines:
{{- if .Values.collector.pipelines.traces.enabled }}
    traces:
      receivers: [otlp]
      processors: [memory_limiter, {{- if eq .Values.collector.sampling.strategy "tail" }} tail_sampling,{{- end }} batch, resource]
      exporters: [{{- include "observeall.activeExporters" . -}}, debug]
{{- end }}
{{- if .Values.collector.pipelines.metrics.enabled }}
    metrics:
      receivers: [otlp, prometheus]
      processors: [memory_limiter, batch, resource]
      exporters: [{{- include "observeall.activeExporters" . -}}, debug]
{{- end }}
{{- if .Values.collector.pipelines.logs.enabled }}
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch, resource]
      exporters: [{{- include "observeall.activeExporters" . -}}, debug]
{{- end }}
  telemetry:
    logs:
      level: info
    metrics:
      address: 0.0.0.0:8888
{{- end }}

{{/*
Build comma-separated list of active exporters
*/}}
{{- define "observeall.activeExporters" -}}
{{- $exporters := list -}}
{{- if .Values.backends.prometheus.enabled }}{{- $exporters = append $exporters "prometheusremotewrite" }}{{- end -}}
{{- if .Values.backends.grafana.enabled }}{{- $exporters = append $exporters "otlp/grafana" }}{{- end -}}
{{- if .Values.backends.datadog.enabled }}{{- $exporters = append $exporters "datadog" }}{{- end -}}
{{- if .Values.backends.dynatrace.enabled }}{{- $exporters = append $exporters "otlp/dynatrace" }}{{- end -}}
{{- if .Values.backends.newrelic.enabled }}{{- $exporters = append $exporters "otlp/newrelic" }}{{- end -}}
{{- if .Values.backends.jaeger.enabled }}{{- $exporters = append $exporters "otlp/jaeger" }}{{- end -}}
{{- if .Values.backends.customOtlp.enabled }}{{- $exporters = append $exporters "otlp/custom" }}{{- end -}}
{{- join ", " $exporters -}}
{{- end }}

# Observability Guide

This guide explains how to use distributed tracing with OpenTelemetry in the Zenith Operator to visualize complete request flows across multiple functions.

## Overview

The Zenith Operator provides automatic distributed tracing integration using OpenTelemetry and the OTLP protocol. When enabled, the operator automatically:

- Injects OpenTelemetry environment variables into function containers
- Configures Dapr (if enabled) to propagate W3C Trace Context headers
- Sends trace spans to an OpenTelemetry Collector

This enables developers to visualize the complete request flow (trace waterfall) across multiple functions, making it easy to debug issues like "Error 500" when Function A calls Function B that fails.

## Architecture

### Components

1. **OpenTelemetry Operator**: Manages the lifecycle of OpenTelemetry Collectors
2. **OpenTelemetry Collector**: Receives trace spans via OTLP protocol and exports them to observability backends
3. **Function Containers**: Automatically configured with OTEL environment variables when tracing is enabled
4. **Dapr (optional)**: Configured to propagate trace context between functions

### Trace Flow

```
Function A (with OTEL SDK)
  ↓ HTTP request with W3C Trace Context headers
Function B (with OTEL SDK)
  ↓ Trace spans sent via OTLP
OpenTelemetry Collector
  ↓ Export to backend
Observability Backend (Jaeger, Tempo, etc.)
```

## Enabling Tracing

### Basic Configuration

To enable tracing for a function, add the `observability.tracing.enabled` field to your Function spec:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-function
spec:
  gitRepo: https://github.com/myorg/my-function
  gitRevision: main
  build:
    image: registry.example.com/my-function:latest
  observability:
    tracing:
      enabled: true
```

### With Custom Sampling Rate

You can configure a custom sampling rate (0.0 to 1.0) to control what percentage of traces are collected:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-function
spec:
  gitRepo: https://github.com/myorg/my-function
  gitRevision: main
  build:
    image: registry.example.com/my-function:latest
  observability:
    tracing:
      enabled: true
      samplingRate: "0.5"  # Sample 50% of traces
```

### With Dapr

When both Dapr and tracing are enabled, the operator automatically configures Dapr to propagate trace context:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-function
spec:
  gitRepo: https://github.com/myorg/my-function
  gitRevision: main
  build:
    image: registry.example.com/my-function:latest
  deploy:
    dapr:
      enabled: true
      appID: my-function
      appPort: 8080
  observability:
    tracing:
      enabled: true
```

## Environment Variables

When tracing is enabled, the operator automatically injects the following environment variables into your function containers:

| Variable | Value | Description |
|----------|-------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://otel-collector.opentelemetry-operator-system.svc.cluster.local:4317` | OTLP endpoint for sending traces |
| `OTEL_SERVICE_NAME` | `<function-name>` | Service name for identifying the function in traces |
| `OTEL_RESOURCE_ATTRIBUTES` | `service.namespace=<namespace>,service.version=latest` | Additional resource attributes |
| `OTEL_TRACES_EXPORTER` | `otlp` | Trace exporter protocol |
| `OTEL_TRACES_SAMPLER` | `traceidratio` | Sampler type (only if samplingRate is set) |
| `OTEL_TRACES_SAMPLER_ARG` | `<samplingRate>` | Sampler argument (only if samplingRate is set) |

## Instrumenting Your Application

To send traces, your application code needs to use an OpenTelemetry SDK. The operator provides the configuration via environment variables, but you need to initialize the SDK in your code.

### Go Example

```go
package main

import (
    "context"
    "log"
    "net/http"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
    // Initialize OTEL tracer (reads config from environment variables)
    ctx := context.Background()
    exporter, err := otlptracegrpc.New(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
    )
    otel.SetTracerProvider(tp)
    
    // Wrap HTTP handlers with OTEL instrumentation
    http.Handle("/", otelhttp.NewHandler(http.HandlerFunc(handler), "my-function"))
    http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Your handler code here
    w.Write([]byte("Hello World"))
}
```

### Python Example

```python
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from flask import Flask

# Initialize OTEL tracer (reads config from environment variables)
trace.set_tracer_provider(TracerProvider())
tracer = trace.get_tracer(__name__)

otlp_exporter = OTLPSpanExporter()
span_processor = BatchSpanProcessor(otlp_exporter)
trace.get_tracer_provider().add_span_processor(span_processor)

# Create Flask app with automatic instrumentation
app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)

@app.route('/')
def hello():
    return 'Hello World'

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
```

### Node.js Example

```javascript
const { NodeTracerProvider } = require('@opentelemetry/sdk-trace-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-grpc');
const { BatchSpanProcessor } = require('@opentelemetry/sdk-trace-base');
const { registerInstrumentations } = require('@opentelemetry/instrumentation');
const { HttpInstrumentation } = require('@opentelemetry/instrumentation-http');
const express = require('express');

// Initialize OTEL tracer (reads config from environment variables)
const provider = new NodeTracerProvider();
const exporter = new OTLPTraceExporter();
provider.addSpanProcessor(new BatchSpanProcessor(exporter));
provider.register();

// Register HTTP instrumentation
registerInstrumentations({
  instrumentations: [new HttpInstrumentation()],
});

// Create Express app
const app = express();

app.get('/', (req, res) => {
  res.send('Hello World');
});

app.listen(8080, () => {
  console.log('Server listening on port 8080');
});
```

## Integrating with Observability Backends

The OpenTelemetry Collector can export traces to various observability backends. By default, the collector is configured with a debug exporter that logs traces to stdout.

### Jaeger

To export traces to Jaeger, update the OpenTelemetry Collector configuration:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: otel-collector
  namespace: opentelemetry-operator-system
spec:
  mode: deployment
  config:
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    processors:
      batch: {}
    exporters:
      jaeger:
        endpoint: jaeger-collector.observability.svc.cluster.local:14250
        tls:
          insecure: true
    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [jaeger]
```

### Grafana Tempo

To export traces to Grafana Tempo:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: otel-collector
  namespace: opentelemetry-operator-system
spec:
  mode: deployment
  config:
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    processors:
      batch: {}
    exporters:
      otlp:
        endpoint: tempo.observability.svc.cluster.local:4317
        tls:
          insecure: true
    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [otlp]
```

## Debugging with Traces

### Viewing Trace Waterfalls

Once your functions are instrumented and sending traces, you can visualize the complete request flow in your observability backend:

1. **Jaeger UI**: Access the Jaeger UI and search for traces by service name (function name)
2. **Grafana**: Use the Tempo data source in Grafana to query and visualize traces
3. **Other backends**: Consult your observability backend's documentation for trace visualization

### Common Debugging Scenarios

#### Scenario 1: Function A calls Function B, which returns Error 500

Without tracing, you only see "Error 500" in Function A's logs. With tracing enabled:

1. Search for traces from Function A
2. View the trace waterfall showing the complete request flow
3. See that Function B took 5 seconds and returned 500
4. Click on Function B's span to see detailed error information
5. Identify the root cause (e.g., database timeout, missing configuration)

#### Scenario 2: Slow request across multiple functions

With tracing enabled:

1. Search for slow traces (e.g., > 1 second)
2. View the trace waterfall showing all function calls
3. Identify which function is the bottleneck
4. See the time spent in each function and external calls (database, API, etc.)

## Best Practices

### 1. Enable Tracing Selectively

Tracing adds overhead to your functions. Enable it only for functions that need debugging or monitoring:

- **Development**: Enable tracing with 100% sampling rate for all functions
- **Staging**: Enable tracing with 50-100% sampling rate for critical functions
- **Production**: Enable tracing with 1-10% sampling rate for critical functions

### 2. Use Appropriate Sampling Rates

- **100% sampling** (`samplingRate: "1.0"`): Capture all traces (use in development)
- **10% sampling** (`samplingRate: "0.1"`): Capture 10% of traces (use in production)
- **1% sampling** (`samplingRate: "0.01"`): Capture 1% of traces (use for high-traffic functions)

### 3. Add Custom Spans

Add custom spans in your application code to provide more detailed trace information:

```go
func processOrder(ctx context.Context, order Order) error {
    tracer := otel.Tracer("my-function")
    ctx, span := tracer.Start(ctx, "processOrder")
    defer span.End()
    
    // Add attributes to the span
    span.SetAttributes(
        attribute.String("order.id", order.ID),
        attribute.Int("order.amount", order.Amount),
    )
    
    // Your processing logic here
    return nil
}
```

### 4. Propagate Context

Always propagate context when making HTTP calls to other functions:

```go
// Create HTTP client with OTEL instrumentation
client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}

// Make request with context
req, _ := http.NewRequestWithContext(ctx, "GET", "http://other-function", nil)
resp, err := client.Do(req)
```

## Troubleshooting

### Traces Not Appearing

1. **Check OTEL environment variables**: Verify that the operator injected the correct environment variables:
   ```bash
   kubectl get ksvc my-function -o jsonpath='{.spec.template.spec.containers[0].env}'
   ```

2. **Check OpenTelemetry Collector**: Verify that the collector is running:
   ```bash
   kubectl get pods -n opentelemetry-operator-system
   kubectl logs -n opentelemetry-operator-system -l app.kubernetes.io/component=opentelemetry-collector
   ```

3. **Check application logs**: Verify that your application is initializing the OTEL SDK correctly

### High Overhead

If tracing is causing high overhead:

1. **Reduce sampling rate**: Lower the `samplingRate` value
2. **Use head-based sampling**: Configure the collector to use head-based sampling
3. **Optimize span creation**: Reduce the number of custom spans in your code

### Missing Spans

If some spans are missing from traces:

1. **Check context propagation**: Ensure you're propagating context correctly
2. **Check HTTP headers**: Verify that W3C Trace Context headers are being sent
3. **Check Dapr configuration**: If using Dapr, verify the `dapr.io/config` annotation is set

## Reference

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OTLP Specification](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [Dapr Distributed Tracing](https://docs.dapr.io/operations/observability/tracing/)

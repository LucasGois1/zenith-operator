# Creating Asynchronous Event Functions

This guide shows how to create functions that process events asynchronously using Knative Eventing and Zenith Operator.

## Overview

Event-driven functions are ideal for:
- Asynchronous data processing
- Event-driven workflows
- Notifications and alerts
- System integration
- Queue message processing

Zenith Operator automatically:
1. Clones your Git repository and builds the image
2. Deploys as a Knative Service
3. Creates a Knative Trigger to subscribe to events
4. Routes events from Broker to your function

## Prerequisites

- Kubernetes cluster with Zenith Operator installed
- Knative Eventing installed
- Knative Broker created in the namespace
- Git repository with function code
- Git authentication Secret (if private repository)

## Event-Driven Architecture

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Producer  │─────▶│   Broker    │─────▶│   Trigger   │─────▶│  Function   │
│  (System)   │      │  (default)  │      │  (filters)  │      │   (Yours)   │
└─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘
```

1. **Producer**: System that sends events (CloudEvents) to the Broker
2. **Broker**: Receives and distributes events
3. **Trigger**: Filters events based on attributes and routes to the function
4. **Function**: Your function that processes the events

## Function Code Structure

Your function must receive HTTP POST requests with events in CloudEvents format:

### Important: Response Format Requirements

When processing CloudEvents, your function **MUST** return one of the following:

1. **Empty response** (HTTP 200 with no body) - Recommended for most cases
2. **Valid CloudEvent response** - Only if you need to emit a new event

Returning any other response (like JSON) will cause errors in the Knative Eventing broker. The broker validates responses and expects either empty or CloudEvent format.

### Go Example

```go
package main

import (
    "io"
    "log"
    "net/http"
    "os"
    "strings"
)

func eventHandler(w http.ResponseWriter, r *http.Request) {
    // Log CloudEvent headers
    log.Printf("Received CloudEvent:")
    log.Printf("  Ce-Id: %s", r.Header.Get("Ce-Id"))
    log.Printf("  Ce-Type: %s", r.Header.Get("Ce-Type"))
    log.Printf("  Ce-Source: %s", r.Header.Get("Ce-Source"))
    log.Printf("  Ce-Specversion: %s", r.Header.Get("Ce-Specversion"))
    
    // Read event data from body
    if r.Body != nil {
        body, err := io.ReadAll(r.Body)
        if err == nil && len(body) > 0 {
            log.Printf("  Data: %s", string(body))
        }
    }
    
    // Your processing logic here
    processEvent(r)
    
    // IMPORTANT: Return empty response for CloudEvents
    // Do NOT return JSON - Knative Eventing expects empty or CloudEvent response
    log.Printf("CloudEvent processed successfully")
    w.WriteHeader(http.StatusOK)
}

func processEvent(r *http.Request) {
    // Implement your processing logic
    // Example: save to database, send notification, etc.
    eventType := r.Header.Get("Ce-Type")
    log.Printf("Processing event of type: %s", eventType)
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    http.HandleFunc("/", eventHandler)
    log.Printf("Event processor listening on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### Python Example

```python
from flask import Flask, request, Response
import logging
import os

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)

@app.route('/', methods=['POST'])
def event_handler():
    # CloudEvent headers are passed as HTTP headers
    ce_id = request.headers.get('Ce-Id')
    ce_type = request.headers.get('Ce-Type')
    ce_source = request.headers.get('Ce-Source')
    
    # Log event
    logging.info(f"Received CloudEvent: id={ce_id}, type={ce_type}, source={ce_source}")
    
    # Event data is in the body
    data = request.get_json(silent=True) or {}
    logging.info(f"Event data: {data}")
    
    # Process event
    process_event(ce_type, data)
    
    # IMPORTANT: Return empty response for CloudEvents
    logging.info("CloudEvent processed successfully")
    return Response(status=200)

def process_event(event_type, data):
    # Implement your processing logic
    logging.info(f"Processing event of type: {event_type}")
    # Your logic here

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8080))
    app.run(host='0.0.0.0', port=port)
```

### Node.js Example

```javascript
const express = require('express');
const app = express();

app.use(express.json());

app.post('/', (req, res) => {
    // CloudEvent headers are passed as HTTP headers
    const ceId = req.headers['ce-id'];
    const ceType = req.headers['ce-type'];
    const ceSource = req.headers['ce-source'];
    
    // Log event
    console.log(`Received CloudEvent: id=${ceId}, type=${ceType}, source=${ceSource}`);
    console.log(`Event data:`, req.body);
    
    // Process event
    processEvent(ceType, req.body);
    
    // IMPORTANT: Return empty response for CloudEvents
    console.log('CloudEvent processed successfully');
    res.status(200).end();
});

function processEvent(eventType, data) {
    // Implement your processing logic
    console.log(`Processing event of type: ${eventType}`);
    // Your logic here
}

const port = process.env.PORT || 8080;
app.listen(port, () => {
    console.log(`Event processor listening on port ${port}`);
});
```

## Step 1: Create Knative Broker

First, create a Broker in the namespace where your function will be deployed:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: eventing.knative.dev/v1
kind: Broker
metadata:
  name: default
  namespace: default
EOF
```

Verify if Broker is ready:

```bash
kubectl get broker default
```

## Step 2: Create Function Custom Resource with Eventing

Create a YAML file with the function definition including eventing configuration:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: order-processor
  namespace: default
spec:
  # Git repository with code
  gitRepo: https://github.com/myorg/order-processor
  gitRevision: main
  
  # Authentication secret (optional)
  gitAuthSecretName: github-auth
  
  # Build configuration
  build:
    image: registry.example.com/order-processor:latest
  
  # Deploy configuration
  deploy: {}
  
  # Eventing configuration
  eventing:
    # Broker name to subscribe to
    broker: default
    
    # Event filters (CloudEvents attributes)
    filters:
      type: com.example.order.created
      source: payment-service
```

Apply the resource:

```bash
kubectl apply -f order-processor.yaml
```

## Network Visibility for Event Functions

Event functions support the same visibility options as HTTP functions. By default, event functions are `cluster-local`, meaning they can only receive events from within the cluster.

### Cluster-Local (Default)

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: order-processor
spec:
  gitRepo: https://github.com/myorg/order-processor
  build:
    image: registry.example.com/order-processor:latest
  deploy:
    visibility: cluster-local  # Default - events from within cluster only
  eventing:
    broker: default
    filters:
      type: com.example.order.created
```

With `cluster-local` visibility:
- Events are delivered via the Knative Eventing Broker/Trigger mechanism
- The function URL is `http://{name}.{namespace}.svc.cluster.local`
- Events must be sent to the Broker from within the cluster

### External Visibility

If you need to also expose the function for direct HTTP access (in addition to event processing):

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: order-processor
spec:
  gitRepo: https://github.com/myorg/order-processor
  build:
    image: registry.example.com/order-processor:latest
  deploy:
    visibility: external  # Accessible externally AND via events
  eventing:
    broker: default
    filters:
      type: com.example.order.created
```

With `external` visibility:
- Events are still delivered via the Broker/Trigger mechanism
- The function is also accessible externally at `http://{name}.{namespace}.{domain}`
- Useful for functions that handle both events AND direct HTTP requests

### Accessing External Event Functions

**Linux:**
```bash
# Get the gateway IP
GATEWAY_IP=$(kubectl get svc -n envoy-gateway-system \
  -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway \
  -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')

# Access function directly (bypassing eventing)
curl -H "Host: order-processor.default.example.com" http://$GATEWAY_IP/
```

**MacOS (Docker Desktop / Colima):**

On MacOS, the LoadBalancer IP is not directly accessible. Use port-forwarding:

```bash
# Terminal 1: Start port-forward
kubectl port-forward -n envoy-gateway-system \
  $(kubectl get svc -n envoy-gateway-system \
    -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway \
    -o jsonpath='{.items[0].metadata.name}') 8080:80

# Terminal 2: Access the function
curl -H "Host: order-processor.default.example.com" http://localhost:8080/
```

### Sending Events to the Broker

Regardless of function visibility, events are always sent to the Broker. The Broker is only accessible from within the cluster:

```bash
# From within the cluster (e.g., from a pod)
curl http://broker-ingress.knative-eventing.svc.cluster.local/{namespace}/{broker-name} \
  -X POST \
  -H "Ce-Id: event-123" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: com.example.order.created" \
  -H "Ce-Source: my-service" \
  -H "Content-Type: application/json" \
  -d '{"orderId": "123"}'
```

To send events from outside the cluster, you can create an "Event Gateway" function - see the section below.

### Sending Events from Outside the Cluster (Event Gateway Pattern)

Since the Knative Eventing Broker is only accessible from within the cluster, you need to create an intermediary service that:
1. Exposes an external HTTP endpoint
2. Receives CloudEvents from external sources
3. Forwards them to the internal Broker

Here's how to create an Event Gateway function:

**1. Create the Event Gateway code (Go example):**

```go
package main

import (
    "bytes"
    "io"
    "log"
    "net/http"
    "os"
)

func main() {
    // Broker URL - adjust namespace and broker name as needed
    brokerURL := os.Getenv("BROKER_URL")
    if brokerURL == "" {
        brokerURL = "http://broker-ingress.knative-eventing.svc.cluster.local/default/default"
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Read the incoming request body
        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "Failed to read body", http.StatusBadRequest)
            return
        }
        defer r.Body.Close()

        // Create request to internal Broker
        req, err := http.NewRequest("POST", brokerURL, bytes.NewReader(body))
        if err != nil {
            http.Error(w, "Failed to create request", http.StatusInternalServerError)
            return
        }

        // Forward all CloudEvent headers (Ce-*)
        for name, values := range r.Header {
            for _, value := range values {
                req.Header.Add(name, value)
            }
        }

        // Send to Broker
        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            log.Printf("Failed to forward event: %v", err)
            http.Error(w, "Failed to forward event", http.StatusBadGateway)
            return
        }
        defer resp.Body.Close()

        // Return Broker's response
        w.WriteHeader(resp.StatusCode)
        io.Copy(w, resp.Body)
        log.Printf("Event forwarded to broker, status: %d", resp.StatusCode)
    })

    log.Printf("Event Gateway listening on port %s, forwarding to %s", port, brokerURL)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

**2. Deploy the Event Gateway with external visibility:**

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: event-gateway
  namespace: default
spec:
  gitRepo: https://github.com/myorg/event-gateway
  gitRevision: main
  build:
    image: registry.example.com/event-gateway:latest
  deploy:
    visibility: external  # Accessible from outside the cluster
    env:
      - name: BROKER_URL
        value: "http://broker-ingress.knative-eventing.svc.cluster.local/default/default"
```

**3. Send events from outside the cluster:**

**Linux:**
```bash
# Get gateway IP
GATEWAY_IP=$(kubectl get svc -n envoy-gateway-system \
  -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway \
  -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')

# Send event through the Event Gateway
curl -X POST \
  -H "Host: event-gateway.default.example.com" \
  -H "Ce-Id: external-event-123" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: com.example.order.created" \
  -H "Ce-Source: external-system" \
  -H "Content-Type: application/json" \
  -d '{"orderId": "EXT-001", "amount": 150.00}' \
  http://$GATEWAY_IP/
```

**MacOS (Docker Desktop / Colima):**
```bash
# Terminal 1: Start port-forward
kubectl port-forward -n envoy-gateway-system \
  $(kubectl get svc -n envoy-gateway-system \
    -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway \
    -o jsonpath='{.items[0].metadata.name}') 8080:80

# Terminal 2: Send event through the Event Gateway
curl -X POST \
  -H "Host: event-gateway.default.example.com" \
  -H "Ce-Id: external-event-123" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: com.example.order.created" \
  -H "Ce-Source: external-system" \
  -H "Content-Type: application/json" \
  -d '{"orderId": "EXT-001", "amount": 150.00}' \
  http://localhost:8080/
```

The Event Gateway will forward the CloudEvent to the internal Broker, which will then route it to the appropriate function based on the Trigger filters.

**Architecture:**
```
External Client → Event Gateway (external) → Broker (internal) → Trigger → Function
```

## Step 3: Understand Event Filters

Filters are based on CloudEvents attributes. You can filter by:

### Filter by Type

```yaml
eventing:
  broker: default
  filters:
    type: com.example.order.created
```

Only events with `type: com.example.order.created` will be routed to your function.

### Filter by Source

```yaml
eventing:
  broker: default
  filters:
    source: payment-service
```

Only events originating from `payment-service` will be processed.

### Multiple Filters (AND)

```yaml
eventing:
  broker: default
  filters:
    type: com.example.order.created
    source: payment-service
    subject: orders
```

All filters must match (AND operation).

### No Filters (All Events)

```yaml
eventing:
  broker: default
  filters: {}
```

Your function will receive all events from the Broker.

## Step 4: Monitor Deployment

Check the status of the function and Trigger:

```bash
# Check function status
kubectl get functions

# Check function details
kubectl describe function order-processor

# Check created Trigger
kubectl get triggers

# Check Trigger details
kubectl describe trigger order-processor-trigger
```

## Step 5: Send Test Events

To test your function, send an event to the Broker:

### Using curl with CloudEvents

```bash
# Get Broker URL
BROKER_URL=$(kubectl get broker default -o jsonpath='{.status.address.url}')

# Send an event
curl -v "$BROKER_URL" \
  -X POST \
  -H "Ce-Id: 12345" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: com.example.order.created" \
  -H "Ce-Source: payment-service" \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORD-001",
    "amount": 99.99,
    "customer": "john@example.com"
  }'
```

### Using a Test Pod

```bash
# Create a pod to send events
kubectl run curl-pod --image=curlimages/curl --rm -it --restart=Never -- sh

# Inside pod, send event
curl -v http://broker-ingress.knative-eventing.svc.cluster.local/default/default \
  -X POST \
  -H "Ce-Id: 12345" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: com.example.order.created" \
  -H "Ce-Source: payment-service" \
  -H "Content-Type: application/json" \
  -d '{"orderId":"ORD-001","amount":99.99}'
```

## Step 6: Verify Processing

Check function logs to confirm the event was processed:

```bash
# Check function pods
kubectl get pods -l serving.knative.dev/service=order-processor

# Check function logs
kubectl logs -l serving.knative.dev/service=order-processor -f
```

You should see logs indicating the event was received and processed.

## Common Use Cases

### 1. Order Processing

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: order-processor
spec:
  gitRepo: https://github.com/myorg/order-processor
  gitRevision: main
  build:
    image: registry.example.com/order-processor:latest
  deploy:
    env:
      - name: DATABASE_URL
        value: postgres://db.example.com/orders
  eventing:
    broker: default
    filters:
      type: com.example.order.created
```

### 2. Email Notifications

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: email-notifier
spec:
  gitRepo: https://github.com/myorg/email-notifier
  gitRevision: main
  build:
    image: registry.example.com/email-notifier:latest
  deploy:
    env:
      - name: SMTP_HOST
        value: smtp.example.com
      - name: SMTP_PORT
        value: "587"
  eventing:
    broker: default
    filters:
      type: com.example.notification.email
```

### 3. Log Processing

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: log-processor
spec:
  gitRepo: https://github.com/myorg/log-processor
  gitRevision: main
  build:
    image: registry.example.com/log-processor:latest
  deploy: {}
  eventing:
    broker: default
    filters:
      type: com.example.log.error
      source: application-backend
```

## Integration with Multiple Brokers

You can have multiple Brokers for different purposes:

```bash
# Create Broker for production
kubectl create -f - <<EOF
apiVersion: eventing.knative.dev/v1
kind: Broker
metadata:
  name: production
  namespace: default
EOF

# Create Broker for staging
kubectl create -f - <<EOF
apiVersion: eventing.knative.dev/v1
kind: Broker
metadata:
  name: staging
  namespace: default
EOF
```

And reference in Function:

```yaml
eventing:
  broker: production  # or staging
  filters:
    type: com.example.order.created
```

## Advanced Patterns

### Dead Letter Queue (DLQ)

Configure a DLQ for events that fail processing:

```yaml
apiVersion: eventing.knative.dev/v1
kind: Trigger
metadata:
  name: order-processor-trigger
spec:
  broker: default
  filter:
    attributes:
      type: com.example.order.created
  subscriber:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: order-processor
  delivery:
    deadLetterSink:
      ref:
        apiVersion: serving.knative.dev/v1
        kind: Service
        name: dlq-handler
    retry: 3
```

### Fan-out (Multiple Functions)

Multiple functions can process the same event:

```yaml
# Function 1: Save to database
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: order-saver
spec:
  # ...
  eventing:
    broker: default
    filters:
      type: com.example.order.created
---
# Function 2: Send email
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: order-emailer
spec:
  # ...
  eventing:
    broker: default
    filters:
      type: com.example.order.created
```

Both functions will receive the same event.

## Troubleshooting

### Events Not Reaching Function

1. **Check Broker**:
```bash
kubectl get broker default
kubectl describe broker default
```

2. **Check Trigger**:
```bash
kubectl get trigger
kubectl describe trigger order-processor-trigger
```

3. **Check Filters**:
Ensure event attributes match the filters.

4. **Test Connectivity**:
```bash
# Send test event
kubectl run curl-pod --image=curlimages/curl --rm -it --restart=Never -- \
  curl -v http://broker-ingress.knative-eventing.svc.cluster.local/default/default \
  -X POST \
  -H "Ce-Id: test-123" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: com.example.order.created" \
  -H "Ce-Source: test" \
  -H "Content-Type: application/json" \
  -d '{"test": true}'
```

### Function Not Processing Events Correctly

1. **Check Logs**:
```bash
kubectl logs -l serving.knative.dev/service=order-processor -f
```

2. **Check Event Format**:
Ensure your function is decoding the CloudEvent correctly.

3. **Check Response Format**:
If you see errors in the broker-filter logs like:
```
"error":"received a non-empty response not recognized as CloudEvent. The response MUST be either empty or a valid CloudEvent"
```

This means your function is returning a response body (like JSON) instead of an empty response. Event handlers **MUST** return either:
- Empty response (HTTP 200 with no body) - Recommended
- Valid CloudEvent response

Fix your function to return an empty response:
```go
// Go
w.WriteHeader(http.StatusOK)
```
```python
# Python
return Response(status=200)
```
```javascript
// Node.js
res.status(200).end();
```

4. **Test Locally**:
Send a test CloudEvent directly to the function:
```bash
FUNCTION_URL=$(kubectl get function order-processor -o jsonpath='{.status.url}')
curl -v "$FUNCTION_URL" \
  -X POST \
  -H "Ce-Id: test-123" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: com.example.order.created" \
  -H "Ce-Source: test" \
  -H "Content-Type: application/json" \
  -d '{"test": true}'
```

## Complete Examples

See complete examples:
- [test/chainsaw/eventing-trigger/](https://github.com/LucasGois1/zenith-operator/tree/main/test/chainsaw/eventing-trigger) - Eventing E2E Test
- [config/samples/](https://github.com/LucasGois1/zenith-operator/tree/main/config/samples) - Function CR Examples

## Next Steps

- [Creating Synchronous HTTP Functions](http-functions.md)
- [Function Communication](function-communication.md)
- [Function CRD Specification](../04-reference/function-crd.md)
- [Operator Reference](../04-reference/operator-reference.md)

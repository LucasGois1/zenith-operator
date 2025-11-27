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

### Go Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
)

// CloudEvent represents an event in CloudEvents format
type CloudEvent struct {
    SpecVersion     string                 `json:"specversion"`
    Type            string                 `json:"type"`
    Source          string                 `json:"source"`
    ID              string                 `json:"id"`
    Time            string                 `json:"time"`
    DataContentType string                 `json:"datacontenttype"`
    Data            map[string]interface{} `json:"data"`
}

type Response struct {
    Status  string `json:"status"`
    Message string `json:"message"`
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
    // Decode CloudEvent
    var event CloudEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Process event
    log.Printf("Received event: type=%s, source=%s, id=%s", 
        event.Type, event.Source, event.ID)
    log.Printf("Event data: %+v", event.Data)
    
    // Your processing logic here
    processEvent(event)
    
    // Return response
    response := Response{
        Status:  "processed",
        Message: fmt.Sprintf("Event %s processed successfully", event.ID),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func processEvent(event CloudEvent) {
    // Implement your processing logic
    // Example: save to database, send notification, etc.
    log.Printf("Processing event of type: %s", event.Type)
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
from flask import Flask, request, jsonify
import logging
import os

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)

@app.route('/', methods=['POST'])
def event_handler():
    # Receive CloudEvent
    event = request.get_json()
    
    # Log event
    logging.info(f"Received event: type={event.get('type')}, "
                f"source={event.get('source')}, id={event.get('id')}")
    logging.info(f"Event data: {event.get('data')}")
    
    # Process event
    process_event(event)
    
    # Return response
    return jsonify({
        'status': 'processed',
        'message': f"Event {event.get('id')} processed successfully"
    })

def process_event(event):
    # Implement your processing logic
    event_type = event.get('type')
    data = event.get('data', {})
    
    logging.info(f"Processing event of type: {event_type}")
    # Your logic here

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8080))
    app.run(host='0.0.0.0', port=8080)
```

### Node.js Example

```javascript
const express = require('express');
const app = express();

app.use(express.json());

app.post('/', (req, res) => {
    // Receive CloudEvent
    const event = req.body;
    
    // Log event
    console.log(`Received event: type=${event.type}, source=${event.source}, id=${event.id}`);
    console.log(`Event data:`, event.data);
    
    // Process event
    processEvent(event);
    
    // Return response
    res.json({
        status: 'processed',
        message: `Event ${event.id} processed successfully`
    });
});

function processEvent(event) {
    // Implement your processing logic
    console.log(`Processing event of type: ${event.type}`);
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

3. **Test Locally**:
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

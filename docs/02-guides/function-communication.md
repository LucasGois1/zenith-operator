# Function Communication

This guide shows how to implement HTTP communication between multiple functions using Zenith Operator.

## Overview

Function communication allows creating microservices architectures where functions communicate via HTTP to:
- Divide responsibilities between services
- Create complex workflows
- Implement patterns like saga, orchestration, choreography
- Build distributed systems

Zenith Operator facilitates this through:
1. Predictable Service URLs (Knative Service URLs)
2. Native Kubernetes Service Discovery
3. Optimized intra-cluster communication
4. Optional integration with Dapr for service mesh

## Communication Architecture

```
┌──────────────────┐      HTTP      ┌──────────────────┐      HTTP      ┌──────────────────┐
│   Transaction    │───────────────▶│     Balance      │───────────────▶│   Audit Logger   │
│   Processor      │                │     Manager      │                │                  │
└──────────────────┘                └──────────────────┘                └──────────────────┘
      Function 1                          Function 2                          Function 3
```

In this example:
1. **Transaction Processor** receives external request
2. Calls **Balance Manager** to update balance
3. **Balance Manager** calls **Audit Logger** to log operation
4. Response returns through the chain

## Service URL Patterns

Each function deployed by Zenith Operator receives a Knative Service URL:

### Internal URL (Cluster)

```
http://<function-name>.<namespace>.svc.cluster.local
```

Example:
```
http://balance-manager.default.svc.cluster.local
```

### External URL (Public)

```
http://<function-name>.<namespace>.<domain>
```

Example:
```
http://balance-manager.default.example.com
```

**Recommendation**: Use internal URLs for communication between functions in the same cluster.

## Step 1: Create Functions

Let's create a financial system with three communicating functions.

### Function 1: Audit Logger

Logs audit operations.

**Code (Go)**:
```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
    "time"
)

type AuditRequest struct {
    Action  string                 `json:"action"`
    Details map[string]interface{} `json:"details"`
}

type AuditResponse struct {
    Status    string    `json:"status"`
    Timestamp time.Time `json:"timestamp"`
}

func logHandler(w http.ResponseWriter, r *http.Request) {
    var req AuditRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Log audit
    log.Printf("[AUDIT] Action: %s, Details: %+v", req.Action, req.Details)
    
    response := AuditResponse{
        Status:    "logged",
        Timestamp: time.Now(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    http.HandleFunc("/log", logHandler)
    log.Printf("Audit Logger listening on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

**Function CR**:
```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: audit-logger
  namespace: default
spec:
  gitRepo: https://github.com/myorg/audit-logger
  gitRevision: main
  gitAuthSecretName: github-auth
  build:
    image: registry.example.com/audit-logger:latest
  deploy: {}
```

### Function 2: Balance Manager

Manages balances and calls Audit Logger.

**Code (Go)**:
```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
)

type UpdateRequest struct {
    Account string  `json:"account"`
    Amount  float64 `json:"amount"`
}

type UpdateResponse struct {
    Status     string  `json:"status"`
    NewBalance float64 `json:"new_balance"`
    Account    string  `json:"account"`
}

type AuditRequest struct {
    Action  string                 `json:"action"`
    Details map[string]interface{} `json:"details"`
}

var balances = make(map[string]float64)

func updateHandler(w http.ResponseWriter, r *http.Request) {
    var req UpdateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Update balance
    balances[req.Account] += req.Amount
    newBalance := balances[req.Account]
    
    log.Printf("Updated balance for %s: %.2f", req.Account, newBalance)
    
    // Call Audit Logger
    namespace := os.Getenv("POD_NAMESPACE")
    if namespace == "" {
        namespace = "default"
    }
    auditURL := fmt.Sprintf("http://audit-logger.%s.svc.cluster.local/log", namespace)
    
    auditReq := AuditRequest{
        Action: "balance_update",
        Details: map[string]interface{}{
            "account":     req.Account,
            "amount":      req.Amount,
            "new_balance": newBalance,
        },
    }
    
    auditJSON, _ := json.Marshal(auditReq)
    resp, err := http.Post(auditURL, "application/json", bytes.NewBuffer(auditJSON))
    if err != nil {
        log.Printf("Failed to call audit logger: %v", err)
    } else {
        resp.Body.Close()
        log.Printf("Audit logged successfully")
    }
    
    // Return response
    response := UpdateResponse{
        Status:     "updated",
        NewBalance: newBalance,
        Account:    req.Account,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    http.HandleFunc("/update", updateHandler)
    log.Printf("Balance Manager listening on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

**Function CR**:
```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: balance-manager
  namespace: default
spec:
  gitRepo: https://github.com/myorg/balance-manager
  gitRevision: main
  gitAuthSecretName: github-auth
  build:
    image: registry.example.com/balance-manager:latest
  deploy:
    env:
      - name: POD_NAMESPACE
        value: default
```

### Function 3: Transaction Processor

Processes transactions and calls Balance Manager.

**Code (Go)**:
```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
)

type TransactionRequest struct {
    Account string  `json:"account"`
    Amount  float64 `json:"amount"`
}

type TransactionResponse struct {
    Status        string    `json:"status"`
    TransactionID string    `json:"transaction_id"`
    Balance       float64   `json:"balance"`
    Timestamp     time.Time `json:"timestamp"`
}

type BalanceUpdateRequest struct {
    Account string  `json:"account"`
    Amount  float64 `json:"amount"`
}

type BalanceUpdateResponse struct {
    Status     string  `json:"status"`
    NewBalance float64 `json:"new_balance"`
    Account    string  `json:"account"`
}

func transactionHandler(w http.ResponseWriter, r *http.Request) {
    var req TransactionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    log.Printf("Processing transaction: account=%s, amount=%.2f", req.Account, req.Amount)
    
    // Call Balance Manager
    namespace := os.Getenv("POD_NAMESPACE")
    if namespace == "" {
        namespace = "default"
    }
    balanceURL := fmt.Sprintf("http://balance-manager.%s.svc.cluster.local/update", namespace)
    
    updateReq := BalanceUpdateRequest{
        Account: req.Account,
        Amount:  req.Amount,
    }
    
    updateJSON, _ := json.Marshal(updateReq)
    resp, err := http.Post(balanceURL, "application/json", bytes.NewBuffer(updateJSON))
    if err != nil {
        log.Printf("Failed to call balance manager: %v", err)
        http.Error(w, "Failed to update balance", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    
    var updateResp BalanceUpdateResponse
    if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
        log.Printf("Failed to decode balance response: %v", err)
        http.Error(w, "Failed to decode balance response", http.StatusInternalServerError)
        return
    }
    
    log.Printf("Balance updated: %+v", updateResp)
    
    // Generate transaction ID
    transactionID := fmt.Sprintf("TXN-%d", time.Now().Unix())
    
    // Return response
    response := TransactionResponse{
        Status:        "success",
        TransactionID: transactionID,
        Balance:       updateResp.NewBalance,
        Timestamp:     time.Now(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    http.HandleFunc("/transaction", transactionHandler)
    log.Printf("Transaction Processor listening on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

**Function CR**:
```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: transaction-processor
  namespace: default
spec:
  gitRepo: https://github.com/myorg/transaction-processor
  gitRevision: main
  gitAuthSecretName: github-auth
  build:
    image: registry.example.com/transaction-processor:latest
  deploy:
    env:
      - name: POD_NAMESPACE
        value: default
```

## Step 2: Deploy Functions

Deploy all functions:

```bash
kubectl apply -f audit-logger.yaml
kubectl apply -f balance-manager.yaml
kubectl apply -f transaction-processor.yaml
```

Wait until they are all ready:

```bash
kubectl get functions
kubectl wait --for=condition=Ready function/audit-logger --timeout=10m
kubectl wait --for=condition=Ready function/balance-manager --timeout=10m
kubectl wait --for=condition=Ready function/transaction-processor --timeout=10m
```

## Step 3: Test Communication

Send a request to the Transaction Processor:

```bash
# Get Transaction Processor URL
TRANSACTION_URL=$(kubectl get function transaction-processor -o jsonpath='{.status.url}')

# Send transaction
curl -X POST "$TRANSACTION_URL/transaction" \
  -H "Content-Type: application/json" \
  -d '{
    "account": "ACC-001",
    "amount": 100.50
  }'
```

Expected response:
```json
{
  "status": "success",
  "transaction_id": "TXN-1234567890",
  "balance": 100.50,
  "timestamp": "2025-01-15T10:30:00Z"
}
```

## Step 4: Check Logs

Check logs of each function to see the call chain:

```bash
# Transaction Processor Logs
kubectl logs -l serving.knative.dev/service=transaction-processor --tail=20

# Balance Manager Logs
kubectl logs -l serving.knative.dev/service=balance-manager --tail=20

# Audit Logger Logs
kubectl logs -l serving.knative.dev/service=audit-logger --tail=20
```

You will see:
1. Transaction Processor received request
2. Balance Manager updated balance
3. Audit Logger logged operation

## Communication Patterns

### 1. Synchronous Request-Response

Pattern used in the examples above. The function waits for response before continuing.

**Advantages**:
- Simple to implement
- Immediate response
- Easy error handling

**Disadvantages**:
- Temporal coupling
- Accumulated latency
- Cascading failure

### 2. Asynchronous Fire-and-Forget

Send request without waiting for response:

```go
go func() {
    http.Post(auditURL, "application/json", bytes.NewBuffer(auditJSON))
}()
```

**Advantages**:
- Low latency
- Decoupling
- Non-blocking

**Disadvantages**:
- No confirmation
- Hard error handling

### 3. Event-Driven via Broker

Use Knative Eventing for asynchronous communication:

```go
// Transaction Processor sends event
event := CloudEvent{
    Type:   "com.example.transaction.created",
    Source: "transaction-processor",
    Data:   transactionData,
}
sendToEventBroker(event)

// Balance Manager subscribes to event
// (configured via spec.eventing in Function CR)
```

**Advantages**:
- Total decoupling
- Scalability
- Resilience

**Disadvantages**:
- Higher complexity
- Eventual consistency

## Advanced Configurations

### Service Discovery with Environment Variables

Make URLs configurable:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: transaction-processor
spec:
  # ...
  deploy:
    env:
      - name: BALANCE_MANAGER_URL
        value: http://balance-manager.default.svc.cluster.local
      - name: AUDIT_LOGGER_URL
        value: http://audit-logger.default.svc.cluster.local
```

In code:
```go
balanceURL := os.Getenv("BALANCE_MANAGER_URL")
auditURL := os.Getenv("AUDIT_LOGGER_URL")
```

### Timeout and Retry

Implement timeout and retry for resilience:

```go
import "time"

client := &http.Client{
    Timeout: 5 * time.Second,
}

var resp *http.Response
var err error
for i := 0; i < 3; i++ {
    resp, err = client.Post(balanceURL, "application/json", body)
    if err == nil {
        break
    }
    log.Printf("Retry %d: %v", i+1, err)
    time.Sleep(time.Second * time.Duration(i+1))
}
```

### Circuit Breaker

Use libraries like `github.com/sony/gobreaker`:

```go
import "github.com/sony/gobreaker"

var cb *gobreaker.CircuitBreaker

func init() {
    cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        "balance-manager",
        MaxRequests: 3,
        Timeout:     10 * time.Second,
    })
}

func callBalanceManager() error {
    _, err := cb.Execute(func() (interface{}, error) {
        return http.Post(balanceURL, "application/json", body)
    })
    return err
}
```

### Integration with Dapr

Use Dapr for service discovery and resilience:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: transaction-processor
spec:
  # ...
  deploy:
    dapr:
      enabled: true
      appID: transaction-processor
      appPort: 8080
```

In code, use Dapr SDK:
```go
import "github.com/dapr/go-sdk/client"

daprClient, _ := client.NewClient()
resp, err := daprClient.InvokeMethod(ctx, "balance-manager", "update", "post")
```

## Troubleshooting

### Error: Connection Refused

**Problem**: Function cannot connect to another function.

**Solutions**:
1. Check if target function is running:
```bash
kubectl get ksvc
kubectl get pods -l serving.knative.dev/service=balance-manager
```

2. Check URL:
```bash
# URL must be: http://<function-name>.<namespace>.svc.cluster.local
kubectl get ksvc balance-manager -o jsonpath='{.status.url}'
```

3. Test connectivity:
```bash
kubectl run curl-pod --image=curlimages/curl --rm -it --restart=Never -- \
  curl -v http://balance-manager.default.svc.cluster.local
```

### Error: Timeout

**Problem**: Request takes too long or times out.

**Solutions**:
1. Check if function is in scale-to-zero:
```bash
kubectl get pods -l serving.knative.dev/service=balance-manager
```

2. Configure higher timeout in HTTP client:
```go
client := &http.Client{
    Timeout: 30 * time.Second,
}
```

3. Increase Knative Service timeout:
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: balance-manager
spec:
  template:
    spec:
      timeoutSeconds: 300
```

### Error: DNS Resolution Failed

**Problem**: Service name does not resolve.

**Solutions**:
1. Check namespace:
```bash
kubectl get ksvc -A
```

2. Use full FQDN:
```
http://balance-manager.default.svc.cluster.local
```

3. Check cluster DNS:
```bash
kubectl run dnsutils --image=gcr.io/kubernetes-e2e-test-images/dnsutils:1.3 --rm -it --restart=Never -- nslookup balance-manager.default.svc.cluster.local
```

## Complete Examples

See complete financial integration example:
- [test/chainsaw/financial-integration/](https://github.com/LucasGois1/zenith-operator/tree/main/test/chainsaw/financial-integration) - Complete E2E Test

## Next Steps

- [Creating Synchronous HTTP Functions](http-functions.md)
- [Creating Asynchronous Event Functions](event-functions.md)
- [Function CRD Specification](../04-reference/function-crd.md)
- [Operator Reference](../04-reference/operator-reference.md)

# Comunicação entre Funções

Este guia mostra como implementar comunicação HTTP entre múltiplas funções usando o Zenith Operator.

## Visão Geral

A comunicação entre funções permite criar arquiteturas de microserviços onde funções se comunicam via HTTP para:
- Dividir responsabilidades entre serviços
- Criar workflows complexos
- Implementar padrões como saga, orchestration, choreography
- Construir sistemas distribuídos

O Zenith Operator facilita isso através de:
1. URLs de serviço previsíveis (Knative Service URLs)
2. Service discovery nativo do Kubernetes
3. Comunicação intra-cluster otimizada
4. Integração opcional com Dapr para service mesh

## Arquitetura de Comunicação

```
┌──────────────────┐      HTTP      ┌──────────────────┐      HTTP      ┌──────────────────┐
│   Transaction    │───────────────▶│     Balance      │───────────────▶│   Audit Logger   │
│   Processor      │                │     Manager      │                │                  │
└──────────────────┘                └──────────────────┘                └──────────────────┘
      Function 1                          Function 2                          Function 3
```

Neste exemplo:
1. **Transaction Processor** recebe requisição externa
2. Chama **Balance Manager** para atualizar saldo
3. **Balance Manager** chama **Audit Logger** para registrar operação
4. Resposta retorna pela cadeia

## Padrões de URL de Serviço

Cada função deployada pelo Zenith Operator recebe uma URL do Knative Service:

### URL Interna (Cluster)

```
http://<function-name>.<namespace>.svc.cluster.local
```

Exemplo:
```
http://balance-manager.default.svc.cluster.local
```

### URL Externa (Pública)

```
http://<function-name>.<namespace>.<domain>
```

Exemplo:
```
http://balance-manager.default.example.com
```

**Recomendação**: Use URLs internas para comunicação entre funções no mesmo cluster.

## Passo 1: Criar as Funções

Vamos criar um sistema financeiro com três funções que se comunicam.

### Função 1: Audit Logger

Registra operações de auditoria.

**Código (Go)**:
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
    
    // Registrar auditoria
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

### Função 2: Balance Manager

Gerencia saldos e chama o Audit Logger.

**Código (Go)**:
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
    
    // Atualizar saldo
    balances[req.Account] += req.Amount
    newBalance := balances[req.Account]
    
    log.Printf("Updated balance for %s: %.2f", req.Account, newBalance)
    
    // Chamar Audit Logger
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
    
    // Retornar resposta
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

### Função 3: Transaction Processor

Processa transações e chama o Balance Manager.

**Código (Go)**:
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
    
    // Chamar Balance Manager
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
    
    // Gerar ID de transação
    transactionID := fmt.Sprintf("TXN-%d", time.Now().Unix())
    
    // Retornar resposta
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

## Passo 2: Deploy das Funções

Deploy todas as funções:

```bash
kubectl apply -f audit-logger.yaml
kubectl apply -f balance-manager.yaml
kubectl apply -f transaction-processor.yaml
```

Aguarde até que todas estejam prontas:

```bash
kubectl get functions
kubectl wait --for=condition=Ready function/audit-logger --timeout=10m
kubectl wait --for=condition=Ready function/balance-manager --timeout=10m
kubectl wait --for=condition=Ready function/transaction-processor --timeout=10m
```

## Passo 3: Testar a Comunicação

Envie uma requisição para o Transaction Processor:

```bash
# Obter URL do Transaction Processor
TRANSACTION_URL=$(kubectl get function transaction-processor -o jsonpath='{.status.url}')

# Enviar transação
curl -X POST "$TRANSACTION_URL/transaction" \
  -H "Content-Type: application/json" \
  -d '{
    "account": "ACC-001",
    "amount": 100.50
  }'
```

Resposta esperada:
```json
{
  "status": "success",
  "transaction_id": "TXN-1234567890",
  "balance": 100.50,
  "timestamp": "2025-01-15T10:30:00Z"
}
```

## Passo 4: Verificar os Logs

Verifique os logs de cada função para ver a cadeia de chamadas:

```bash
# Logs do Transaction Processor
kubectl logs -l serving.knative.dev/service=transaction-processor --tail=20

# Logs do Balance Manager
kubectl logs -l serving.knative.dev/service=balance-manager --tail=20

# Logs do Audit Logger
kubectl logs -l serving.knative.dev/service=audit-logger --tail=20
```

Você verá:
1. Transaction Processor recebeu a requisição
2. Balance Manager atualizou o saldo
3. Audit Logger registrou a operação

## Padrões de Comunicação

### 1. Request-Response Síncrono

Padrão usado nos exemplos acima. A função aguarda resposta antes de continuar.

**Vantagens**:
- Simples de implementar
- Resposta imediata
- Fácil tratamento de erros

**Desvantagens**:
- Acoplamento temporal
- Latência acumulada
- Falha em cascata

### 2. Fire-and-Forget Assíncrono

Enviar requisição sem aguardar resposta:

```go
go func() {
    http.Post(auditURL, "application/json", bytes.NewBuffer(auditJSON))
}()
```

**Vantagens**:
- Baixa latência
- Desacoplamento
- Não bloqueia

**Desvantagens**:
- Sem confirmação
- Difícil tratamento de erros

### 3. Event-Driven via Broker

Usar Knative Eventing para comunicação assíncrona:

```go
// Transaction Processor envia evento
event := CloudEvent{
    Type:   "com.example.transaction.created",
    Source: "transaction-processor",
    Data:   transactionData,
}
sendToEventBroker(event)

// Balance Manager subscreve ao evento
// (configurado via spec.eventing no Function CR)
```

**Vantagens**:
- Desacoplamento total
- Escalabilidade
- Resiliência

**Desvantagens**:
- Complexidade maior
- Eventual consistency

## Configurações Avançadas

### Service Discovery com Variáveis de Ambiente

Torne as URLs configuráveis:

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

No código:
```go
balanceURL := os.Getenv("BALANCE_MANAGER_URL")
auditURL := os.Getenv("AUDIT_LOGGER_URL")
```

### Timeout e Retry

Implemente timeout e retry para resiliência:

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

Use bibliotecas como `github.com/sony/gobreaker`:

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

### Integração com Dapr

Use Dapr para service discovery e resiliência:

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

No código, use Dapr SDK:
```go
import "github.com/dapr/go-sdk/client"

daprClient, _ := client.NewClient()
resp, err := daprClient.InvokeMethod(ctx, "balance-manager", "update", "post")
```

## Troubleshooting

### Erro: Connection Refused

**Problema**: Função não consegue conectar a outra função.

**Soluções**:
1. Verifique se a função de destino está rodando:
```bash
kubectl get ksvc
kubectl get pods -l serving.knative.dev/service=balance-manager
```

2. Verifique a URL:
```bash
# URL deve ser: http://<function-name>.<namespace>.svc.cluster.local
kubectl get ksvc balance-manager -o jsonpath='{.status.url}'
```

3. Teste conectividade:
```bash
kubectl run curl-pod --image=curlimages/curl --rm -it --restart=Never -- \
  curl -v http://balance-manager.default.svc.cluster.local
```

### Erro: Timeout

**Problema**: Requisição demora muito ou timeout.

**Soluções**:
1. Verifique se a função está em scale-to-zero:
```bash
kubectl get pods -l serving.knative.dev/service=balance-manager
```

2. Configure timeout maior no cliente HTTP:
```go
client := &http.Client{
    Timeout: 30 * time.Second,
}
```

3. Aumente o timeout do Knative Service:
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

### Erro: DNS Resolution Failed

**Problema**: Nome do serviço não resolve.

**Soluções**:
1. Verifique o namespace:
```bash
kubectl get ksvc -A
```

2. Use FQDN completo:
```
http://balance-manager.default.svc.cluster.local
```

3. Verifique DNS do cluster:
```bash
kubectl run dnsutils --image=gcr.io/kubernetes-e2e-test-images/dnsutils:1.3 --rm -it --restart=Never -- nslookup balance-manager.default.svc.cluster.local
```

## Exemplos Completos

Veja o exemplo completo de integração financeira:
- [test/chainsaw/financial-integration/](https://github.com/LucasGois1/zenith-operator/tree/main/test/chainsaw/financial-integration) - Teste E2E completo

## Próximos Passos

- [Criando Funções HTTP Síncronas](funcoes-http.md)
- [Criando Funções Assíncronas com Eventos](funcoes-eventos.md)
- [Especificação do CRD Function](../04-referencia/function-crd.md)
- [Referência do Operator](../04-referencia/operator-reference.md)

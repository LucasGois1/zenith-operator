# Criando Funções Assíncronas com Eventos

Este guia mostra como criar funções que processam eventos de forma assíncrona usando Knative Eventing e o Zenith Operator.

## Visão Geral

Funções event-driven (orientadas a eventos) são ideais para:
- Processamento assíncrono de dados
- Workflows event-driven
- Notificações e alertas
- Integração entre sistemas
- Processamento de mensagens de filas

O Zenith Operator automaticamente:
1. Clona seu repositório Git e constrói a imagem
2. Faz deploy como um Knative Service
3. Cria um Knative Trigger para subscrever eventos
4. Roteia eventos do Broker para sua função

## Pré-requisitos

- Cluster Kubernetes com Zenith Operator instalado
- Knative Eventing instalado
- Knative Broker criado no namespace
- Repositório Git com código da função
- Secret de autenticação Git (se repositório privado)

## Arquitetura Event-Driven

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Producer  │─────▶│   Broker    │─────▶│   Trigger   │─────▶│  Function   │
│  (Sistema)  │      │  (default)  │      │  (filtros)  │      │   (Sua)     │
└─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘
```

1. **Producer**: Sistema que envia eventos (CloudEvents) para o Broker
2. **Broker**: Recebe e distribui eventos
3. **Trigger**: Filtra eventos baseado em atributos e roteia para a função
4. **Function**: Sua função que processa os eventos

## Estrutura do Código da Função

Sua função deve receber requisições HTTP POST com eventos no formato CloudEvents:

### Exemplo em Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
)

// CloudEvent representa um evento no formato CloudEvents
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
    // Decodificar o CloudEvent
    var event CloudEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Processar o evento
    log.Printf("Received event: type=%s, source=%s, id=%s", 
        event.Type, event.Source, event.ID)
    log.Printf("Event data: %+v", event.Data)
    
    // Sua lógica de processamento aqui
    processEvent(event)
    
    // Retornar resposta
    response := Response{
        Status:  "processed",
        Message: fmt.Sprintf("Event %s processed successfully", event.ID),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func processEvent(event CloudEvent) {
    // Implementar sua lógica de processamento
    // Exemplo: salvar em banco de dados, enviar notificação, etc.
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

### Exemplo em Python

```python
from flask import Flask, request, jsonify
import logging
import os

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)

@app.route('/', methods=['POST'])
def event_handler():
    # Receber o CloudEvent
    event = request.get_json()
    
    # Log do evento
    logging.info(f"Received event: type={event.get('type')}, "
                f"source={event.get('source')}, id={event.get('id')}")
    logging.info(f"Event data: {event.get('data')}")
    
    # Processar o evento
    process_event(event)
    
    # Retornar resposta
    return jsonify({
        'status': 'processed',
        'message': f"Event {event.get('id')} processed successfully"
    })

def process_event(event):
    # Implementar sua lógica de processamento
    event_type = event.get('type')
    data = event.get('data', {})
    
    logging.info(f"Processing event of type: {event_type}")
    # Sua lógica aqui

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8080))
    app.run(host='0.0.0.0', port=port)
```

### Exemplo em Node.js

```javascript
const express = require('express');
const app = express();

app.use(express.json());

app.post('/', (req, res) => {
    // Receber o CloudEvent
    const event = req.body;
    
    // Log do evento
    console.log(`Received event: type=${event.type}, source=${event.source}, id=${event.id}`);
    console.log(`Event data:`, event.data);
    
    // Processar o evento
    processEvent(event);
    
    // Retornar resposta
    res.json({
        status: 'processed',
        message: `Event ${event.id} processed successfully`
    });
});

function processEvent(event) {
    // Implementar sua lógica de processamento
    console.log(`Processing event of type: ${event.type}`);
    // Sua lógica aqui
}

const port = process.env.PORT || 8080;
app.listen(port, () => {
    console.log(`Event processor listening on port ${port}`);
});
```

## Passo 1: Criar o Knative Broker

Primeiro, crie um Broker no namespace onde sua função será deployada:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: eventing.knative.dev/v1
kind: Broker
metadata:
  name: default
  namespace: default
EOF
```

Verifique se o Broker está pronto:

```bash
kubectl get broker default
```

## Passo 2: Criar o Custom Resource Function com Eventing

Crie um arquivo YAML com a definição da função incluindo configuração de eventing:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: order-processor
  namespace: default
spec:
  # Repositório Git com o código
  gitRepo: https://github.com/myorg/order-processor
  gitRevision: main
  
  # Secret de autenticação (opcional)
  gitAuthSecretName: github-auth
  
  # Configuração de build
  build:
    image: registry.example.com/order-processor:latest
  
  # Configuração de deploy
  deploy: {}
  
  # Configuração de eventing
  eventing:
    # Nome do Broker para subscrever
    broker: default
    
    # Filtros de eventos (CloudEvents attributes)
    filters:
      type: com.example.order.created
      source: payment-service
```

Aplique o recurso:

```bash
kubectl apply -f order-processor.yaml
```

## Passo 3: Entender os Filtros de Eventos

Os filtros são baseados em atributos do CloudEvents. Você pode filtrar por:

### Filtro por Type

```yaml
eventing:
  broker: default
  filters:
    type: com.example.order.created
```

Apenas eventos com `type: com.example.order.created` serão roteados para sua função.

### Filtro por Source

```yaml
eventing:
  broker: default
  filters:
    source: payment-service
```

Apenas eventos originados de `payment-service` serão processados.

### Múltiplos Filtros (AND)

```yaml
eventing:
  broker: default
  filters:
    type: com.example.order.created
    source: payment-service
    subject: orders
```

Todos os filtros devem corresponder (operação AND).

### Sem Filtros (Todos os Eventos)

```yaml
eventing:
  broker: default
  filters: {}
```

Sua função receberá todos os eventos do Broker.

## Passo 4: Monitorar o Deploy

Verifique o status da função e do Trigger:

```bash
# Ver status da função
kubectl get functions

# Ver detalhes da função
kubectl describe function order-processor

# Ver Trigger criado
kubectl get triggers

# Ver detalhes do Trigger
kubectl describe trigger order-processor-trigger
```

## Passo 5: Enviar Eventos de Teste

Para testar sua função, envie um evento para o Broker:

### Usando curl com CloudEvents

```bash
# Obter a URL do Broker
BROKER_URL=$(kubectl get broker default -o jsonpath='{.status.address.url}')

# Enviar um evento
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

### Usando um Pod de Teste

```bash
# Criar um pod para enviar eventos
kubectl run curl-pod --image=curlimages/curl --rm -it --restart=Never -- sh

# Dentro do pod, enviar evento
curl -v http://broker-ingress.knative-eventing.svc.cluster.local/default/default \
  -X POST \
  -H "Ce-Id: 12345" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: com.example.order.created" \
  -H "Ce-Source: payment-service" \
  -H "Content-Type: application/json" \
  -d '{"orderId":"ORD-001","amount":99.99}'
```

## Passo 6: Verificar Processamento

Verifique os logs da função para confirmar que o evento foi processado:

```bash
# Ver pods da função
kubectl get pods -l serving.knative.dev/service=order-processor

# Ver logs da função
kubectl logs -l serving.knative.dev/service=order-processor -f
```

Você deve ver logs indicando que o evento foi recebido e processado.

## Casos de Uso Comuns

### 1. Processamento de Pedidos

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

### 2. Notificações por Email

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

### 3. Processamento de Logs

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

## Integração com Múltiplos Brokers

Você pode ter múltiplos Brokers para diferentes propósitos:

```bash
# Criar Broker para produção
kubectl create -f - <<EOF
apiVersion: eventing.knative.dev/v1
kind: Broker
metadata:
  name: production
  namespace: default
EOF

# Criar Broker para staging
kubectl create -f - <<EOF
apiVersion: eventing.knative.dev/v1
kind: Broker
metadata:
  name: staging
  namespace: default
EOF
```

E referenciar no Function:

```yaml
eventing:
  broker: production  # ou staging
  filters:
    type: com.example.order.created
```

## Padrões Avançados

### Dead Letter Queue (DLQ)

Configure um DLQ para eventos que falharem no processamento:

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

### Fan-out (Múltiplas Funções)

Múltiplas funções podem processar o mesmo evento:

```yaml
# Função 1: Salvar no banco
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
# Função 2: Enviar email
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

Ambas as funções receberão o mesmo evento.

## Troubleshooting

### Eventos Não Chegam na Função

1. **Verificar Broker**:
```bash
kubectl get broker default
kubectl describe broker default
```

2. **Verificar Trigger**:
```bash
kubectl get trigger
kubectl describe trigger order-processor-trigger
```

3. **Verificar Filtros**:
Certifique-se de que os atributos do evento correspondem aos filtros.

4. **Testar Conectividade**:
```bash
# Enviar evento de teste
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

### Função Não Processa Eventos Corretamente

1. **Ver Logs**:
```bash
kubectl logs -l serving.knative.dev/service=order-processor -f
```

2. **Verificar Formato do Evento**:
Certifique-se de que sua função está decodificando o CloudEvent corretamente.

3. **Testar Localmente**:
Envie um CloudEvent de teste diretamente para a função:
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

## Exemplos Completos

Veja exemplos completos:
- [test/chainsaw/eventing-trigger/](../test/chainsaw/eventing-trigger/) - Teste E2E de eventing
- [config/samples/](../config/samples/) - Exemplos de Function CRs

## Próximos Passos

- [Criando Funções HTTP Síncronas](CREATING_HTTP_FUNCTIONS.md)
- [Comunicação entre Funções](INTER_FUNCTION_COMMUNICATION.md)
- [Referência Completa do Operator](OPERATOR_REFERENCE.md)

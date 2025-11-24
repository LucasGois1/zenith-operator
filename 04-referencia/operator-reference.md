# Referência do Operator - Comportamento e Integrações

Esta documentação descreve o comportamento do Zenith Operator e suas integrações com Tekton, Knative e Dapr.

## Visão Geral

O Zenith Operator é um operador Kubernetes que gerencia o ciclo de vida completo de funções serverless através do Custom Resource `Function`.

### Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                        Zenith Operator                          │
│                                                                 │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐      │
│  │   Function   │──▶│ PipelineRun  │──▶│   Service    │      │
│  │  Controller  │   │   (Tekton)   │   │  (Knative)   │      │
│  └──────────────┘   └──────────────┘   └──────────────┘      │
│         │                                       │              │
│         │                                       │              │
│         └───────────────────┬───────────────────┘              │
│                             │                                  │
│                      ┌──────────────┐                         │
│                      │   Trigger    │                         │
│                      │  (Eventing)  │                         │
│                      └──────────────┘                         │
└─────────────────────────────────────────────────────────────────┘
```

### Fluxo de Reconciliação

1. **Usuário cria Function CR** → Operator detecta novo recurso
2. **Build Phase** → Operator cria PipelineRun do Tekton
3. **PipelineRun executa** → Clona Git, constrói imagem, push para registry
4. **Build completa** → Operator extrai image digest
5. **Deploy Phase** → Operator cria Knative Service
6. **Eventing Phase** (opcional) → Operator cria Knative Trigger
7. **Status atualizado** → Function.status reflete estado atual

## Comportamento do Operator

### Reconciliation Loop

O operator reconcilia Functions continuamente:

1. **Watch**: Monitora mudanças em Function CRs
2. **Reconcile**: Processa cada Function
3. **Update Status**: Atualiza status baseado no estado atual
4. **Requeue**: Agenda próxima reconciliação se necessário

### Triggers de Reconciliação

O operator reconcilia quando:
- Function CR é criada
- Function CR é atualizada (spec mudou)
- PipelineRun completa
- Knative Service muda
- Trigger muda
- Reconciliação periódica (a cada 10 minutos)

### Idempotência

O operator é idempotente:
- Múltiplas reconciliações produzem o mesmo resultado
- Recursos existentes não são recriados
- Updates são aplicados apenas quando necessário

### Garbage Collection

O operator usa OwnerReferences para garbage collection:
- PipelineRuns são owned pela Function
- Knative Services são owned pela Function
- Triggers são owned pela Function
- ServiceAccounts são owned pela Function

Quando uma Function é deletada, todos os recursos owned são automaticamente deletados.

## Integração com Tekton

### PipelineRun Creation

O operator cria um PipelineRun para cada Function:

**Nome**: `<function-name>-<timestamp>`

**Tasks**:
1. **git-clone**: Clona o repositório Git
2. **buildpacks-phases**: Constrói a imagem usando Cloud Native Buildpacks

**Parameters**:
- `git-url`: URL do repositório Git
- `git-revision`: Revisão Git
- `image`: Nome da imagem de destino

**Workspaces**:
- `source`: Workspace para código-fonte
- `cache`: Workspace para cache de build

### ServiceAccount Management

O operator cria um ServiceAccount dedicado para cada Function:

**Nome**: `<function-name>-sa`

**Purpose**:
- Autenticação Git (via secrets)
- Autenticação Registry (via imagePullSecrets)

**Secrets Binding**:
- Git auth secret → `serviceAccount.secrets`
- Registry secret → `serviceAccount.imagePullSecrets`

### Image Digest Extraction

Após build bem-sucedido, o operator:
1. Aguarda PipelineRun completar
2. Extrai image digest do PipelineRun status
3. Atualiza Function.status.imageDigest

## Integração com Knative

### Service Creation

O operator cria um Knative Service para cada Function:

**Nome**: `<function-name>`

**Spec**:
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: my-function
  ownerReferences:
    - apiVersion: functions.zenith.com/v1alpha1
      kind: Function
      name: my-function
      uid: <function-uid>
      controller: true
      blockOwnerDeletion: true
spec:
  template:
    metadata:
      annotations:
        # Dapr annotations (if enabled)
        dapr.io/enabled: "true"
        dapr.io/app-id: "my-function"
        dapr.io/app-port: "8080"
    spec:
      containers:
        - image: registry.example.com/my-function@sha256:abc123...
          env:
            - name: DATABASE_URL
              value: postgres://db.example.com/mydb
      imagePullSecrets:
        - name: registry-credentials
```

### Auto-scaling

Knative Services auto-scale baseado em tráfego:
- **Scale-to-zero**: Pods são terminados quando não há tráfego
- **Scale-from-zero**: Pods são criados quando tráfego chega
- **Horizontal scaling**: Múltiplos pods para alto tráfego

### URL Exposure

Knative expõe URLs:
- **Internal**: `http://<service-name>.<namespace>.svc.cluster.local`
- **External**: `http://<service-name>.<namespace>.<domain>`

### Trigger Creation

Se `spec.eventing` está configurado, o operator cria um Trigger:

**Nome**: `<function-name>-trigger`

**Spec**:
```yaml
apiVersion: eventing.knative.dev/v1
kind: Trigger
metadata:
  name: my-function-trigger
  ownerReferences:
    - apiVersion: functions.zenith.com/v1alpha1
      kind: Function
      name: my-function
spec:
  broker: default
  filter:
    attributes:
      type: com.example.order.created
      source: payment-service
  subscriber:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: my-function
```

## Integração com Dapr

### Sidecar Injection

Quando `spec.deploy.dapr.enabled=true`, o operator adiciona annotations ao Knative Service:

```yaml
annotations:
  dapr.io/enabled: "true"
  dapr.io/app-id: "<appID>"
  dapr.io/app-port: "<appPort>"
```

### Dapr Features

Com Dapr habilitado, funções podem usar:

#### Service Invocation

```go
import "github.com/dapr/go-sdk/client"

daprClient, _ := client.NewClient()
resp, _ := daprClient.InvokeMethod(ctx, "other-function", "endpoint", "post")
```

#### Pub/Sub

```go
// Publish
daprClient.PublishEvent(ctx, "pubsub", "topic", data)

// Subscribe (via annotation)
// dapr.io/subscribe: '[{"pubsubname":"pubsub","topic":"topic","route":"/events"}]'
```

#### State Management

```go
// Save state
daprClient.SaveState(ctx, "statestore", "key", []byte("value"), nil)

// Get state
item, _ := daprClient.GetState(ctx, "statestore", "key", nil)
```

#### Secrets

```go
// Get secret
secret, _ := daprClient.GetSecret(ctx, "secretstore", "key", nil)
```

## Autenticação e Secrets

### Git Authentication

#### HTTPS Authentication

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-auth
  annotations:
    tekton.dev/git-0: https://github.com
type: kubernetes.io/basic-auth
stringData:
  username: myusername
  password: ghp_mytoken  # GitHub Personal Access Token
```

#### SSH Authentication

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-ssh
  annotations:
    tekton.dev/git-0: github.com
type: kubernetes.io/ssh-auth
stringData:
  ssh-privatekey: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    ...
    -----END OPENSSH PRIVATE KEY-----
```

### Registry Authentication

```bash
# Create secret
kubectl create secret docker-registry registry-credentials \
  --docker-server=registry.example.com \
  --docker-username=myuser \
  --docker-password=mypassword \
  --docker-email=myemail@example.com
```

## Variáveis de Ambiente

### Variáveis Injetadas pelo Operator

O operator injeta automaticamente:

- `PORT`: Porta em que a aplicação deve escutar (padrão: `8080`)
- `K_SERVICE`: Nome do Knative Service
- `K_CONFIGURATION`: Nome da Configuration
- `K_REVISION`: Nome da Revision

### Variáveis Customizadas

Adicione via `spec.deploy.env`:

```yaml
spec:
  deploy:
    env:
      - name: DATABASE_URL
        value: postgres://db.example.com/mydb
      - name: REDIS_URL
        value: redis://redis.default.svc.cluster.local:6379
```

### Variáveis de Secrets

Use `valueFrom` para referenciar Secrets:

```yaml
spec:
  deploy:
    env:
      - name: API_KEY
        valueFrom:
          secretKeyRef:
            name: api-credentials
            key: api-key
```

## Próximos Passos

- [Especificação do CRD](function-crd.md) - Campos e configurações do Function CRD
- [Troubleshooting](troubleshooting.md) - Solução de problemas comuns
- [Guia de Autenticação Git](../02-guias/autenticacao-git.md)
- [Configuração de Registry](../05-operacoes/configuracao-registry.md)

# Referência Completa do Zenith Operator

Esta é a documentação completa de todas as features, parâmetros, campos do CRD e funcionalidades do Zenith Operator.

## Índice

- [Visão Geral](#visão-geral)
- [Custom Resource Definition (CRD)](#custom-resource-definition-crd)
- [Spec Fields](#spec-fields)
- [Status Fields](#status-fields)
- [Condições de Status](#condições-de-status)
- [Comportamento do Operator](#comportamento-do-operator)
- [Integração com Tekton](#integração-com-tekton)
- [Integração com Knative](#integração-com-knative)
- [Integração com Dapr](#integração-com-dapr)
- [Autenticação e Secrets](#autenticação-e-secrets)
- [Variáveis de Ambiente](#variáveis-de-ambiente)
- [Troubleshooting](#troubleshooting)

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

## Custom Resource Definition (CRD)

### API Group e Version

- **API Group**: `functions.zenith.com`
- **Version**: `v1alpha1`
- **Kind**: `Function`
- **Plural**: `functions`
- **Singular**: `function`
- **Short Names**: `fn`, `func`

### Exemplo Completo

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: example-function
  namespace: default
  labels:
    app: my-app
    environment: production
  annotations:
    description: "Example function with all features"
spec:
  # Git Configuration
  gitRepo: https://github.com/myorg/my-function
  gitRevision: main
  gitAuthSecretName: github-auth
  
  # Build Configuration
  build:
    image: registry.example.com/my-function:latest
    registrySecretName: registry-credentials
  
  # Deploy Configuration
  deploy:
    dapr:
      enabled: true
      appID: example-function
      appPort: 8080
    env:
      - name: DATABASE_URL
        value: postgres://db.example.com/mydb
      - name: LOG_LEVEL
        value: info
  
  # Eventing Configuration (optional)
  eventing:
    broker: default
    filters:
      type: com.example.event.created
      source: my-service
status:
  # Populated by operator
  conditions:
    - type: Ready
      status: "True"
      reason: FunctionReady
      message: Function is ready
  imageDigest: registry.example.com/my-function@sha256:abc123...
  url: http://example-function.default.svc.cluster.local
  observedGeneration: 1
```

## Spec Fields

### gitRepo (Required)

**Type**: `string`

**Description**: URL do repositório Git contendo o código-fonte da função.

**Supported Protocols**:
- HTTPS: `https://github.com/myorg/my-function`
- SSH: `git@github.com:myorg/my-function.git`

**Examples**:
```yaml
# GitHub HTTPS
gitRepo: https://github.com/myorg/my-function

# GitLab HTTPS
gitRepo: https://gitlab.com/myorg/my-function

# GitHub SSH
gitRepo: git@github.com:myorg/my-function.git

# Self-hosted
gitRepo: https://git.example.com/myorg/my-function
```

### gitRevision (Optional)

**Type**: `string`

**Default**: `main`

**Description**: Revisão Git a ser usada (branch, tag, ou commit hash).

**Examples**:
```yaml
# Branch
gitRevision: main
gitRevision: develop
gitRevision: feature/new-feature

# Tag
gitRevision: v1.0.0
gitRevision: release-2024-01

# Commit hash
gitRevision: abc123def456
gitRevision: 1234567890abcdef1234567890abcdef12345678
```

### gitAuthSecretName (Optional)

**Type**: `string`

**Description**: Nome do Secret usado para autenticar com o repositório Git privado.

**Secret Type**: `kubernetes.io/basic-auth` ou `kubernetes.io/ssh-auth`

**Required Annotation**: `tekton.dev/git-0: <git-server>`

**Examples**:
```yaml
# Para repositórios privados
gitAuthSecretName: github-auth
gitAuthSecretName: gitlab-credentials
```

**Secret Example**:
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
  password: ghp_mytoken
```

### build (Required)

**Type**: `BuildSpec`

**Description**: Configuração do pipeline de build.

#### build.image (Required)

**Type**: `string`

**Description**: Nome completo da imagem de destino (sem tag ou digest).

**Format**: `<registry>/<repository>/<image>`

**Examples**:
```yaml
# Docker Hub
image: docker.io/myorg/my-function

# GitHub Container Registry
image: ghcr.io/myorg/my-function

# Google Container Registry
image: gcr.io/myproject/my-function

# Azure Container Registry
image: myregistry.azurecr.io/my-function

# Local registry
image: registry.registry.svc.cluster.local:5000/my-function
```

**Note**: O operator adiciona automaticamente o digest após o build: `image@sha256:...`

#### build.registrySecretName (Optional)

**Type**: `string`

**Description**: Nome do Secret usado para autenticar com o container registry.

**Secret Type**: `kubernetes.io/dockerconfigjson`

**Examples**:
```yaml
registrySecretName: registry-credentials
registrySecretName: dockerhub-secret
```

**Secret Example**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: registry-credentials
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64-encoded-docker-config>
```

### deploy (Required)

**Type**: `DeploySpec`

**Description**: Configuração de deployment da função.

#### deploy.dapr (Optional)

**Type**: `DaprConfig`

**Description**: Configuração do sidecar Dapr.

##### deploy.dapr.enabled (Required if dapr specified)

**Type**: `boolean`

**Description**: Se `true`, injeta o sidecar Dapr no pod.

**Default**: `false`

**Examples**:
```yaml
dapr:
  enabled: true
```

##### deploy.dapr.appID (Required if dapr.enabled=true)

**Type**: `string`

**Description**: ID único da aplicação Dapr.

**Constraints**:
- Deve ser único no namespace
- Lowercase alphanumeric e hífens
- Máximo 63 caracteres

**Examples**:
```yaml
dapr:
  enabled: true
  appID: my-function
  appID: order-processor
  appID: payment-service
```

##### deploy.dapr.appPort (Required if dapr.enabled=true)

**Type**: `integer`

**Description**: Porta em que a aplicação escuta.

**Default**: Nenhum (deve ser especificado)

**Common Values**: `8080`, `3000`, `8000`

**Examples**:
```yaml
dapr:
  enabled: true
  appID: my-function
  appPort: 8080
```

#### deploy.env (Optional)

**Type**: `[]EnvVar`

**Description**: Lista de variáveis de ambiente para injetar no container.

**EnvVar Fields**:
- `name` (string, required): Nome da variável
- `value` (string, required): Valor da variável

**Examples**:
```yaml
env:
  - name: DATABASE_URL
    value: postgres://db.example.com/mydb
  - name: API_KEY
    value: secret-key-123
  - name: LOG_LEVEL
    value: debug
  - name: FEATURE_FLAG_X
    value: "true"
```

**Note**: Para secrets sensíveis, considere usar Kubernetes Secrets e referenciá-los via `valueFrom`.

### eventing (Optional)

**Type**: `EventingSpec`

**Description**: Configuração de subscrição a eventos via Knative Eventing.

**Note**: Se especificado, o operator cria um Knative Trigger.

#### eventing.broker (Optional)

**Type**: `string`

**Default**: `default`

**Description**: Nome do Knative Broker para subscrever.

**Examples**:
```yaml
eventing:
  broker: default
  broker: production
  broker: staging
```

**Note**: O Broker deve existir no mesmo namespace.

#### eventing.filters (Optional)

**Type**: `map[string]string`

**Description**: Mapa de atributos CloudEvents para filtrar eventos.

**Common Attributes**:
- `type`: Tipo do evento
- `source`: Origem do evento
- `subject`: Assunto do evento
- Custom attributes

**Examples**:
```yaml
# Filtrar por type
eventing:
  broker: default
  filters:
    type: com.example.order.created

# Filtrar por type e source
eventing:
  broker: default
  filters:
    type: com.example.order.created
    source: payment-service

# Múltiplos filtros (AND)
eventing:
  broker: default
  filters:
    type: com.example.order.created
    source: payment-service
    subject: orders

# Sem filtros (todos os eventos)
eventing:
  broker: default
  filters: {}
```

**Note**: Todos os filtros devem corresponder (operação AND).

## Status Fields

O operator atualiza automaticamente o campo `status` da Function.

### conditions

**Type**: `[]metav1.Condition`

**Description**: Lista de condições que descrevem o estado da função.

**Condition Types**:
- `Ready`: Indica se a função está pronta para receber requisições
- `BuildSucceeded`: Indica se o build foi bem-sucedido
- `DeploySucceeded`: Indica se o deploy foi bem-sucedido

**Condition Fields**:
- `type` (string): Tipo da condição
- `status` (string): `True`, `False`, ou `Unknown`
- `reason` (string): Razão legível por máquina
- `message` (string): Mensagem legível por humano
- `lastTransitionTime` (timestamp): Última vez que a condição mudou

**Examples**:
```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: FunctionReady
      message: Function is ready and serving traffic
      lastTransitionTime: "2025-01-15T10:30:00Z"
    
    - type: BuildSucceeded
      status: "True"
      reason: BuildCompleted
      message: Image built successfully
      lastTransitionTime: "2025-01-15T10:25:00Z"
```

### imageDigest

**Type**: `string`

**Description**: Referência imutável da imagem construída (com digest SHA256).

**Format**: `<registry>/<repository>/<image>@sha256:<hash>`

**Examples**:
```yaml
imageDigest: registry.example.com/my-function@sha256:abc123def456...
imageDigest: docker.io/myorg/my-function@sha256:1234567890abcdef...
```

**Note**: Populado após build bem-sucedido.

### url

**Type**: `string`

**Description**: URL publicamente acessível da função (do Knative Service).

**Format**: `http://<function-name>.<namespace>.<domain>`

**Examples**:
```yaml
# URL interna (cluster)
url: http://my-function.default.svc.cluster.local

# URL externa (pública)
url: http://my-function.default.example.com
```

**Note**: Populado após deploy bem-sucedido.

### observedGeneration

**Type**: `integer`

**Description**: Geração do spec que foi observada pelo operator.

**Usage**: Usado para detectar se o operator já processou a última mudança no spec.

**Examples**:
```yaml
observedGeneration: 1
observedGeneration: 5
```

## Condições de Status

### Progressão de Status

O operator atualiza as condições conforme a função progride:

#### 1. Initial State

```yaml
status:
  conditions: []
  observedGeneration: 0
```

#### 2. Building

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: Building
      message: Building container image
  observedGeneration: 1
```

#### 3. Build Succeeded

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: BuildSucceeded
      message: Image built successfully, deploying...
    - type: BuildSucceeded
      status: "True"
      reason: BuildCompleted
      message: Image built successfully
  imageDigest: registry.example.com/my-function@sha256:abc123...
  observedGeneration: 1
```

#### 4. Ready

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: FunctionReady
      message: Function is ready and serving traffic
    - type: BuildSucceeded
      status: "True"
      reason: BuildCompleted
      message: Image built successfully
  imageDigest: registry.example.com/my-function@sha256:abc123...
  url: http://my-function.default.svc.cluster.local
  observedGeneration: 1
```

#### 5. Build Failed

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: BuildFailed
      message: "Build failed: git clone authentication failed"
    - type: BuildSucceeded
      status: "False"
      reason: BuildFailed
      message: "Git clone authentication failed"
  observedGeneration: 1
```

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

## Troubleshooting

### Comandos Úteis

```bash
# Ver Functions
kubectl get functions
kubectl get fn

# Ver detalhes
kubectl describe function my-function

# Ver status
kubectl get function my-function -o jsonpath='{.status}'

# Ver PipelineRuns
kubectl get pipelineruns

# Ver Knative Services
kubectl get ksvc

# Ver Triggers
kubectl get triggers

# Ver logs do operator
kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager -f

# Ver logs da função
kubectl logs -l serving.knative.dev/service=my-function -f
```

### Problemas Comuns

#### Build Falha

**Sintoma**: `BuildFailed` condition

**Causas**:
- Autenticação Git falhou
- Buildpack não detectou linguagem
- Registry push falhou

**Solução**:
```bash
# Ver logs do PipelineRun
kubectl get pipelineruns
kubectl logs <pipelinerun-name>-fetch-source-pod --all-containers
```

#### Função Não Responde

**Sintoma**: Timeout ou connection refused

**Causas**:
- Aplicação não escuta na porta correta
- Aplicação não inicia
- Scale-from-zero lento

**Solução**:
```bash
# Ver pods
kubectl get pods -l serving.knative.dev/service=my-function

# Ver logs
kubectl logs -l serving.knative.dev/service=my-function
```

#### Eventos Não Chegam

**Sintoma**: Função não recebe eventos

**Causas**:
- Broker não existe
- Filtros não correspondem
- Trigger não criado

**Solução**:
```bash
# Ver Broker
kubectl get broker

# Ver Trigger
kubectl get trigger
kubectl describe trigger my-function-trigger
```

## Próximos Passos

- [Criando Funções HTTP Síncronas](CREATING_HTTP_FUNCTIONS.md)
- [Criando Funções Assíncronas com Eventos](CREATING_EVENT_FUNCTIONS.md)
- [Comunicação entre Funções](INTER_FUNCTION_COMMUNICATION.md)
- [Configuração de Autenticação Git](GIT_AUTHENTICATION.md)
- [Configuração de Registry](REGISTRY_CONFIGURATION.md)

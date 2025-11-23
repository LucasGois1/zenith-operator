# Function CRD - Especificação Completa

Esta documentação descreve a especificação completa do Custom Resource Definition (CRD) `Function`.

## API Group e Version

- **API Group**: `functions.zenith.com`
- **Version**: `v1alpha1`
- **Kind**: `Function`
- **Plural**: `functions`
- **Singular**: `function`
- **Short Names**: `fn`, `func`

## Exemplo Completo

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

**Type**: `[]corev1.EnvVar`

**Description**: Lista de variáveis de ambiente para injetar no container da função. Suporta valores estáticos, referências a Secrets/ConfigMaps, e referências a campos do Pod.

**EnvVar Fields**:
- `name` (string, required): Nome da variável de ambiente
- `value` (string, optional): Valor estático da variável
- `valueFrom` (object, optional): Fonte para o valor da variável (não pode ser usado com `value`)
  - `secretKeyRef`: Referência a uma chave em um Secret
    - `name` (string): Nome do Secret
    - `key` (string): Chave dentro do Secret
    - `optional` (boolean): Se true, não falha se o Secret não existir
  - `configMapKeyRef`: Referência a uma chave em um ConfigMap
    - `name` (string): Nome do ConfigMap
    - `key` (string): Chave dentro do ConfigMap
    - `optional` (boolean): Se true, não falha se o ConfigMap não existir
  - `fieldRef`: Referência a um campo do Pod
    - `fieldPath` (string): Caminho do campo (ex: metadata.name, metadata.namespace)
  - `resourceFieldRef`: Referência a recursos do container
    - `resource` (string): Nome do recurso (ex: limits.cpu, requests.memory)

**Examples**:

**Valores estáticos**:
```yaml
env:
  - name: DATABASE_URL
    value: postgres://db.example.com/mydb
  - name: LOG_LEVEL
    value: debug
  - name: FEATURE_FLAG_X
    value: "true"
```

**Referências a Secrets**:
```yaml
env:
  - name: API_KEY
    valueFrom:
      secretKeyRef:
        name: api-credentials
        key: api-key
  - name: DATABASE_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-credentials
        key: password
  - name: OPTIONAL_TOKEN
    valueFrom:
      secretKeyRef:
        name: optional-secret
        key: token
        optional: true
```

**Referências a ConfigMaps**:
```yaml
env:
  - name: APP_CONFIG
    valueFrom:
      configMapKeyRef:
        name: app-config
        key: config.json
  - name: FEATURE_FLAGS
    valueFrom:
      configMapKeyRef:
        name: feature-flags
        key: flags
```

**Referências a campos do Pod**:
```yaml
env:
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
  - name: POD_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  - name: POD_IP
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
```

**Referências a recursos**:
```yaml
env:
  - name: CPU_LIMIT
    valueFrom:
      resourceFieldRef:
        resource: limits.cpu
  - name: MEMORY_REQUEST
    valueFrom:
      resourceFieldRef:
        resource: requests.memory
```

**Exemplo combinado**:
```yaml
env:
  - name: APP_ENV
    value: production
  - name: DATABASE_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-credentials
        key: password
  - name: CONFIG_PATH
    valueFrom:
      configMapKeyRef:
        name: app-config
        key: config-path
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
```

**Important Notes**:
- O operator valida que Secrets e ConfigMaps referenciados existem antes de deployar a função
- Se um Secret ou ConfigMap não existir, a função ficará com status `Ready=False` e reason `SecretNotFound` ou `ConfigMapNotFound`
- Use `optional: true` para recursos que podem não existir
- Mudanças nas variáveis de ambiente disparam uma nova revisão do Knative Service

#### deploy.envFrom (Optional)

**Type**: `[]corev1.EnvFromSource`

**Description**: Lista de fontes para popular variáveis de ambiente no container. Todas as chaves do Secret ou ConfigMap serão expostas como variáveis de ambiente.

**EnvFromSource Fields**:
- `secretRef`: Referência a um Secret
  - `name` (string): Nome do Secret
  - `optional` (boolean): Se true, não falha se o Secret não existir
- `configMapRef`: Referência a um ConfigMap
  - `name` (string): Nome do ConfigMap
  - `optional` (boolean): Se true, não falha se o ConfigMap não existir
- `prefix` (string, optional): Prefixo para adicionar aos nomes das variáveis

**Examples**:

**Injetar todas as chaves de um Secret**:
```yaml
envFrom:
  - secretRef:
      name: api-credentials
```

**Injetar todas as chaves de um ConfigMap**:
```yaml
envFrom:
  - configMapRef:
      name: app-config
```

**Com prefixo**:
```yaml
envFrom:
  - prefix: DB_
    secretRef:
      name: database-credentials
  - prefix: CACHE_
    configMapRef:
      name: redis-config
```

**Múltiplas fontes**:
```yaml
envFrom:
  - secretRef:
      name: api-credentials
  - configMapRef:
      name: app-config
  - secretRef:
      name: optional-secrets
      optional: true
```

**Exemplo combinado com env**:
```yaml
deploy:
  env:
    - name: APP_ENV
      value: production
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
  envFrom:
    - secretRef:
        name: api-credentials
    - configMapRef:
        name: app-config
```

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

### observability (Optional)

**Type**: `ObservabilitySpec`

**Description**: Configuração de observabilidade e distributed tracing via OpenTelemetry.

**Note**: Se especificado, o operator injeta variáveis de ambiente OpenTelemetry no container.

#### observability.tracing (Optional)

**Type**: `TracingConfig`

**Description**: Configuração de distributed tracing.

##### observability.tracing.enabled (Required if tracing specified)

**Type**: `boolean`

**Description**: Se `true`, habilita distributed tracing via OpenTelemetry.

**Default**: `false`

**Examples**:
```yaml
observability:
  tracing:
    enabled: true
```

**Behavior**:
- Quando habilitado, o operator injeta automaticamente variáveis de ambiente OpenTelemetry no container
- Se Dapr também estiver habilitado, o operator configura Dapr para propagar trace context

**Environment Variables Injected**:
- `OTEL_EXPORTER_OTLP_ENDPOINT`: Endpoint do OpenTelemetry Collector
- `OTEL_SERVICE_NAME`: Nome da função (usado para identificar o serviço nos traces)
- `OTEL_RESOURCE_ATTRIBUTES`: Atributos de recurso (namespace, version)
- `OTEL_TRACES_EXPORTER`: Protocolo de exportação (otlp)
- `OTEL_TRACES_SAMPLER`: Tipo de sampler (se samplingRate especificado)
- `OTEL_TRACES_SAMPLER_ARG`: Argumento do sampler (se samplingRate especificado)

##### observability.tracing.samplingRate (Optional)

**Type**: `string`

**Description**: Taxa de amostragem de traces (0.0 a 1.0). Se não especificado, usa a taxa padrão do OpenTelemetry Collector.

**Format**: String representando um número decimal entre 0.0 e 1.0

**Validation**: Deve corresponder ao padrão regex `^(0(\.\d+)?|1(\.0+)?)$`

**Examples**:
```yaml
# 100% sampling (captura todos os traces)
observability:
  tracing:
    enabled: true
    samplingRate: "1.0"

# 50% sampling
observability:
  tracing:
    enabled: true
    samplingRate: "0.5"

# 10% sampling
observability:
  tracing:
    enabled: true
    samplingRate: "0.1"

# 1% sampling
observability:
  tracing:
    enabled: true
    samplingRate: "0.01"
```

**Recommendations**:
- **Development**: Use `"1.0"` (100%) para capturar todos os traces
- **Staging**: Use `"0.5"` a `"1.0"` (50-100%) para funções críticas
- **Production**: Use `"0.01"` a `"0.1"` (1-10%) para funções de alto tráfego

**Example with Dapr**:
```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: payment-processor
spec:
  gitRepo: https://github.com/myorg/payment-processor
  gitRevision: main
  build:
    image: registry.example.com/payment-processor:latest
  deploy:
    dapr:
      enabled: true
      appID: payment-processor
      appPort: 8080
  observability:
    tracing:
      enabled: true
      samplingRate: "0.1"
```

**Note**: Quando Dapr e tracing estão habilitados, o operator adiciona automaticamente a anotação `dapr.io/config: tracing-config` ao pod template.

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

## Próximos Passos

- [Referência do Operator](operator-reference.md) - Comportamento e integrações do operator
- [Troubleshooting](troubleshooting.md) - Solução de problemas comuns
- [Guia de Funções HTTP](../02-guias/funcoes-http.md)
- [Guia de Funções com Eventos](../02-guias/funcoes-eventos.md)

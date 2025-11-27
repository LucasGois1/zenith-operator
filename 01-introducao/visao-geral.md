# Zenith Operator

[![Lint](https://github.com/LucasGois1/zenith-operator/actions/workflows/lint.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/lint.yml)
[![Tests](https://github.com/LucasGois1/zenith-operator/actions/workflows/test.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/test.yml)
[![E2E Tests](https://github.com/LucasGois1/zenith-operator/actions/workflows/test-e2e.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/test-e2e.yml)

Zenith Operator é um operador Kubernetes que fornece uma plataforma serverless para funções, orquestrando builds (Tekton Pipelines), deployments (Knative Serving) e invocações orientadas a eventos (Knative Eventing) através de um único Custom Resource `Function`.

## 🚀 Visão Geral

O Zenith Operator abstrai a complexidade de integrar Tekton, Knative e Dapr, permitindo que desenvolvedores definam funções serverless de forma declarativa usando apenas um Custom Resource.

### Principais Características

- **Build Automático**: Clona repositórios Git e constrói imagens de container usando Tekton Pipelines e Buildpacks
- **Serverless Deployment**: Deploy automático como Knative Services com scale-to-zero
- **Event-Driven**: Subscrição a eventos via Knative Eventing com filtros baseados em atributos
- **Service Mesh**: Integração opcional com Dapr para service discovery, pub/sub e state management
- **Comunicação entre Funções**: Suporte nativo para comunicação HTTP entre funções
- **Distributed Tracing**: Rastreamento distribuído automático via OpenTelemetry para visualizar fluxos de requisições
- **Imagens Imutáveis**: Rastreamento de image digests para reprodutibilidade e segurança

## 📚 Documentação

### Guias de Uso

- **[Criando Funções HTTP Síncronas](../02-guias/funcoes-http.md)** - Como criar funções que respondem a requisições HTTP
- **[Criando Funções Assíncronas com Eventos](../02-guias/funcoes-eventos.md)** - Como criar funções que processam eventos assíncronos
- **[Comunicação entre Funções](../02-guias/comunicacao-funcoes.md)** - Como implementar comunicação entre múltiplas funções
- **[Observabilidade e Distributed Tracing](../02-guias/observabilidade.md)** - Como usar OpenTelemetry para rastrear requisições entre funções

### Referência Técnica

- **[Especificação do CRD Function](../04-referencia/function-crd.md)** - Documentação completa de todos os campos do Custom Resource
- **[Referência do Operator](../04-referencia/operator-reference.md)** - Comportamento interno e integrações do operator
- **[Configuração de Autenticação Git](../02-guias/autenticacao-git.md)** - Como configurar autenticação para repositórios Git privados
- **[Configuração de Registry](../05-operacoes/configuracao-registry.md)** - Como configurar registries de container

## 🎯 Casos de Uso

### 1. Funções HTTP Síncronas

Funções que respondem a requisições HTTP síncronas, ideais para APIs REST, webhooks e microserviços.

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: hello-api
spec:
  gitRepo: https://github.com/myorg/hello-function
  gitRevision: main
  build:
    image: registry.example.com/hello-api:latest
  deploy: {}
```

### 2. Funções Assíncronas com Eventos

Funções que processam eventos de forma assíncrona, ideais para processamento de dados, notificações e workflows event-driven.

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
  deploy: {}
  eventing:
    broker: default
    filters:
      type: com.example.order.created
      source: payment-service
```

### 3. Comunicação entre Funções

Múltiplas funções que se comunicam via HTTP, ideais para arquiteturas de microserviços e sistemas distribuídos.

```yaml
# transaction-processor chama balance-manager que chama audit-logger
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: transaction-processor
spec:
  gitRepo: https://github.com/myorg/transaction-processor
  gitRevision: main
  build:
    image: registry.example.com/transaction-processor:latest
  deploy:
    env:
      - name: BALANCE_MANAGER_URL
        value: http://balance-manager.default.svc.cluster.local
```

## 🛠️ Instalação

### Pré-requisitos

- Kubernetes 1.33.0+
- Tekton Pipelines v0.50+
- Knative Serving v1.20+
- Knative Eventing v1.20+ (opcional, para event-driven functions)
- Envoy Gateway v1.6+ (para ingress)

### Instalação via Helm

```bash
# Adicionar o repositório Helm
helm repo add zenith https://lucasgois1.github.io/zenith-operator

# Instalar o operator
helm install zenith-operator zenith/zenith-operator \
  --namespace zenith-operator-system \
  --create-namespace
```

### Instalação via Kustomize

```bash
# Instalar CRDs
make install

# Deploy do operator
make deploy IMG=ghcr.io/lucasgois1/zenith-operator:latest
```

## 🚦 Quick Start

1. **Criar um Secret para autenticação Git** (se usar repositório privado):

```bash
kubectl create secret generic github-auth \
  --from-literal=username=myuser \
  --from-literal=password=mytoken \
  --type=kubernetes.io/basic-auth

kubectl annotate secret github-auth \
  tekton.dev/git-0=https://github.com
```

2. **Criar sua primeira função**:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-first-function
spec:
  gitRepo: https://github.com/LucasGois1/zenith-test-functions
  gitRevision: main
  gitAuthSecretName: github-auth
  build:
    image: registry.example.com/my-first-function:latest
  deploy: {}
EOF
```

3. **Verificar o status**:

```bash
kubectl get functions
kubectl describe function my-first-function
```

4. **Acessar a função**:

```bash
# Obter a URL da função
FUNCTION_URL=$(kubectl get function my-first-function -o jsonpath='{.status.url}')
echo "Function URL: $FUNCTION_URL"

# Fazer uma requisição
curl $FUNCTION_URL
```

## 🧪 Desenvolvimento

### Executar testes localmente

```bash
# Testes unitários
make test

# Testes E2E
make test-e2e

# Testes Chainsaw específicos
make test-chainsaw-basic        # Teste básico de função
make test-chainsaw-eventing     # Teste de eventing
make test-chainsaw-integration  # Teste de integração entre funções
```

### Desenvolvimento local

```bash
# Setup do ambiente de desenvolvimento
make dev-up

# Rebuild e redeploy rápido
make dev-redeploy

# Limpar ambiente
make dev-down
```

## 📖 Exemplos

Veja o diretório [config/samples/](config/samples/) para exemplos completos de Functions.

## 🤝 Contribuindo

Contribuições são bem-vindas! Por favor, abra issues e pull requests no GitHub.

## 📄 Licença

Este projeto está licenciado sob a Apache License 2.0 - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🔗 Links Úteis

- [Documentação Completa](../README.md)
- [Início Rápido](inicio-rapido.md)
- [Exemplos](https://github.com/LucasGois1/zenith-operator/tree/main/config/samples)
- [Testes Chainsaw](https://github.com/LucasGois1/zenith-operator/tree/main/test/chainsaw)
- [Issues](https://github.com/LucasGois1/zenith-operator/issues)

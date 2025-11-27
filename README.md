# Zenith Operator

[![codecov](https://codecov.io/github/LucasGois1/zenith-operator/branch/main/graph/badge.svg?token=2QNLMH3D7H)](https://codecov.io/github/LucasGois1/zenith-operator)
[![Lint](https://github.com/LucasGois1/zenith-operator/actions/workflows/lint.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/lint.yml)
[![Tests](https://github.com/LucasGois1/zenith-operator/actions/workflows/test.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/test.yml)
[![E2E Tests](https://github.com/LucasGois1/zenith-operator/actions/workflows/test-e2e.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/test-e2e.yml)

Zenith Operator é um operador Kubernetes que fornece uma plataforma serverless para funções, orquestrando builds (Tekton Pipelines), deployments (Knative Serving) e invocações orientadas a eventos (Knative Eventing) através de um único Custom Resource `Function`.

## 🚀 Início Rápido

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: hello-function
spec:
  gitRepo: https://github.com/myorg/hello-function
  gitRevision: main
  build:
    image: registry.example.com/hello-function:latest
  deploy: {}
```

## 📖 Documentação

**[Acesse a documentação completa →](docs/)**

- **[Introdução](docs/01-introducao/)** - Visão geral, instalação e início rápido
- **[Guias](docs/02-guias/)** - Tutoriais práticos para criar funções
- **[Conceitos](docs/03-conceitos/)** - Arquitetura e conceitos fundamentais
- **[Referência](docs/04-referencia/)** - Especificação completa da API
- **[Operações](docs/05-operacoes/)** - Configuração e gerenciamento

## ✨ Principais Características

- **Build Automático**: Clona repositórios Git e constrói imagens usando Tekton Pipelines e Buildpacks
- **Serverless Deployment**: Deploy automático como Knative Services com scale-to-zero
- **Event-Driven**: Subscrição a eventos via Knative Eventing com filtros
- **Service Mesh**: Integração opcional com Dapr para service discovery e pub/sub
- **Distributed Tracing**: Rastreamento automático via OpenTelemetry
- **Imagens Imutáveis**: Rastreamento de image digests para reprodutibilidade

## 🛠️ Instalação

### Via Helm

**Desenvolvimento Local (kind/Minikube):**
```bash
helm repo add zenith https://lucasgois1.github.io/zenith-operator

# Baixar o values-dev.yaml
curl -O https://raw.githubusercontent.com/LucasGois1/zenith-operator/main/charts/zenith-operator/values-dev.yaml

# Instalar com o profile de desenvolvimento (já inclui MetalLB, registry local, etc.)
helm install zenith-operator zenith/zenith-operator \
  -f values-dev.yaml \
  --namespace zenith-operator-system \
  --create-namespace
```

**Produção (GKE/EKS/AKS):**
```bash
helm repo add zenith https://lucasgois1.github.io/zenith-operator
helm install zenith-operator zenith/zenith-operator \
  --namespace zenith-operator-system \
  --create-namespace
```

> **Nota:** O MetalLB é necessário apenas em clusters locais (kind/Minikube) que não possuem suporte nativo a LoadBalancer. Em clouds gerenciadas (GKE, EKS, AKS), o load balancer da cloud é usado automaticamente.

### Via Kustomize

```bash
make install  # Instalar CRDs
make deploy IMG=ghcr.io/lucasgois1/zenith-operator:latest
```

**[Guia completo de instalação →](docs/01-introducao/instalacao.md)**

## 🎯 Casos de Uso

### Funções HTTP Síncronas
APIs REST, webhooks e microserviços que respondem a requisições HTTP.

**[Ver guia →](docs/02-guias/funcoes-http.md)**

### Funções Assíncronas com Eventos
Processamento de eventos, notificações e workflows event-driven.

**[Ver guia →](docs/02-guias/funcoes-eventos.md)**

### Comunicação entre Funções
Arquiteturas de microserviços com múltiplas funções se comunicando.

**[Ver guia →](docs/02-guias/comunicacao-funcoes.md)**

## 🧪 Desenvolvimento

```bash
# Setup completo do ambiente
make dev-up

# Rebuild e redeploy rápido
make dev-redeploy

# Executar testes
make test
make test-chainsaw
```

## 📄 Licença

Apache License 2.0 - veja [LICENSE](LICENSE) para detalhes.

## 🤝 Contribuindo

Contribuições são bem-vindas! Abra issues e pull requests no GitHub.

## 🔗 Links

- [Documentação](docs/)
- [Exemplos](config/samples/)
- [Issues](https://github.com/LucasGois1/zenith-operator/issues)

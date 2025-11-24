# Zenith Operator - Documentação

Bem-vindo à documentação do Zenith Operator! Esta é uma plataforma serverless para Kubernetes que simplifica o desenvolvimento e deployment de funções através de um único Custom Resource.

## 🚀 O que é o Zenith Operator?

O Zenith Operator é um operador Kubernetes que abstrai a complexidade de integrar múltiplas tecnologias cloud-native (Tekton Pipelines, Knative Serving, Knative Eventing e Dapr) em uma experiência simples e declarativa.

Com o Zenith Operator, você pode:

- **Construir** imagens de container automaticamente a partir do código-fonte (sem Dockerfile)
- **Deployar** funções serverless com auto-scaling e scale-to-zero
- **Conectar** funções a eventos para processamento assíncrono
- **Comunicar** entre funções usando HTTP ou service mesh
- **Rastrear** requisições distribuídas com OpenTelemetry

Tudo isso através de um único Custom Resource `Function`.

## 📖 Navegação da Documentação

### [01. Introdução](01-introducao/)

Comece aqui se você é novo no Zenith Operator.

- **[Visão Geral](01-introducao/visao-geral.md)** - Entenda o que é o operator e suas principais características
- **[Instalação](01-introducao/instalacao.md)** - Instale o operator em seu cluster Kubernetes
- **[Início Rápido](01-introducao/inicio-rapido.md)** - Crie sua primeira função em 5 minutos

### [02. Guias](02-guias/)

Tutoriais práticos para criar diferentes tipos de funções.

- **[Funções HTTP](02-guias/funcoes-http.md)** - APIs REST, webhooks e microserviços síncronos
- **[Funções com Eventos](02-guias/funcoes-eventos.md)** - Processamento assíncrono orientado a eventos
- **[Comunicação entre Funções](02-guias/comunicacao-funcoes.md)** - Arquiteturas de microserviços distribuídos
- **[Autenticação Git](02-guias/autenticacao-git.md)** - Configure acesso a repositórios privados
- **[Observabilidade](02-guias/observabilidade.md)** - Distributed tracing com OpenTelemetry

### [03. Conceitos](03-conceitos/)

Entenda a arquitetura e os conceitos fundamentais.

- **[Arquitetura](03-conceitos/arquitetura.md)** - Diagramas e explicações da arquitetura completa
- **[Ciclo de Vida das Funções](03-conceitos/ciclo-vida-funcoes.md)** - Como as funções são criadas, atualizadas e removidas

### [04. Referência](04-referencia/)

Documentação técnica completa da API.

- **[Function CRD](04-referencia/function-crd.md)** - Especificação completa de todos os campos
- **[Referência do Operator](04-referencia/operator-reference.md)** - Comportamento e integrações internas
- **[Troubleshooting](04-referencia/troubleshooting.md)** - Solução de problemas comuns

### [05. Operações](05-operacoes/)

Configuração e gerenciamento em produção.

- **[Helm Chart](05-operacoes/helm-chart.md)** - Instalação via Helm e configuração da stack
- **[Configuração de Registry](05-operacoes/configuracao-registry.md)** - Setup de container registries

## 🎯 Casos de Uso Comuns

### API REST Síncrona

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

### Processamento de Eventos

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
```

### Microserviços com Service Mesh

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: payment-service
spec:
  gitRepo: https://github.com/myorg/payment-service
  gitRevision: main
  build:
    image: registry.example.com/payment-service:latest
  deploy:
    dapr:
      enabled: true
      appID: payment-service
      appPort: 8080
```

## 🚦 Início Rápido

1. **Instale o operator** seguindo o [guia de instalação](01-introducao/instalacao.md)

2. **Crie sua primeira função** com o [tutorial de início rápido](01-introducao/inicio-rapido.md)

3. **Explore os guias** para aprender recursos avançados:
   - [Funções HTTP](02-guias/funcoes-http.md)
   - [Funções com Eventos](02-guias/funcoes-eventos.md)
   - [Comunicação entre Funções](02-guias/comunicacao-funcoes.md)

## 🔍 Encontrando o que Você Precisa

### Estou começando agora
→ Comece com [Introdução](01-introducao/) e siga o [Início Rápido](01-introducao/inicio-rapido.md)

### Quero criar uma função HTTP
→ Veja o guia [Funções HTTP](02-guias/funcoes-http.md)

### Quero processar eventos
→ Veja o guia [Funções com Eventos](02-guias/funcoes-eventos.md)

### Preciso configurar autenticação Git
→ Veja o guia [Autenticação Git](02-guias/autenticacao-git.md)

### Estou tendo problemas
→ Consulte o [Troubleshooting](04-referencia/troubleshooting.md)

### Preciso da referência completa da API
→ Veja [Function CRD](04-referencia/function-crd.md)

### Quero entender como funciona internamente
→ Leia sobre [Arquitetura](03-conceitos/arquitetura.md) e [Referência do Operator](04-referencia/operator-reference.md)

## 🤝 Contribuindo

Contribuições são bem-vindas! Visite o [repositório no GitHub](https://github.com/LucasGois1/zenith-operator) para:

- Reportar bugs e problemas
- Sugerir novas funcionalidades
- Contribuir com código
- Melhorar a documentação

## 📄 Licença

Este projeto está licenciado sob a Apache License 2.0.

## 🔗 Links Úteis

- [Repositório GitHub](https://github.com/LucasGois1/zenith-operator)
- [Exemplos de Funções](https://github.com/LucasGois1/zenith-test-functions)
- [Issues e Suporte](https://github.com/LucasGois1/zenith-operator/issues)

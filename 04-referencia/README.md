# Referência

Documentação técnica completa da API do Zenith Operator e guia de troubleshooting.

## Conteúdo

### [Function CRD - Especificação Completa](function-crd.md)
Referência completa do Custom Resource Definition `Function`.

**Tópicos abordados:**
- API Group e Version
- Todos os campos do Spec (gitRepo, build, deploy, eventing, observability)
- Todos os campos do Status (conditions, imageDigest, url)
- Exemplos completos para cada configuração
- Progressão de status durante o ciclo de vida
- Validações e constraints

**Use este documento quando:**
- Precisar saber todos os campos disponíveis
- Quiser entender as opções de configuração
- Estiver escrevendo manifestos YAML
- Precisar de exemplos de configuração específicos

### [Referência do Operator](operator-reference.md)
Comportamento interno do operator e suas integrações.

**Tópicos abordados:**
- Reconciliation loop e triggers
- Integração com Tekton (PipelineRun, ServiceAccount, image digest)
- Integração com Knative (Service, auto-scaling, URLs)
- Integração com Dapr (sidecar injection, features)
- Autenticação e Secrets (Git, Registry)
- Variáveis de ambiente

**Use este documento quando:**
- Quiser entender como o operator funciona internamente
- Precisar debugar problemas de reconciliação
- Estiver configurando autenticação
- Quiser entender as integrações com outras tecnologias

### [Troubleshooting](troubleshooting.md)
Guia completo de solução de problemas e debugging.

**Tópicos abordados:**
- Comandos úteis para diagnóstico
- Problemas comuns e soluções:
  - Build falha (autenticação Git, buildpack, registry)
  - Função não responde (porta, inicialização, cold start)
  - Eventos não chegam (broker, filtros, trigger)
  - URL não acessível (gateway, DNS)
- Logs e debugging
- Validação de configuração

**Use este documento quando:**
- Encontrar erros ou problemas
- Precisar debugar uma função
- Quiser validar sua configuração
- Precisar coletar informações para reportar um bug

## Estrutura da API

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: example
spec:
  gitRepo: https://github.com/org/repo
  gitRevision: main
  gitAuthSecretName: github-auth
  build:
    image: registry.io/image
    registrySecretName: registry-creds
  deploy:
    dapr:
      enabled: true
      appID: example
      appPort: 8080
    env: []
    envFrom: []
  eventing:
    broker: default
    filters:
      type: event.type
  observability:
    tracing:
      enabled: true
      samplingRate: "0.1"
status:
  conditions: []
  imageDigest: registry.io/image@sha256:...
  url: http://example.default.svc.cluster.local
  observedGeneration: 1
```

## Próximos Passos

- **[Guias](../02-guias/)** - Aplique a referência em tutoriais práticos
- **[Conceitos](../03-conceitos/)** - Entenda a arquitetura por trás da API
- **[Operações](../05-operacoes/)** - Configure o ambiente de produção

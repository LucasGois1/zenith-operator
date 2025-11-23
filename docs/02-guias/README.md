# Guias

Tutoriais práticos e guias passo a passo para usar o Zenith Operator.

## Conteúdo

### [Funções HTTP](funcoes-http.md)
Aprenda a criar funções HTTP síncronas que respondem a requisições REST. Ideal para APIs, webhooks e microserviços.

**Tópicos abordados:**
- Estrutura do código da função
- Configuração do Function CR
- Monitoramento do build
- Acesso via URL pública
- Variáveis de ambiente e configurações avançadas

### [Funções com Eventos](funcoes-eventos.md)
Crie funções assíncronas orientadas a eventos usando Knative Eventing. Perfeito para processamento assíncrono e workflows event-driven.

**Tópicos abordados:**
- Arquitetura event-driven
- Configuração de Brokers e Triggers
- Filtros de eventos CloudEvents
- Envio e processamento de eventos
- Padrões avançados (DLQ, fan-out)

### [Comunicação entre Funções](comunicacao-funcoes.md)
Implemente comunicação HTTP entre múltiplas funções para criar arquiteturas de microserviços complexas.

**Tópicos abordados:**
- Padrões de URL de serviço
- Request-response síncrono
- Fire-and-forget assíncrono
- Service discovery
- Timeout, retry e circuit breaker
- Integração com Dapr

### [Autenticação Git](autenticacao-git.md)
Configure autenticação para repositórios Git privados usando HTTPS ou SSH.

**Tópicos abordados:**
- HTTPS com GitHub Token
- SSH com Deploy Keys
- Como funciona a autenticação
- Troubleshooting de problemas comuns
- Suporte para GitLab, Bitbucket e servidores privados

### [Observabilidade](observabilidade.md)
Configure distributed tracing e observabilidade para suas funções usando OpenTelemetry.

**Tópicos abordados:**
- Configuração de tracing
- Integração com OpenTelemetry Collector
- Visualização de traces
- Sampling rates
- Integração com Dapr

## Próximos Passos

Após dominar os guias práticos:

- **[Conceitos](../03-conceitos/)** - Aprofunde-se na arquitetura do operator
- **[Referência](../04-referencia/)** - Consulte a especificação completa da API
- **[Operações](../05-operacoes/)** - Configure o ambiente de produção

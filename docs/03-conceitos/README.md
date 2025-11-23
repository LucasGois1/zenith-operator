# Conceitos

Entenda a arquitetura e os conceitos fundamentais do Zenith Operator.

## Conteúdo

### [Arquitetura](arquitetura.md)
Documentação completa da arquitetura do Zenith Operator com diagramas detalhados.

**Tópicos abordados:**
- Visão geral de alto nível
- Estrutura do Function CRD
- Fluxo de reconciliação do operator
- Experiência completa do desenvolvedor
- Integração com Tekton Pipelines
- Integração com Knative Serving
- Arquitetura event-driven
- Recursos e funcionalidades principais

**Diagramas incluídos:**
- Arquitetura de alto nível
- Estrutura do Custom Resource
- Fluxo de reconciliação
- Pipeline de build do Tekton
- Deployment do Knative Service
- Roteamento de eventos

### [Ciclo de Vida das Funções](ciclo-vida-funcoes.md)
*(Em desenvolvimento)* Detalhes sobre o ciclo de vida completo de uma função, desde a criação até a remoção.

## Como o Operator Funciona

O Zenith Operator abstrai a complexidade de múltiplas tecnologias cloud-native:

1. **Tekton Pipelines** - Constrói imagens de container a partir do código-fonte
2. **Knative Serving** - Faz deploy e gerencia o auto-scaling das funções
3. **Knative Eventing** - Roteia eventos para funções event-driven
4. **Dapr** (opcional) - Fornece service mesh, pub/sub e state management

Tudo isso é controlado através de um único Custom Resource `Function`, tornando o desenvolvimento serverless simples e declarativo.

## Próximos Passos

Após entender os conceitos:

- **[Referência](../04-referencia/)** - Consulte a especificação completa da API
- **[Guias](../02-guias/)** - Aplique os conceitos em tutoriais práticos
- **[Operações](../05-operacoes/)** - Configure o ambiente de produção

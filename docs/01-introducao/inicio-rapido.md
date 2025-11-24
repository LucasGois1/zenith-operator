# Início Rápido

Este guia mostra como criar e deployar sua primeira função serverless com o Zenith Operator em 5 minutos.

## Pré-requisitos

Antes de começar, certifique-se de ter:

- Cluster Kubernetes rodando (kind, minikube, GKE, EKS, etc.)
- Zenith Operator instalado ([veja o guia de instalação](instalacao.md))
- `kubectl` configurado para acessar seu cluster
- Container registry acessível (Docker Hub, GCR, etc.)

## Passo 1: Criar Secret de Autenticação Git (Opcional)

Se você estiver usando um repositório Git privado, crie um Secret para autenticação:

```bash
kubectl create secret generic github-auth \
  --from-literal=username=myuser \
  --from-literal=password=mytoken \
  --type=kubernetes.io/basic-auth

kubectl annotate secret github-auth \
  tekton.dev/git-0=https://github.com
```

**Nota**: Para repositórios públicos, você pode pular este passo.

## Passo 2: Criar sua Primeira Função

Crie um arquivo `my-first-function.yaml` com o seguinte conteúdo:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-first-function
spec:
  # Repositório Git com o código da função
  gitRepo: https://github.com/LucasGois1/zenith-test-functions
  gitRevision: main
  
  # Secret de autenticação (remova se usar repositório público)
  gitAuthSecretName: github-auth
  
  # Configuração de build
  build:
    image: registry.example.com/my-first-function:latest
  
  # Configuração de deploy
  deploy: {}
```

**Importante**: Substitua `registry.example.com` pelo seu registry (ex: `docker.io/myuser`).

Aplique o recurso:

```bash
kubectl apply -f my-first-function.yaml
```

## Passo 3: Verificar o Status

Acompanhe o progresso da função:

```bash
# Ver todas as funções
kubectl get functions

# Ver detalhes da função
kubectl describe function my-first-function

# Ver o PipelineRun (build)
kubectl get pipelineruns

# Ver logs do build
kubectl logs -f <pipelinerun-name>-fetch-source-pod --all-containers
```

O status da função passará por estas fases:

1. **Building** - Construindo a imagem de container
2. **BuildSucceeded** - Build completado com sucesso
3. **Ready** - Função deployada e pronta para receber requisições

## Passo 4: Acessar a Função

Após a função estar pronta (status `Ready`), você pode acessá-la:

```bash
# Obter a URL da função
FUNCTION_URL=$(kubectl get function my-first-function -o jsonpath='{.status.url}')
echo "Function URL: $FUNCTION_URL"

# Fazer uma requisição
curl $FUNCTION_URL
```

**Resposta esperada**:
```json
{
  "status": "ok",
  "message": "Hello from Zenith Function!",
  "type": "http-sync"
}
```

## Passo 5: Atualizar a Função

Para atualizar a função, faça mudanças no código e push para o Git, depois force um rebuild:

```bash
# Opção 1: Adicionar anotação para forçar rebuild
kubectl annotate function my-first-function \
  rebuild=$(date +%s) --overwrite

# Opção 2: Atualizar a revisão Git
kubectl patch function my-first-function \
  --type merge \
  -p '{"spec":{"gitRevision":"v2.0.0"}}'
```

## Passo 6: Limpar Recursos

Para remover a função:

```bash
kubectl delete function my-first-function
```

O operator automaticamente remove todos os recursos relacionados (PipelineRun, Knative Service, etc.).

## Próximos Passos

Agora que você criou sua primeira função, explore recursos mais avançados:

### Funções HTTP Síncronas
Aprenda a criar APIs REST e webhooks:
- [Guia de Funções HTTP](../02-guias/funcoes-http.md)

### Funções Assíncronas com Eventos
Crie funções orientadas a eventos:
- [Guia de Funções com Eventos](../02-guias/funcoes-eventos.md)

### Comunicação entre Funções
Implemente arquiteturas de microserviços:
- [Guia de Comunicação entre Funções](../02-guias/comunicacao-funcoes.md)

### Configurações Avançadas
Explore todas as opções de configuração:
- [Especificação Completa do CRD](../04-referencia/function-crd.md)
- [Referência do Operator](../04-referencia/operator-reference.md)

## Troubleshooting

### Build Falha

Se o build falhar, verifique os logs:

```bash
kubectl get pipelineruns
kubectl logs <pipelinerun-name>-fetch-source-pod --all-containers
```

Problemas comuns:
- **Autenticação Git**: Verifique se o Secret está correto
- **Registry**: Verifique se você tem permissão para push
- **Buildpack**: Certifique-se de ter arquivos como `go.mod`, `package.json` na raiz

### Função Não Responde

Se a função não responder:

```bash
# Ver pods da função
kubectl get pods -l serving.knative.dev/service=my-first-function

# Ver logs da função
kubectl logs -l serving.knative.dev/service=my-first-function
```

Problemas comuns:
- **Porta incorreta**: Certifique-se de escutar na porta da variável `PORT`
- **Aplicação não inicia**: Verifique logs do pod

### Mais Ajuda

Para mais informações sobre troubleshooting:
- [Guia de Troubleshooting](../04-referencia/troubleshooting.md)

## Exemplos Completos

Veja exemplos completos no repositório:
- [zenith-test-functions](https://github.com/LucasGois1/zenith-test-functions) - Funções de exemplo
- [config/samples/](https://github.com/LucasGois1/zenith-operator/tree/main/config/samples) - Exemplos de Function CRs

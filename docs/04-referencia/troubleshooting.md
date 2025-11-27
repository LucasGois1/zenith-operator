# Troubleshooting - Solução de Problemas

Este guia ajuda a diagnosticar e resolver problemas comuns ao usar o Zenith Operator.

## Comandos Úteis

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

## Problemas Comuns

### Build Falha

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

**Detalhes por Causa**:

#### 1. Autenticação Git Falhou

**Mensagem de erro**: `fatal: could not read Username` ou `Permission denied (publickey)`

**Verificações**:
```bash
# Verificar se o Secret existe
kubectl get secret github-auth -n seu-namespace

# Verificar anotações do Secret
kubectl get secret github-auth -n seu-namespace -o jsonpath='{.metadata.annotations}'

# Verificar se o ServiceAccount foi criado
kubectl get serviceaccount <function-name>-sa -n seu-namespace

# Verificar se o Secret está anexado ao ServiceAccount
kubectl get serviceaccount <function-name>-sa -n seu-namespace -o yaml
```

**Soluções**:
- Para HTTPS: Verifique se o token tem permissões corretas (scope `repo` para Classic PAT ou `Contents: Read` para Fine-grained)
- Para SSH: Verifique se a chave pública está registrada como Deploy Key no GitHub
- Verifique se a anotação `tekton.dev/git-0` corresponde à URL do repositório

#### 2. Buildpack Não Detectou Linguagem

**Mensagem de erro**: `ERROR: No buildpack groups passed detection`

**Causas**:
- Arquivos de configuração da linguagem não estão na raiz do repositório
- Linguagem não suportada pelos buildpacks padrão

**Soluções**:
```bash
# Verificar estrutura do repositório
# Certifique-se de ter na RAIZ:
# - Go: go.mod
# - Node.js: package.json
# - Python: requirements.txt ou Pipfile
# - Java: pom.xml ou build.gradle
```

#### 3. Registry Push Falhou

**Mensagem de erro**: `401 Unauthorized` ou `denied: requested access to the resource is denied`

**Verificações**:
```bash
# Verificar se o Secret do registry existe
kubectl get secret registry-credentials -n seu-namespace

# Verificar se está anexado ao ServiceAccount
kubectl get serviceaccount <function-name>-sa -n seu-namespace -o yaml | grep imagePullSecrets
```

**Soluções**:
- Verifique as credenciais do registry
- Teste manualmente: `docker login <registry-url> -u <username> -p <password>`
- Para registries privados, certifique-se de que o Secret está corretamente configurado

### Função Não Responde

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

**Detalhes por Causa**:

#### 1. Porta Incorreta

**Problema**: Aplicação escuta em porta diferente da esperada

**Verificação**:
```bash
# Ver logs da aplicação
kubectl logs -l serving.knative.dev/service=my-function | grep -i "listening\|port"
```

**Solução**:
- Certifique-se de que sua aplicação lê a variável de ambiente `PORT`
- Padrão é `8080`, mas Knative pode usar outras portas
- Exemplo em Go:
  ```go
  port := os.Getenv("PORT")
  if port == "" {
      port = "8080"
  }
  ```

#### 2. Aplicação Não Inicia

**Problema**: Container crashloop ou erro de inicialização

**Verificação**:
```bash
# Ver eventos do pod
kubectl get pods -l serving.knative.dev/service=my-function
kubectl describe pod <pod-name>

# Ver logs completos
kubectl logs <pod-name> --all-containers
```

**Soluções Comuns**:
- Verifique dependências faltando
- Verifique variáveis de ambiente necessárias
- Verifique permissões de arquivo
- Verifique se o comando de inicialização está correto

#### 3. Scale-from-Zero Lento

**Problema**: Primeira requisição demora muito (cold start)

**Explicação**: Knative precisa criar o pod antes de processar a requisição

**Soluções**:
- Configure timeout maior no cliente HTTP
- Configure min-scale para manter pelo menos 1 pod:
  ```yaml
  apiVersion: serving.knative.dev/v1
  kind: Service
  metadata:
    name: my-function
  spec:
    template:
      metadata:
        annotations:
          autoscaling.knative.dev/min-scale: "1"
  ```

### Eventos Não Chegam

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

**Detalhes por Causa**:

#### 1. Broker Não Existe

**Verificação**:
```bash
kubectl get broker <broker-name> -n <namespace>
```

**Solução**:
```bash
# Criar Broker
cat <<EOF | kubectl apply -f -
apiVersion: eventing.knative.dev/v1
kind: Broker
metadata:
  name: default
  namespace: default
EOF
```

#### 2. Filtros Não Correspondem

**Problema**: Atributos do evento não correspondem aos filtros do Trigger

**Verificação**:
```bash
# Ver filtros do Trigger
kubectl get trigger my-function-trigger -o yaml | grep -A 10 filter
```

**Solução**:
- Certifique-se de que os eventos enviados têm os atributos corretos
- Teste enviando evento com atributos correspondentes:
  ```bash
  curl -v http://broker-ingress.knative-eventing.svc.cluster.local/default/default \
    -X POST \
    -H "Ce-Id: test-123" \
    -H "Ce-Specversion: 1.0" \
    -H "Ce-Type: com.example.order.created" \
    -H "Ce-Source: payment-service" \
    -H "Content-Type: application/json" \
    -d '{"test": true}'
  ```

#### 3. Trigger Não Criado

**Verificação**:
```bash
kubectl get trigger
kubectl describe function my-function
```

**Solução**:
- Verifique se `spec.eventing` está configurado no Function CR
- Verifique logs do operator para erros:
  ```bash
  kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager
  ```

### URL Não Acessível

**Sintoma**: Não consegue acessar a URL da função

**Causas**:
- Envoy Gateway não configurado
- HTTPRoute não criado
- DNS não resolve

**Solução**:
```bash
# Verificar Envoy Gateway
kubectl get svc -n envoy-gateway-system

# Verificar HTTPRoute
kubectl get httproute

# Verificar Gateway
kubectl get gateway -n knative-serving
```

**Detalhes**:

#### Acesso Interno (Cluster)

**URL**: `http://<function-name>.<namespace>.svc.cluster.local`

**Teste**:
```bash
kubectl run curl-pod --image=curlimages/curl --rm -it --restart=Never -- \
  curl -v http://my-function.default.svc.cluster.local
```

#### Acesso Externo (Público)

**Requisitos**:
- Envoy Gateway instalado
- Gateway configurado
- LoadBalancer ou NodePort configurado

**Teste**:
```bash
# Obter IP do Envoy Gateway
ENVOY_IP=$(kubectl get svc -n envoy-gateway-system \
  -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway \
  -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')

# Obter hostname da função
FUNCTION_HOST=$(kubectl get function my-function -o jsonpath='{.status.url}' | sed 's|http://||')

# Fazer requisição com Host header
curl -H "Host: $FUNCTION_HOST" http://$ENVOY_IP/
```

### LoadBalancer Service em Estado "Pending" (kind/Minikube)

**Sintoma**: O Service do Envoy Gateway fica em estado "Pending" e não recebe IP externo

**Verificação**:
```bash
# Verificar status do Service
kubectl get svc -n envoy-gateway-system

# Se EXTERNAL-IP mostrar <pending>, o MetalLB não está funcionando
```

**Causa**: Clusters locais (kind/Minikube) não possuem suporte nativo a LoadBalancer. O MetalLB é necessário para fornecer IPs externos.

**Solução**:

1. **Verificar se o MetalLB foi habilitado na instalação:**
```bash
# Verificar se o MetalLB está instalado
kubectl get pods -n metallb-system

# Se não houver pods, reinstale com MetalLB habilitado:
helm upgrade zenith-operator zenith/zenith-operator \
  --set metallb.enabled=true \
  --namespace zenith-operator-system
```

2. **Verificar se o IPAddressPool foi criado:**
```bash
kubectl get ipaddresspool -n metallb-system
kubectl get l2advertisement -n metallb-system
```

3. **Verificar logs do MetalLB:**
```bash
kubectl logs -n metallb-system -l app=metallb -c controller
```

> **Nota:** Em clouds gerenciadas (GKE/EKS/AKS), NÃO habilite o MetalLB. O LoadBalancer nativo da cloud é usado automaticamente.

### Status da Function Mostra "GitAuthMissing"

**Sintoma**: Condition com reason `GitAuthMissing`

**Causa**: Secret especificado em `gitAuthSecretName` não existe

**Solução**:
```bash
# Verificar se o Secret existe
kubectl get secret <secret-name> -n <namespace>

# Se não existir, criar conforme documentação
# Ver: docs/02-guias/autenticacao-git.md
```

### Dapr Sidecar Não Injeta

**Sintoma**: Pod não tem container Dapr

**Causas**:
- Dapr não instalado no cluster
- Namespace não tem label para injeção Dapr
- Configuração incorreta no Function CR

**Verificação**:
```bash
# Verificar se Dapr está instalado
kubectl get pods -n dapr-system

# Verificar annotations do pod
kubectl get pod <pod-name> -o yaml | grep -A 5 annotations

# Verificar se tem sidecar
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[*].name}'
```

**Soluções**:
- Instalar Dapr: `helm install dapr dapr/dapr --namespace dapr-system`
- Verificar configuração no Function CR:
  ```yaml
  spec:
    deploy:
      dapr:
        enabled: true
        appID: my-function
        appPort: 8080
  ```

### Imagem Não Atualiza

**Sintoma**: Função continua usando imagem antiga após rebuild

**Causa**: Knative Service não foi atualizado com novo digest

**Verificação**:
```bash
# Ver imageDigest no Function status
kubectl get function my-function -o jsonpath='{.status.imageDigest}'

# Ver imagem no Knative Service
kubectl get ksvc my-function -o jsonpath='{.spec.template.spec.containers[0].image}'
```

**Solução**:
```bash
# Forçar reconciliação
kubectl annotate function my-function \
  reconcile=$(date +%s) --overwrite

# Ou deletar e recriar o PipelineRun
kubectl delete pipelinerun -l function=my-function
```

## Logs e Debugging

### Ver Logs do Operator

```bash
# Logs em tempo real
kubectl logs -n zenith-operator-system \
  deployment/zenith-operator-controller-manager -f

# Logs com filtro
kubectl logs -n zenith-operator-system \
  deployment/zenith-operator-controller-manager \
  | grep "my-function"
```

### Ver Logs de Build

```bash
# Listar PipelineRuns
kubectl get pipelineruns

# Ver logs do git-clone
kubectl logs <pipelinerun-name>-fetch-source-pod \
  -c step-clone --tail=50

# Ver logs do buildpacks
kubectl logs <pipelinerun-name>-build-and-push-pod \
  -c step-build --tail=100
```

### Ver Logs da Função

```bash
# Logs em tempo real
kubectl logs -l serving.knative.dev/service=my-function -f

# Logs de todos os containers (incluindo Dapr)
kubectl logs -l serving.knative.dev/service=my-function \
  --all-containers=true

# Logs de um pod específico
kubectl logs <pod-name> -c user-container
```

### Debug Interativo

```bash
# Executar shell no pod da função
kubectl exec -it <pod-name> -c user-container -- /bin/sh

# Testar conectividade de dentro do pod
kubectl exec -it <pod-name> -c user-container -- \
  curl http://other-service.default.svc.cluster.local
```

## Validação de Configuração

### Validar Function CR

```bash
# Validar sintaxe YAML
kubectl apply --dry-run=client -f function.yaml

# Validar com servidor (inclui validação de schema)
kubectl apply --dry-run=server -f function.yaml
```

### Validar Secrets

```bash
# Verificar Secret Git
kubectl get secret github-auth -o yaml

# Verificar anotações
kubectl get secret github-auth -o jsonpath='{.metadata.annotations}'

# Verificar conteúdo (base64 decoded)
kubectl get secret github-auth -o jsonpath='{.data.password}' | base64 -d
```

### Validar ServiceAccount

```bash
# Ver ServiceAccount
kubectl get serviceaccount <function-name>-sa -o yaml

# Verificar secrets anexados
kubectl get serviceaccount <function-name>-sa \
  -o jsonpath='{.secrets[*].name}'

# Verificar imagePullSecrets
kubectl get serviceaccount <function-name>-sa \
  -o jsonpath='{.imagePullSecrets[*].name}'
```

## Recursos Adicionais

- [Especificação do CRD](function-crd.md) - Campos e configurações detalhadas
- [Referência do Operator](operator-reference.md) - Comportamento e integrações
- [Guia de Autenticação Git](../02-guias/autenticacao-git.md) - Configuração de autenticação
- [Configuração de Registry](../05-operacoes/configuracao-registry.md) - Setup de registries

## Obtendo Ajuda

Se você não conseguir resolver o problema:

1. **Colete informações**:
   ```bash
   # Salvar status da Function
   kubectl get function my-function -o yaml > function-status.yaml
   
   # Salvar logs do operator
   kubectl logs -n zenith-operator-system \
     deployment/zenith-operator-controller-manager \
     --tail=200 > operator-logs.txt
   
   # Salvar logs do PipelineRun
   kubectl logs <pipelinerun-name>-fetch-source-pod \
     --all-containers > build-logs.txt
   ```

2. **Abra uma issue**: https://github.com/LucasGois1/zenith-operator/issues

3. **Inclua**:
   - Descrição do problema
   - Versão do operator
   - Function CR (sem informações sensíveis)
   - Logs relevantes
   - Passos para reproduzir

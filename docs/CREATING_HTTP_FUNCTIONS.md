# Criando Funções HTTP Síncronas

Este guia mostra como criar funções que respondem a requisições HTTP síncronas usando o Zenith Operator.

## Visão Geral

Funções HTTP síncronas são ideais para:
- APIs REST
- Webhooks
- Microserviços
- Endpoints HTTP que retornam respostas imediatas

O Zenith Operator automaticamente:
1. Clona seu repositório Git
2. Constrói uma imagem de container usando Buildpacks
3. Faz deploy como um Knative Service
4. Expõe uma URL pública acessível via HTTP

## Pré-requisitos

- Cluster Kubernetes com Zenith Operator instalado
- Repositório Git com código da função
- Registry de container (ou usar registry local)
- Secret de autenticação Git (se repositório privado)

## Estrutura do Código da Função

Sua função deve ser uma aplicação HTTP que escuta em uma porta (geralmente 8080). O Zenith Operator usa Cloud Native Buildpacks para detectar automaticamente a linguagem e construir a imagem.

### Exemplo em Go

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
)

type Response struct {
    Status  string `json:"status"`
    Message string `json:"message"`
    Type    string `json:"type"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    response := Response{
        Status:  "ok",
        Message: "Hello from Zenith Function!",
        Type:    "http-sync",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    http.HandleFunc("/", handler)
    log.Printf("Listening on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### Exemplo em Python

```python
from flask import Flask, jsonify
import os

app = Flask(__name__)

@app.route('/')
def hello():
    return jsonify({
        'status': 'ok',
        'message': 'Hello from Zenith Function!',
        'type': 'http-sync'
    })

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8080))
    app.run(host='0.0.0.0', port=port)
```

### Exemplo em Node.js

```javascript
const express = require('express');
const app = express();

app.get('/', (req, res) => {
    res.json({
        status: 'ok',
        message: 'Hello from Zenith Function!',
        type: 'http-sync'
    });
});

const port = process.env.PORT || 8080;
app.listen(port, () => {
    console.log(`Listening on port ${port}`);
});
```

## Passo 1: Preparar o Repositório Git

1. Crie um repositório Git com o código da sua função
2. Certifique-se de que o código está na raiz do repositório
3. Faça commit e push para o GitHub/GitLab

```bash
git init
git add .
git commit -m "Initial function implementation"
git remote add origin https://github.com/myorg/my-function
git push -u origin main
```

## Passo 2: Criar Secret de Autenticação Git (Opcional)

Se seu repositório é privado, crie um Secret para autenticação:

```bash
# Criar secret com credenciais
kubectl create secret generic github-auth \
  --from-literal=username=myusername \
  --from-literal=password=ghp_mytoken \
  --type=kubernetes.io/basic-auth

# Adicionar anotação para Tekton
kubectl annotate secret github-auth \
  tekton.dev/git-0=https://github.com
```

**Nota**: Para GitHub, use um Personal Access Token (PAT) como password.

## Passo 3: Criar o Custom Resource Function

Crie um arquivo YAML com a definição da função:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-http-function
  namespace: default
spec:
  # Repositório Git com o código
  gitRepo: https://github.com/myorg/my-function
  gitRevision: main
  
  # Secret de autenticação (opcional)
  gitAuthSecretName: github-auth
  
  # Configuração de build
  build:
    # Imagem de destino
    image: registry.example.com/my-http-function:latest
    
    # Secret do registry (opcional)
    # registrySecretName: registry-credentials
  
  # Configuração de deploy
  deploy: {}
```

Aplique o recurso:

```bash
kubectl apply -f my-http-function.yaml
```

## Passo 4: Monitorar o Build

O operator criará automaticamente um PipelineRun do Tekton para construir a imagem:

```bash
# Ver o status da função
kubectl get functions

# Ver detalhes da função
kubectl describe function my-http-function

# Ver PipelineRuns
kubectl get pipelineruns

# Ver logs do build
kubectl logs -f <pipelinerun-name>-fetch-source-pod --all-containers
```

O status da função passará por estas fases:
1. **Building**: Build em progresso
2. **BuildSucceeded**: Build completado com sucesso
3. **Ready**: Função deployada e pronta para receber requisições

## Passo 5: Acessar a Função

Após o deploy, a função estará acessível via URL:

```bash
# Obter a URL da função
FUNCTION_URL=$(kubectl get function my-http-function -o jsonpath='{.status.url}')
echo "Function URL: $FUNCTION_URL"

# Fazer uma requisição
curl $FUNCTION_URL

# Resposta esperada:
# {"status":"ok","message":"Hello from Zenith Function!","type":"http-sync"}
```

### Acessar via Envoy Gateway

Se você estiver acessando de fora do cluster, use o Envoy Gateway:

```bash
# Obter o endpoint do Envoy Gateway
ENVOY_IP=$(kubectl get svc -n envoy-gateway-system \
  -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway \
  -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')

# Obter o hostname da função
FUNCTION_HOST=$(echo $FUNCTION_URL | sed 's|http://||' | sed 's|https://||')

# Fazer requisição com Host header
curl -H "Host: $FUNCTION_HOST" http://$ENVOY_IP/
```

## Passo 6: Atualizar a Função

Para atualizar a função, faça mudanças no código e push para o Git:

```bash
# Fazer mudanças no código
git add .
git commit -m "Update function"
git push

# Atualizar o Function CR para triggerar rebuild
kubectl annotate function my-http-function \
  rebuild=$(date +%s) --overwrite
```

Ou atualize a revisão Git no spec:

```yaml
spec:
  gitRevision: v2.0.0  # Nova tag ou branch
```

## Configurações Avançadas

### Variáveis de Ambiente

Adicione variáveis de ambiente para sua função:

```yaml
spec:
  deploy:
    env:
      - name: DATABASE_URL
        value: postgres://db.example.com/mydb
      - name: API_KEY
        value: secret-key
```

### Integração com Dapr

Habilite o sidecar Dapr para service mesh:

```yaml
spec:
  deploy:
    dapr:
      enabled: true
      appID: my-http-function
      appPort: 8080
```

Com Dapr habilitado, você pode usar:
- Service discovery
- Pub/Sub
- State management
- Secret stores

### Registry Privado

Para usar um registry privado, crie um Secret:

```bash
kubectl create secret docker-registry registry-credentials \
  --docker-server=registry.example.com \
  --docker-username=myuser \
  --docker-password=mypassword \
  --docker-email=myemail@example.com
```

E referencie no Function:

```yaml
spec:
  build:
    image: registry.example.com/my-http-function:latest
    registrySecretName: registry-credentials
```

## Troubleshooting

### Build Falha

Se o build falhar, verifique os logs:

```bash
# Ver PipelineRuns
kubectl get pipelineruns

# Ver logs do PipelineRun
kubectl describe pipelinerun <pipelinerun-name>

# Ver logs detalhados
kubectl logs <pipelinerun-name>-fetch-source-pod --all-containers
```

Problemas comuns:
- **Autenticação Git falhou**: Verifique o Secret e token
- **Buildpack não detectou a linguagem**: Certifique-se de ter arquivos como `go.mod`, `package.json`, `requirements.txt` na raiz
- **Registry push falhou**: Verifique credenciais do registry

### Função Não Responde

Se a função não responder:

```bash
# Ver status do Knative Service
kubectl get ksvc

# Ver pods da função
kubectl get pods -l serving.knative.dev/service=my-http-function

# Ver logs da função
kubectl logs -l serving.knative.dev/service=my-http-function
```

Problemas comuns:
- **Porta incorreta**: Certifique-se de escutar na porta especificada pela variável `PORT`
- **Aplicação não inicia**: Verifique logs do pod
- **Timeout**: Função demora muito para responder (scale-from-zero)

### URL Não Acessível

Se a URL não estiver acessível:

```bash
# Verificar Envoy Gateway
kubectl get svc -n envoy-gateway-system

# Verificar HTTPRoute
kubectl get httproute

# Verificar Gateway
kubectl get gateway -n knative-serving
```

## Exemplos Completos

Veja exemplos completos no repositório:
- [zenith-test-functions](https://github.com/LucasGois1/zenith-test-functions) - Função Go básica
- [config/samples/](../config/samples/) - Exemplos de Function CRs

## Próximos Passos

- [Criando Funções Assíncronas com Eventos](CREATING_EVENT_FUNCTIONS.md)
- [Comunicação entre Funções](INTER_FUNCTION_COMMUNICATION.md)
- [Referência Completa do Operator](OPERATOR_REFERENCE.md)

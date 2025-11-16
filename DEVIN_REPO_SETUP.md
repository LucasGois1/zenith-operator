# Comandos para Configuração do Repo Setup no Devin

Este documento contém todos os comandos e prompts que devem ser adicionados na configuração do repositório zenith-operator no Devin.

Acesse: **Settings > Devin's Workspace > zenith-operator > Edit > Set up in VSCode**

---

## 1. Git Pull

**Campo**: Git Pull  
**Comando**:
```bash
git pull --recurse-submodules
```

---

## 2. Configure Secrets

**Campo**: Configure Secrets

### Passo 1: Criar Secret GITHUB_TOKEN

1. Vá em **Settings > Secrets**
2. Clique em **Add Secret**
3. Nome: `GITHUB_TOKEN`
4. Valor: Seu Personal Access Token do GitHub com:
   - **Classic PAT**: scope `repo` (acesso completo a repositórios privados)
   - **Fine-grained PAT**: 
     - Permissions: `Contents: Read`
     - Repository access: Adicionar explicitamente `LucasGois1/zenith-test-functions`

### Passo 2: Configurar direnv (no terminal do VSCode)

```bash
# Instalar direnv (se não estiver instalado)
sudo apt install direnv -y

# Adicionar hook ao ~/.bashrc
echo 'eval "$(direnv hook bash)"' >> ~/.bashrc

# Criar .envrc no repositório
cat > ~/repos/zenith-operator/.envrc << 'EOF'
# Environment variables para zenith-operator
export CLUSTER_NAME="zenith-operator-test-e2e"
export IMG="zenith-operator:test"
export GITHUB_USERNAME="LucasGois1"
# GITHUB_TOKEN será injetado automaticamente pelo Devin via Secrets
EOF

# Permitir o .envrc
cd ~/repos/zenith-operator && direnv allow

# Adicionar .envrc ao .gitignore
echo ".envrc" >> ~/repos/zenith-operator/.gitignore
```

### Passo 3: Configurar auto-setup no ~/.bashrc

```bash
# Adicionar ao final do ~/.bashrc
cat >> ~/.bashrc << 'EOF'

# Auto-configuração para zenith-operator
function custom_cd() {
  builtin cd "$@"
  
  if [[ "$PWD" == "$HOME/repos/zenith-operator"* ]]; then
    # Configurar variáveis de ambiente
    export CLUSTER_NAME="${CLUSTER_NAME:-zenith-operator-test-e2e}"
    export IMG="${IMG:-zenith-operator:test}"
    export GITHUB_USERNAME="${GITHUB_USERNAME:-LucasGois1}"
    
    # Verificar se GITHUB_TOKEN está configurado
    if [ -z "$GITHUB_TOKEN" ]; then
      echo "⚠️  GITHUB_TOKEN não configurado. Configure em Settings > Secrets."
    fi
  fi
}

alias cd='custom_cd'
cd $PWD
EOF

# Recarregar bashrc
source ~/.bashrc
```

---

## 3. Install Dependencies

**Campo**: Install Dependencies

```bash
# Instalar Go (se não estiver instalado)
if ! command -v go &> /dev/null; then
  wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
  source ~/.bashrc
fi

# Instalar kind (se não estiver instalado)
if ! command -v kind &> /dev/null; then
  curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
  chmod +x ./kind
  sudo mv ./kind /usr/local/bin/kind
fi

# Instalar kubectl (se não estiver instalado)
if ! command -v kubectl &> /dev/null; then
  curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
  chmod +x kubectl
  sudo mv kubectl /usr/local/bin/
fi

# Instalar Chainsaw
curl -L https://github.com/kyverno/chainsaw/releases/latest/download/chainsaw_linux_amd64.tar.gz -o /tmp/chainsaw.tar.gz
tar -xzf /tmp/chainsaw.tar.gz -C /tmp
sudo mv /tmp/chainsaw /usr/local/bin/
rm /tmp/chainsaw.tar.gz

# Instalar jq (para scripts de verificação)
sudo apt-get update && sudo apt-get install -y jq

# Instalar dependências Go do projeto
cd ~/repos/zenith-operator
go mod download

# Gerar manifests e código
make manifests generate

# Setup completo do ambiente (cluster + operator)
make dev-up
```

---

## 4. Maintain Dependencies

**Campo**: Maintain Dependencies

```bash
cd ~/repos/zenith-operator && go mod tidy && make manifests generate
```

---

## 5. Setup Lint

**Campo**: Setup Lint

```bash
cd ~/repos/zenith-operator && make lint
```

**Nota**: Este comando pode falhar com warnings pré-existentes (complexidade ciclomática, unparam). Isso é esperado e não deve bloquear commits.

---

## 6. Setup Tests

**Campo**: Setup Tests

```bash
# Teste rápido de validação Git (~2 min)
cd ~/repos/zenith-operator && make test-chainsaw-git
```

**Nota**: Para testes completos (leva ~10 min), use `make test-chainsaw`. Para desenvolvimento, use testes individuais.

---

## 7. Setup Local App

**Campo**: Setup Local App

```bash
# Executar operator localmente (fora do cluster)
cd ~/repos/zenith-operator && make run
```

**Nota**: Para testar no cluster, use `make dev-redeploy` após fazer mudanças no código.

---

## 8. Additional Notes

**Campo**: Additional Notes

```markdown
## Informações Importantes

### Estrutura dos Repositórios
- **zenith-operator**: Operator principal (este repo)
- **zenith-test-functions**: Repositório PRIVADO com funções de teste
  - Branch de testes: `devin/1763232295-add-test-functions`
  - Arquivos Go devem estar na RAIZ do repositório

### Comandos Úteis

#### Desenvolvimento
```bash
make dev-up              # Setup completo (cluster + operator)
make dev-redeploy        # Rebuild e redeploy rápido
make docker-build        # Build da imagem do operator
make deploy              # Deploy do operator no cluster
```

#### Testes
```bash
make test-chainsaw              # Todos os testes (~10 min)
make test-chainsaw-git          # Teste git-clone (~2 min)
make test-chainsaw-basic        # Teste básico (~10 min)
make test-chainsaw-sa           # Teste ServiceAccount (~2 min)

# Teste com namespace preservado para debug
make test-chainsaw CHAINSAW_ARGS="--test-dir test/chainsaw/basic-function --skip-delete"
```

#### Debug
```bash
# Ver PipelineRuns e TaskRuns
kubectl get pipelineruns,taskruns -n <namespace>

# Ver logs do git-clone
kubectl logs -n <namespace> <pipelinerun>-fetch-source-pod --all-containers

# Ver ServiceAccount e secrets
kubectl get sa <function-name>-sa -o yaml -n <namespace>
kubectl get secret github-auth -o yaml -n <namespace>

# Verificar token (primeiros 12 caracteres)
kubectl get secret github-auth -n <namespace> -o jsonpath='{.data.password}' | base64 -d | head -c 12

# Verificar se token tem permissões corretas
bash hack/verify-github-token.sh
```

### Troubleshooting Comum

#### 1. Teste falha com 403 "Write access to repository not granted"
**Causa**: Token não tem permissões corretas

**Verificar**:
```bash
# Testar via API
curl -H "Authorization: token ${GITHUB_TOKEN}" https://api.github.com/repos/LucasGois1/zenith-test-functions

# Testar via git
git ls-remote https://LucasGois1:${GITHUB_TOKEN}@github.com/LucasGois1/zenith-test-functions
```

**Solução**:
- Classic PAT: Verificar se tem scope `repo` (não apenas `public_repo`)
- Fine-grained PAT: 
  1. GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens
  2. Clicar no token
  3. Em "Repository access", adicionar `LucasGois1/zenith-test-functions`
  4. Em "Permissions", configurar "Contents" como "Read-only"

#### 2. Build falha rapidamente (~5 segundos)
**Causa**: Provavelmente git-clone falhou

**Verificar**: Logs do TaskRun fetch-source
```bash
kubectl logs -n <namespace> <pipelinerun>-fetch-source-pod --all-containers
```

#### 3. Testes levam muito tempo
**Causa**: Testes completos esperam builds completarem (~10 min)

**Solução**: Para desenvolvimento, usar testes individuais:
```bash
make test-chainsaw-git    # ~2 min
make test-chainsaw-sa     # ~2 min
```

### Configuração de Autenticação Git

O operator suporta dois métodos de autenticação:

#### HTTPS (recomendado para testes)
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-auth
  annotations:
    tekton.dev/git-0: https://github.com
type: kubernetes.io/basic-auth
stringData:
  username: LucasGois1  # Username do GitHub (NÃO usar x-access-token)
  password: <GITHUB_TOKEN>
```

#### SSH
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-ssh
  annotations:
    tekton.dev/git-0: github.com
type: kubernetes.io/ssh-auth
stringData:
  ssh-privatekey: <SSH_PRIVATE_KEY>
```

### Arquitetura do Operator

O operator implementa autenticação Git usando o padrão Tekton:

1. **ServiceAccount por Function**: Cada Function recebe um ServiceAccount dedicado (`{function-name}-sa`)
2. **Secret Attachment**: Git secrets → `serviceAccount.secrets`, Registry secrets → `serviceAccount.imagePullSecrets`
3. **Tekton Credential Initializer**: Descobre automaticamente secrets via anotação `tekton.dev/git-0`
4. **OwnerReference**: ServiceAccounts são automaticamente deletados quando a Function é removida

### Documentação Adicional

- **Git Authentication**: `docs/GIT_AUTHENTICATION.md`
- **Chainsaw Tests**: `test/chainsaw/README.md`
- **CRD Reference**: `api/v1alpha1/function_types.go`

### Scripts Úteis

- `hack/dev-up.sh`: Setup completo do ambiente
- `hack/verify-github-token.sh`: Verificar permissões do token
```

---

## Resumo dos Arquivos Criados

Os seguintes arquivos foram criados no repositório e devem ser commitados:

1. `hack/dev-up.sh` - Script de setup completo do ambiente
2. `hack/verify-github-token.sh` - Script de verificação de token
3. Makefile atualizado com novos targets:
   - `make dev-up`
   - `make dev-redeploy`
   - `make test-chainsaw-git`
   - `make test-chainsaw-basic`
   - `make test-chainsaw-sa`

---

## Verificação Final

Após configurar tudo, execute no terminal do VSCode:

```bash
# 1. Verificar variáveis de ambiente
echo "CLUSTER_NAME: $CLUSTER_NAME"
echo "IMG: $IMG"
echo "GITHUB_USERNAME: $GITHUB_USERNAME"
echo "GITHUB_TOKEN: ${GITHUB_TOKEN:0:20}..."  # Primeiros 20 caracteres

# 2. Verificar token
bash hack/verify-github-token.sh

# 3. Setup completo
make dev-up

# 4. Executar teste rápido
make test-chainsaw-git
```

Se tudo funcionar, você verá:
- ✅ Cluster kind criado
- ✅ Tekton e Knative instalados
- ✅ Operator deployed
- ✅ Token verificado
- ✅ Teste git-clone passou (~2 min)

---

## Próximos Passos

Após configurar o repo setup:

1. **Salvar snapshot**: Clique em "Finish" para salvar o snapshot da VM
2. **Testar em nova sessão**: Inicie uma nova sessão e verifique se tudo funciona
3. **Documentar no README**: Adicionar instruções de desenvolvimento no README.md do projeto

---

## Suporte

Se encontrar problemas:
1. Verificar logs do operator: `kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager`
2. Verificar status do cluster: `kubectl get pods -A`
3. Executar verificação de token: `bash hack/verify-github-token.sh`
4. Consultar documentação: `docs/GIT_AUTHENTICATION.md`

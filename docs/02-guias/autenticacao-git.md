# Autenticação Git para Repositórios Privados

O Zenith Operator suporta autenticação com repositórios Git privados usando Secrets do Kubernetes. Este guia explica como configurar a autenticação para diferentes cenários.

## Visão Geral

Quando você especifica um repositório Git privado em um recurso `Function`, o operator precisa de credenciais para clonar o código-fonte. O Tekton Pipelines (usado internamente pelo operator) descobre automaticamente as credenciais através de Secrets anotados anexados ao ServiceAccount.

## Opção 1: HTTPS com GitHub Token (Recomendado)

Esta é a abordagem mais simples e funciona bem para a maioria dos casos de uso.

### Passo 1: Criar um GitHub Token

1. Acesse https://github.com/settings/tokens
2. Clique em **"Generate new token"** → **"Fine-grained tokens"** (recomendado)
3. Configure o token:
   - **Repository access**: Selecione os repositórios específicos que o operator precisa acessar
   - **Permissions**: 
     - **Contents**: Read-only
   - **Expiration**: Defina conforme suas políticas de segurança
4. Clique em **"Generate token"** e copie o token gerado

**Nota**: Para tokens clássicos (Classic PAT), use o escopo `repo` (read).

### Passo 2: Criar o Secret no Kubernetes

```bash
# Substitua SEU_TOKEN_AQUI pelo token gerado no Passo 1
kubectl create secret generic github-auth \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=x-access-token \
  --from-literal=password=SEU_TOKEN_AQUI \
  -n seu-namespace

# Anotar o Secret para o Tekton descobrir
kubectl annotate secret github-auth \
  'tekton.dev/git-0=https://github.com' \
  -n seu-namespace
```

**Importante**: A anotação `tekton.dev/git-0` deve corresponder ao prefixo da URL do repositório Git.

### Passo 3: Referenciar o Secret no Function CR

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: minha-funcao
  namespace: seu-namespace
spec:
  gitRepo: https://github.com/SuaOrg/repo-privado
  gitRevision: main
  gitAuthSecretName: github-auth  # ← Referência ao Secret criado
  build:
    image: registry.io/sua-imagem:latest
  deploy:
    dapr:
      enabled: false
      appID: ""
      appPort: 8080
```

## Opção 2: SSH com Deploy Key

Para ambientes corporativos ou quando SSH é preferido, você pode usar Deploy Keys do GitHub.

### Passo 1: Gerar um Par de Chaves SSH

```bash
# Gerar chave ED25519 (recomendado)
ssh-keygen -t ed25519 -f ./deploy-key -N ""

# Ou RSA se ED25519 não for suportado
ssh-keygen -t rsa -b 4096 -f ./deploy-key -N ""
```

Isso criará dois arquivos:
- `deploy-key` (chave privada)
- `deploy-key.pub` (chave pública)

### Passo 2: Adicionar a Chave Pública como Deploy Key no GitHub

1. Vá para o repositório no GitHub
2. Navegue até **Settings** → **Deploy keys**
3. Clique em **"Add deploy key"**
4. Cole o conteúdo de `deploy-key.pub`
5. Marque **"Allow read access"** (não marque write a menos que necessário)
6. Clique em **"Add key"**

### Passo 3: Criar o Secret no Kubernetes

```bash
# Obter known_hosts do GitHub
ssh-keyscan github.com > known_hosts

# Criar Secret com a chave privada
kubectl create secret generic github-ssh \
  --type=kubernetes.io/ssh-auth \
  --from-file=ssh-privatekey=./deploy-key \
  --from-file=known_hosts=./known_hosts \
  -n seu-namespace

# Anotar o Secret para o Tekton descobrir
kubectl annotate secret github-ssh \
  'tekton.dev/git-0=github.com' \
  -n seu-namespace

# Limpar arquivos de chave local (importante!)
rm -f ./deploy-key ./deploy-key.pub ./known_hosts
```

### Passo 4: Usar URL SSH no Function CR

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: minha-funcao
  namespace: seu-namespace
spec:
  gitRepo: git@github.com:SuaOrg/repo-privado.git  # ← Formato SSH
  gitRevision: main
  gitAuthSecretName: github-ssh  # ← Referência ao Secret SSH
  build:
    image: registry.io/sua-imagem:latest
  deploy:
    dapr:
      enabled: false
      appID: ""
      appPort: 8080
```

## Como Funciona

O Zenith Operator implementa autenticação Git da seguinte forma:

1. **ServiceAccount Dedicado**: Para cada Function, o operator cria um ServiceAccount dedicado (`<function-name>-sa`)

2. **Anexação de Secrets**: 
   - Secrets de Git são anexados a `serviceAccount.secrets`
   - Secrets de Registry são anexados a `serviceAccount.imagePullSecrets`

3. **Descoberta pelo Tekton**: O Tekton Pipelines usa um "credential initializer" que:
   - Examina todos os Secrets referenciados pelo ServiceAccount
   - Procura por anotações `tekton.dev/git-0` que correspondam à URL do repositório
   - Configura automaticamente as credenciais Git no container da Task

4. **Correspondência de URL**:
   - Para HTTPS: A anotação deve ser `https://github.com` (ou outro host)
   - Para SSH: A anotação deve ser `github.com` (sem protocolo)
   - O tipo do Secret deve corresponder ao protocolo (basic-auth para HTTPS, ssh-auth para SSH)

## Troubleshooting

### Erro: "fatal: could not read Username"

**Causa**: O Secret não está corretamente anotado ou não está anexado ao ServiceAccount.

**Solução**:
```bash
# Verificar se o Secret existe
kubectl get secret github-auth -n seu-namespace

# Verificar anotações
kubectl get secret github-auth -n seu-namespace -o jsonpath='{.metadata.annotations}'

# Verificar se o ServiceAccount foi criado
kubectl get serviceaccount <function-name>-sa -n seu-namespace

# Verificar se o Secret está anexado
kubectl get serviceaccount <function-name>-sa -n seu-namespace -o yaml
```

### Erro: "Host key verification failed"

**Causa**: O arquivo `known_hosts` não foi incluído no Secret SSH.

**Solução**:
```bash
# Recriar o Secret com known_hosts
ssh-keyscan github.com > known_hosts
kubectl delete secret github-ssh -n seu-namespace
kubectl create secret generic github-ssh \
  --type=kubernetes.io/ssh-auth \
  --from-file=ssh-privatekey=./deploy-key \
  --from-file=known_hosts=./known_hosts \
  -n seu-namespace
kubectl annotate secret github-ssh 'tekton.dev/git-0=github.com' -n seu-namespace
```

### Erro: "Permission denied (publickey)"

**Causa**: A chave SSH não foi adicionada como Deploy Key no GitHub ou o Secret contém a chave errada.

**Solução**:
1. Verifique se a chave pública está registrada no GitHub
2. Verifique se o Secret contém a chave privada correta:
   ```bash
   kubectl get secret github-ssh -n seu-namespace -o jsonpath='{.data.ssh-privatekey}' | base64 -d
   ```

### Status da Function mostra "GitAuthMissing"

**Causa**: O Secret especificado em `gitAuthSecretName` não existe no namespace.

**Solução**:
```bash
# Verificar se o Secret existe
kubectl get secret <secret-name> -n seu-namespace

# Se não existir, criar conforme os passos acima
```

## Boas Práticas de Segurança

1. **Use Fine-grained Tokens**: Prefira tokens fine-grained do GitHub com permissões mínimas necessárias

2. **Limite o Escopo**: Configure tokens para acessar apenas os repositórios específicos necessários

3. **Rotação de Credenciais**: Implemente rotação regular de tokens e chaves SSH

4. **Secrets por Namespace**: Crie Secrets separados para cada namespace/equipe

5. **Não Commite Credenciais**: Nunca commite tokens ou chaves privadas no Git

6. **Use RBAC**: Restrinja quem pode ler Secrets no cluster:
   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: secret-reader
     namespace: seu-namespace
   rules:
   - apiGroups: [""]
     resources: ["secrets"]
     verbs: ["get", "list"]
   ```

7. **Monitore Uso**: Configure alertas para uso anormal de credenciais

## Suporte para Outros Provedores Git

O mesmo padrão funciona para outros provedores Git:

### GitLab

```bash
# HTTPS
kubectl create secret generic gitlab-auth \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=oauth2 \
  --from-literal=password=SEU_TOKEN_GITLAB \
  -n seu-namespace

kubectl annotate secret gitlab-auth \
  'tekton.dev/git-0=https://gitlab.com' \
  -n seu-namespace
```

### Bitbucket

```bash
# HTTPS com App Password
kubectl create secret generic bitbucket-auth \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=SEU_USERNAME \
  --from-literal=password=SEU_APP_PASSWORD \
  -n seu-namespace

kubectl annotate secret bitbucket-auth \
  'tekton.dev/git-0=https://bitbucket.org' \
  -n seu-namespace
```

### Git Server Privado

```bash
# Substitua git.empresa.com pelo seu servidor
kubectl create secret generic git-privado-auth \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=SEU_USERNAME \
  --from-literal=password=SUA_SENHA \
  -n seu-namespace

kubectl annotate secret git-privado-auth \
  'tekton.dev/git-0=https://git.empresa.com' \
  -n seu-namespace
```

## Referências

- [Tekton Authentication Documentation](https://tekton.dev/docs/pipelines/auth/)
- [GitHub Fine-grained Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-fine-grained-personal-access-token)
- [GitHub Deploy Keys](https://docs.github.com/en/developers/overview/managing-deploy-keys#deploy-keys)
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)

# Git Authentication for Private Repositories

Zenith Operator supports authentication with private Git repositories using Kubernetes Secrets. This guide explains how to configure authentication for different scenarios.

## Overview

When you specify a private Git repository in a `Function` resource, the operator needs credentials to clone the source code. Tekton Pipelines (used internally by the operator) automatically discovers credentials through annotated Secrets attached to the ServiceAccount.

## Option 1: HTTPS with GitHub Token (Recommended)

This is the simplest approach and works well for most use cases.

### Step 1: Create a GitHub Token

1. Go to https://github.com/settings/tokens
2. Click **"Generate new token"** → **"Fine-grained tokens"** (recommended)
3. Configure the token:
   - **Repository access**: Select specific repositories the operator needs to access
   - **Permissions**: 
     - **Contents**: Read-only
   - **Expiration**: Set according to your security policies
4. Click **"Generate token"** and copy the generated token

**Note**: For classic tokens (Classic PAT), use the `repo` (read) scope.

### Step 2: Create Secret in Kubernetes

```bash
# Replace YOUR_TOKEN_HERE with the token generated in Step 1
kubectl create secret generic github-auth \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=x-access-token \
  --from-literal=password=YOUR_TOKEN_HERE \
  -n your-namespace

# Annotate the Secret for Tekton to discover
kubectl annotate secret github-auth \
  'tekton.dev/git-0=https://github.com' \
  -n your-namespace
```

**Important**: The `tekton.dev/git-0` annotation must match the Git repository URL prefix.

### Step 3: Reference Secret in Function CR

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-function
  namespace: your-namespace
spec:
  gitRepo: https://github.com/YourOrg/private-repo
  gitRevision: main
  gitAuthSecretName: github-auth  # ← Reference to created Secret
  build:
    image: registry.io/your-image:latest
  deploy:
    dapr:
      enabled: false
      appID: ""
      appPort: 8080
```

## Option 2: SSH with Deploy Key

For corporate environments or when SSH is preferred, you can use GitHub Deploy Keys.

### Step 1: Generate SSH Key Pair

```bash
# Generate ED25519 key (recommended)
ssh-keygen -t ed25519 -f ./deploy-key -N ""

# Or RSA if ED25519 is not supported
ssh-keygen -t rsa -b 4096 -f ./deploy-key -N ""
```

This will create two files:
- `deploy-key` (private key)
- `deploy-key.pub` (public key)

### Step 2: Add Public Key as Deploy Key in GitHub

1. Go to the repository on GitHub
2. Navigate to **Settings** → **Deploy keys**
3. Click **"Add deploy key"**
4. Paste the content of `deploy-key.pub`
5. Check **"Allow read access"** (do not check write unless necessary)
6. Click **"Add key"**

### Step 3: Create Secret in Kubernetes

```bash
# Get GitHub known_hosts
ssh-keyscan github.com > known_hosts

# Create Secret with private key
kubectl create secret generic github-ssh \
  --type=kubernetes.io/ssh-auth \
  --from-file=ssh-privatekey=./deploy-key \
  --from-file=known_hosts=./known_hosts \
  -n your-namespace

# Annotate the Secret for Tekton to discover
kubectl annotate secret github-ssh \
  'tekton.dev/git-0=github.com' \
  -n your-namespace

# Clean up local key files (important!)
rm -f ./deploy-key ./deploy-key.pub ./known_hosts
```

### Step 4: Use SSH URL in Function CR

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-function
  namespace: your-namespace
spec:
  gitRepo: git@github.com:YourOrg/private-repo.git  # ← SSH format
  gitRevision: main
  gitAuthSecretName: github-ssh  # ← Reference to SSH Secret
  build:
    image: registry.io/your-image:latest
  deploy:
    dapr:
      enabled: false
      appID: ""
      appPort: 8080
```

## How It Works

Zenith Operator implements Git authentication as follows:

1. **Dedicated ServiceAccount**: For each Function, the operator creates a dedicated ServiceAccount (`<function-name>-sa`)

2. **Secret Attachment**: 
   - Git Secrets are attached to `serviceAccount.secrets`
   - Registry Secrets are attached to `serviceAccount.imagePullSecrets`

3. **Tekton Discovery**: Tekton Pipelines uses a "credential initializer" that:
   - Examines all Secrets referenced by the ServiceAccount
   - Looks for `tekton.dev/git-0` annotations matching the repository URL
   - Automatically configures Git credentials in the Task container

4. **URL Matching**:
   - For HTTPS: The annotation must be `https://github.com` (or other host)
   - For SSH: The annotation must be `github.com` (without protocol)
   - Secret type must match protocol (basic-auth for HTTPS, ssh-auth for SSH)

## Troubleshooting

### Error: "fatal: could not read Username"

**Cause**: Secret is not correctly annotated or not attached to ServiceAccount.

**Solution**:
```bash
# Verify if Secret exists
kubectl get secret github-auth -n your-namespace

# Verify annotations
kubectl get secret github-auth -n your-namespace -o jsonpath='{.metadata.annotations}'

# Verify if ServiceAccount was created
kubectl get serviceaccount <function-name>-sa -n your-namespace

# Verify if Secret is attached
kubectl get serviceaccount <function-name>-sa -n your-namespace -o yaml
```

### Error: "Host key verification failed"

**Cause**: `known_hosts` file was not included in the SSH Secret.

**Solution**:
```bash
# Recreate Secret with known_hosts
ssh-keyscan github.com > known_hosts
kubectl delete secret github-ssh -n your-namespace
kubectl create secret generic github-ssh \
  --type=kubernetes.io/ssh-auth \
  --from-file=ssh-privatekey=./deploy-key \
  --from-file=known_hosts=./known_hosts \
  -n your-namespace
kubectl annotate secret github-ssh 'tekton.dev/git-0=github.com' -n your-namespace
```

### Error: "Permission denied (publickey)"

**Cause**: SSH key was not added as Deploy Key in GitHub or Secret contains wrong key.

**Solution**:
1. Verify public key is registered in GitHub
2. Verify Secret contains correct private key:
   ```bash
   kubectl get secret github-ssh -n your-namespace -o jsonpath='{.data.ssh-privatekey}' | base64 -d
   ```

### Function Status shows "GitAuthMissing"

**Cause**: Secret specified in `gitAuthSecretName` does not exist in namespace.

**Solution**:
```bash
# Verify if Secret exists
kubectl get secret <secret-name> -n your-namespace

# If not exists, create following steps above
```

## Security Best Practices

1. **Use Fine-grained Tokens**: Prefer GitHub fine-grained tokens with minimum required permissions

2. **Limit Scope**: Configure tokens to access only specific necessary repositories

3. **Credential Rotation**: Implement regular token and SSH key rotation

4. **Secrets per Namespace**: Create separate Secrets for each namespace/team

5. **Do Not Commit Credentials**: Never commit tokens or private keys to Git

6. **Use RBAC**: Restrict who can read Secrets in the cluster:
   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: secret-reader
     namespace: your-namespace
   rules:
   - apiGroups: [""]
     resources: ["secrets"]
     verbs: ["get", "list"]
   ```

7. **Monitor Usage**: Configure alerts for abnormal credential usage

## Support for Other Git Providers

The same pattern works for other Git providers:

### GitLab

```bash
# HTTPS
kubectl create secret generic gitlab-auth \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=oauth2 \
  --from-literal=password=YOUR_GITLAB_TOKEN \
  -n your-namespace

kubectl annotate secret gitlab-auth \
  'tekton.dev/git-0=https://gitlab.com' \
  -n your-namespace
```

### Bitbucket

```bash
# HTTPS with App Password
kubectl create secret generic bitbucket-auth \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=YOUR_USERNAME \
  --from-literal=password=YOUR_APP_PASSWORD \
  -n your-namespace

kubectl annotate secret bitbucket-auth \
  'tekton.dev/git-0=https://bitbucket.org' \
  -n your-namespace
```

### Private Git Server

```bash
# Replace git.company.com with your server
kubectl create secret generic private-git-auth \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=YOUR_USERNAME \
  --from-literal=password=YOUR_PASSWORD \
  -n your-namespace

kubectl annotate secret private-git-auth \
  'tekton.dev/git-0=https://git.company.com' \
  -n your-namespace
```

## References

- [Tekton Authentication Documentation](https://tekton.dev/docs/pipelines/auth/)
- [GitHub Fine-grained Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-fine-grained-personal-access-token)
- [GitHub Deploy Keys](https://docs.github.com/en/developers/overview/managing-deploy-keys#deploy-keys)
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)

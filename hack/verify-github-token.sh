#!/bin/bash

set -e

GITHUB_USERNAME="${GITHUB_USERNAME:-LucasGois1}"
REPO="zenith-test-functions"

if [ -z "$GITHUB_TOKEN" ]; then
  echo "❌ GITHUB_TOKEN não configurado!"
  echo ""
  echo "Configure o token em Settings > Secrets com:"
  echo "  - Classic PAT: scope 'repo'"
  echo "  - Fine-grained PAT: 'Contents: Read' + acesso explícito ao repo ${GITHUB_USERNAME}/${REPO}"
  exit 1
fi

echo "Verificando acesso ao repositório ${GITHUB_USERNAME}/${REPO}..."

if ! curl -s -H "Authorization: token ${GITHUB_TOKEN}" "https://api.github.com/repos/${GITHUB_USERNAME}/${REPO}" | jq -e '.id' >/dev/null 2>&1; then
  echo "❌ GITHUB_TOKEN não tem acesso ao repositório ${GITHUB_USERNAME}/${REPO} via API"
  echo ""
  echo "Verifique se o token tem:"
  echo "  - Classic PAT: scope 'repo' (não apenas 'public_repo')"
  echo "  - Fine-grained PAT: 'Contents: Read' + acesso explícito ao repo ${GITHUB_USERNAME}/${REPO}"
  echo ""
  echo "Para fine-grained tokens:"
  echo "  1. Vá em GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens"
  echo "  2. Clique no token"
  echo "  3. Em 'Repository access', adicione ${GITHUB_USERNAME}/${REPO}"
  echo "  4. Em 'Permissions', configure 'Contents' como 'Read-only'"
  exit 1
fi

if ! git ls-remote "https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com/${GITHUB_USERNAME}/${REPO}" >/dev/null 2>&1; then
  echo "❌ GITHUB_TOKEN não consegue clonar o repositório ${GITHUB_USERNAME}/${REPO}"
  echo ""
  echo "Erro: remote: Write access to repository not granted."
  echo ""
  echo "Verifique se o token tem:"
  echo "  - Classic PAT: scope 'repo' (não apenas 'public_repo')"
  echo "  - Fine-grained PAT: 'Contents: Read' + acesso explícito ao repo ${GITHUB_USERNAME}/${REPO}"
  exit 1
fi

echo "✅ GITHUB_TOKEN válido e com permissões corretas!"

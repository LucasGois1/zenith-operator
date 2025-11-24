#!/bin/bash


set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 Verificando ambiente de desenvolvimento..."
echo ""

MISSING_TOOLS=()

check_tool() {
    local tool=$1
    local install_hint=$2
    
    if command -v "$tool" &> /dev/null; then
        local version=$($tool --version 2>&1 | head -1)
        echo -e "${GREEN}✓${NC} $tool: $version"
    else
        echo -e "${RED}✗${NC} $tool: não encontrado"
        echo -e "  ${YELLOW}→${NC} $install_hint"
        MISSING_TOOLS+=("$tool")
    fi
}

check_tool "docker" "Install: https://docs.docker.com/get-docker/"
check_tool "kind" "Install: curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64 && chmod +x ./kind && sudo mv ./kind /usr/local/bin/"
check_tool "kubectl" "Install: curl -LO https://dl.k8s.io/release/\$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/"
check_tool "helm" "Install: curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"
check_tool "go" "Install: https://go.dev/doc/install"
check_tool "make" "Install: sudo apt-get install build-essential"
check_tool "openssl" "Install: sudo apt-get install openssl"

echo ""
echo "Ferramentas opcionais:"
check_tool "chainsaw" "Install: bash hack/install-chainsaw.sh"
check_tool "jq" "Install: sudo apt-get install jq"

echo ""
if [ ${#MISSING_TOOLS[@]} -eq 0 ]; then
    echo -e "${GREEN}✅ Todos os requisitos estão instalados!${NC}"
    exit 0
else
    echo -e "${RED}❌ Faltam ${#MISSING_TOOLS[@]} ferramenta(s): ${MISSING_TOOLS[*]}${NC}"
    echo ""
    echo "Execute os comandos de instalação acima ou rode:"
    echo "  bash hack/install-chainsaw.sh  # Para instalar Chainsaw"
    exit 1
fi

#!/bin/bash


set -e

CHAINSAW_VERSION="${CHAINSAW_VERSION:-0.2.13}"
INSTALL_DIR="${INSTALL_DIR:-./bin}"
OS="linux"
ARCH="amd64"

echo "📦 Instalando Chainsaw v${CHAINSAW_VERSION}..."

mkdir -p "$INSTALL_DIR"

DOWNLOAD_URL="https://github.com/kyverno/chainsaw/releases/download/v${CHAINSAW_VERSION}/chainsaw_${OS}_${ARCH}.tar.gz"
echo "⬇️  Baixando de: $DOWNLOAD_URL"

curl -sL "$DOWNLOAD_URL" | tar -xz -C "$INSTALL_DIR" chainsaw

chmod +x "$INSTALL_DIR/chainsaw"

if "$INSTALL_DIR/chainsaw" version &> /dev/null; then
    VERSION=$("$INSTALL_DIR/chainsaw" version 2>&1 | head -1)
    echo "✅ Chainsaw instalado com sucesso: $VERSION"
    echo ""
    echo "Para usar, adicione ao PATH:"
    echo "  export PATH=\"\$(pwd)/bin:\$PATH\""
    echo ""
    echo "Ou mova para /usr/local/bin:"
    echo "  sudo mv $INSTALL_DIR/chainsaw /usr/local/bin/"
else
    echo "❌ Erro ao instalar Chainsaw"
    exit 1
fi

#!/bin/bash
set -e

# Install Go if needed
if ! command -v go &>/dev/null; then
  echo "→ Installing Go..."
  GO_VERSION="1.23.4"
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
  rm "go${GO_VERSION}.linux-amd64.tar.gz"
  export PATH=$PATH:/usr/local/go/bin
  echo "✓ Go installed"
fi

# Install flyctl if needed
if ! command -v fly &>/dev/null; then
  echo "→ Installing flyctl..."
  curl -L https://fly.io/install.sh | sh
  export PATH=$PATH:$HOME/.fly/bin
  echo "✓ flyctl installed"
fi

echo "→ Generating go.sum..."
cd api && go mod tidy && cd ..

echo "→ Committing go.sum..."
git add api/go.sum && git commit -m "Add go.sum" 2>/dev/null || echo "  (already committed)"

echo "→ Deploying to Fly.io..."
fly deploy

echo "✓ Done. Visit https://playground-flywheel.fly.dev"

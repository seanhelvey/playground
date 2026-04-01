#!/bin/bash
set -e

# Install flyctl if needed
if ! command -v fly &>/dev/null; then
  echo "→ Installing flyctl..."
  curl -L https://fly.io/install.sh | sh
  export PATH=$PATH:$HOME/.fly/bin
fi

# Generate go.sum if needed
if [ ! -f api/go.sum ]; then
  echo "→ Generating go.sum (requires Go)..."
  cd api && go mod tidy && cd ..
  git add api/go.sum && git commit -m "Add go.sum"
fi

fly deploy

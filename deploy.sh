#!/bin/bash
set -e

echo "→ Generating go.sum..."
cd api && go mod tidy && cd ..

echo "→ Committing go.sum..."
git add api/go.sum && git commit -m "Add go.sum" 2>/dev/null || echo "  (already committed)"

echo "→ Deploying to Fly.io..."
fly deploy

echo "✓ Done. Visit https://playground-flywheel.fly.dev"

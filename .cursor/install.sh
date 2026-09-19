#!/usr/bin/env bash
# Cloud Agent install: idempotent, terminates. Prepares the Scenic Spot Guide
# System for the local SQLite demo path (no external services or secrets).
set -euo pipefail

# Run from the repository root regardless of where the script is invoked.
cd "$(dirname "$0")/.."

DEMO_PASSWORD="${SCENIC_GUIDE_DEMO_PASSWORD:-ScenicDemo123456}"

echo "==> [1/5] Downloading Go modules"
go mod download

echo "==> [2/5] Building Vue frontend into static/vue-app"
(
  cd web-vue
  npm ci
  npm run build
)

echo "==> [3/5] Ensuring local SQLite config (configs/config.yaml)"
if [ ! -f configs/config.yaml ]; then
  cp configs/config.example.yaml configs/config.yaml
fi

echo "==> [4/5] Compiling backend binary (validates build, warms cache)"
go build -o bin/scenic-guide .

echo "==> [5/5] Seeding demo data into SQLite (idempotent)"
# demo-seed loads configs/config.yaml, which requires a JWT secret to be present.
# It is only used to satisfy config validation here; the running server generates
# its own fresh secret at boot.
export SCENIC_GUIDE_SECURITY_JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/demo-seed --admin-password "${DEMO_PASSWORD}"

echo "==> Install complete"

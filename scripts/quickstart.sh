#!/usr/bin/env bash
# PhishForge quickstart — one-command setup.
# Creates .env with strong random secrets (if missing), then builds & starts the
# full stack with docker compose. Idempotent: safe to re-run.
set -euo pipefail

cd "$(dirname "$0")/.."

# --- checks ---
if ! command -v docker >/dev/null 2>&1; then
  echo "✗ docker not found. Install Docker first: https://docs.docker.com/get-docker/" >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "✗ 'docker compose' plugin not found. Install Docker Compose v2." >&2
  exit 1
fi

gen() { openssl rand -hex 32 2>/dev/null || head -c32 /dev/urandom | xxd -p | tr -d '\n'; }

# --- .env ---
if [ ! -f .env ]; then
  echo "→ Creating .env with random secrets..."
  cp .env.example .env
  JWT="$(gen)"; RID="$(gen)"
  ADMIN_EMAIL="${ADMIN_EMAIL:-admin@phishforge.local}"
  ADMIN_PASS="${ADMIN_PASS:-$(gen | cut -c1-16)}"
  sed -i.bak "s|^JWT_SECRET=.*|JWT_SECRET=${JWT}|"                       .env
  sed -i.bak "s|^RID_SECRET=.*|RID_SECRET=${RID}|"                       .env
  sed -i.bak "s|^BOOTSTRAP_ADMIN_EMAIL=.*|BOOTSTRAP_ADMIN_EMAIL=${ADMIN_EMAIL}|" .env
  sed -i.bak "s|^BOOTSTRAP_ADMIN_PASSWORD=.*|BOOTSTRAP_ADMIN_PASSWORD=${ADMIN_PASS}|" .env
  rm -f .env.bak
  echo "✓ .env created."
  echo
  echo "  ┌─────────────────────────────────────────────"
  echo "  │  Admin login (save these!):"
  echo "  │    email:    ${ADMIN_EMAIL}"
  echo "  │    password: ${ADMIN_PASS}"
  echo "  └─────────────────────────────────────────────"
  echo
else
  echo "→ .env already exists, keeping it. (Login is whatever BOOTSTRAP_ADMIN_* holds.)"
fi

# --- launch ---
echo "→ Building and starting the stack (this may take a few minutes the first time)..."
docker compose up -d --build

echo
echo "✓ PhishForge is starting."
echo "  Admin UI:   http://localhost:${ADMIN_PORT:-8080}"
echo "  Tracking:   http://localhost:${PHISH_PORT:-8081}"
echo "  Health:     http://localhost:${ADMIN_PORT:-8080}/healthz"
echo
echo "  Logs:  docker compose logs -f api worker"
echo "  Stop:  docker compose down     (add -v to also wipe data)"

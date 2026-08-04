#!/bin/bash

# AIOF Orchestrator Server Environment

[ -f .env ] && set -a && source .env && set +a

mkdir -p bin
cd apps/daemon && go build -o ../../bin/aiof ./cmd/aiof/main.go && cd ../..

export DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/aiof?sslmode=disable}"

echo "Starting AIOF Orchestrator..."
env GEMINI_API_KEY="$GEMINI_API_KEY" DATABASE_URL="$DATABASE_URL" ./bin/aiof "$@"

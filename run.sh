#!/bin/bash

# AIOF Orchestrator Server Environment

[ -f .env ] && set -a && source .env && set +a

if [ ! -f ./aiof_server ] || [ "$1" == "--build" ]; then
    go build -o aiof_server cmd/aiof/*.go
fi

export DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/aiof?sslmode=disable}"

echo "Starting AIOF Orchestrator..."
env GEMINI_API_KEY="$GEMINI_API_KEY" DATABASE_URL="$DATABASE_URL" ./aiof_server start

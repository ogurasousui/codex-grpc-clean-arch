#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$PROJECT_ROOT"

CONFIG_PATH=${CONFIG_PATH:-"$PROJECT_ROOT/assets/ci.yaml"}
COMPOSE_FILE=${COMPOSE_FILE:-"$PROJECT_ROOT/docker-compose.yaml"}
COMPOSE_PROFILE=${COMPOSE_PROFILE:-ci}
SERVICE_NAME=${SERVICE_NAME:-postgres}

cleanup() {
  local status=$?
  if [ $status -ne 0 ]; then
    docker compose -f "$COMPOSE_FILE" --profile "$COMPOSE_PROFILE" logs "$SERVICE_NAME" || true
  fi
  docker compose -f "$COMPOSE_FILE" --profile "$COMPOSE_PROFILE" down --volumes --remove-orphans || true
  exit $status
}
trap cleanup EXIT

docker compose -f "$COMPOSE_FILE" --profile "$COMPOSE_PROFILE" up -d "$SERVICE_NAME"

echo "Waiting for $SERVICE_NAME to become ready..."
ready=0
for attempt in {1..30}; do
  if docker compose -f "$COMPOSE_FILE" --profile "$COMPOSE_PROFILE" exec -T "$SERVICE_NAME" pg_isready -U app_user -d app_db >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done

if [ $ready -ne 1 ]; then
  echo "Postgres did not become ready in time" >&2
  exit 1
fi

go run ./cmd/migrate -config "$CONFIG_PATH" up
go test -tags integration ./test/...

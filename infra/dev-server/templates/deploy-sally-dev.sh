#!/usr/bin/env bash

set -euo pipefail

: "${DEPLOY_ROOT:=/opt/sally}"
: "${BRANCH:=main}"
: "${COMPOSE_FILE:=infra/dev-server/templates/docker-compose.sally-dev.yml}"

cd "${DEPLOY_ROOT}"

git fetch origin "${BRANCH}"
git checkout "${BRANCH}"
if git ls-tree -r --name-only "origin/${BRANCH}" -- infra/dev-server/templates/docker-compose.sally-dev.yml | grep -q .; then
  git clean -fd -- infra/dev-server .github/workflows/deploy-sally-dev.yml
fi
git reset --hard "origin/${BRANCH}"

if [[ ! -f .env ]]; then
  echo "${DEPLOY_ROOT}/.env is required" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1091
. ./.env
set +a

docker compose --env-file .env -f "${COMPOSE_FILE}" up -d --build --remove-orphans
docker compose --env-file .env -f "${COMPOSE_FILE}" ps

health_url="http://127.0.0.1:${PORT:-8080}/healthz"
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "${health_url}" | grep -qx "ok"; then
    echo "Sally dev server is healthy at ${health_url}"
    exit 0
  fi
  sleep 3
done

echo "Sally dev server failed health check at ${health_url}" >&2
docker compose --env-file .env -f "${COMPOSE_FILE}" logs --tail=100 sally-server >&2
exit 1

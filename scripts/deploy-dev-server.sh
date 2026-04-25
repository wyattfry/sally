#!/usr/bin/env bash

set -euo pipefail

: "${DEPLOY_ROOT:=$HOME/.local/share/sally-dev}"
: "${PORT:=40123}"
: "${OPENAI_MODEL:=gpt-5-mini}"

if [[ -z "${OPENAI_API_KEY:-}" ]]; then
  echo "OPENAI_API_KEY is required" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVER_DIR="${REPO_ROOT}/server"

BIN_DIR="${DEPLOY_ROOT}/bin"
RUN_DIR="${DEPLOY_ROOT}/run"
LOG_DIR="${DEPLOY_ROOT}/log"
ENV_FILE="${DEPLOY_ROOT}/server.env"
PID_FILE="${RUN_DIR}/sally-server.pid"
LOG_FILE="${LOG_DIR}/sally-server.log"

mkdir -p "${BIN_DIR}" "${RUN_DIR}" "${LOG_DIR}"

pushd "${SERVER_DIR}" >/dev/null
go build -o "${BIN_DIR}/sally-server" ./cmd/sally-server
popd >/dev/null

cat >"${ENV_FILE}" <<EOF
PORT=${PORT}
OPENAI_API_KEY=${OPENAI_API_KEY}
OPENAI_MODEL=${OPENAI_MODEL}
SALLY_ALLOW_MOCK_FALLBACK=false
EOF

if [[ -f "${PID_FILE}" ]]; then
  EXISTING_PID="$(cat "${PID_FILE}")"
  if kill -0 "${EXISTING_PID}" 2>/dev/null; then
    kill "${EXISTING_PID}"
    for _ in {1..20}; do
      if ! kill -0 "${EXISTING_PID}" 2>/dev/null; then
        break
      fi
      sleep 0.5
    done
  fi
  rm -f "${PID_FILE}"
fi

set -a
source "${ENV_FILE}"
set +a
nohup "${BIN_DIR}/sally-server" >"${LOG_FILE}" 2>&1 &
SERVER_PID=$!
echo "${SERVER_PID}" >"${PID_FILE}"

for _ in {1..20}; do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null; then
    echo "sally-server deployed on port ${PORT}"
    exit 0
  fi
  sleep 0.5
done

echo "sally-server failed health check; see ${LOG_FILE}" >&2
exit 1

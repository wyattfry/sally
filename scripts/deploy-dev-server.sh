#!/usr/bin/env bash

set -euo pipefail

: "${DEPLOY_ROOT:=$HOME/.local/share/sally-dev}"
: "${PORT:=8080}"
: "${LLM_PROVIDER:=stub}"
: "${OPENAI_MODEL:=gpt-5-mini}"

case "${LLM_PROVIDER}" in
  stub)
    ;;
  openai)
    if [[ -z "${OPENAI_API_KEY:-}" ]]; then
      echo "OPENAI_API_KEY is required when LLM_PROVIDER=openai" >&2
      exit 1
    fi
    ;;
  ollama)
    if [[ -z "${OLLAMA_BASE_URL:-}" || -z "${OLLAMA_MODEL:-}" ]]; then
      echo "OLLAMA_BASE_URL and OLLAMA_MODEL are required when LLM_PROVIDER=ollama" >&2
      exit 1
    fi
    ;;
  *)
    echo "Unsupported LLM_PROVIDER: ${LLM_PROVIDER}" >&2
    exit 1
    ;;
esac

if [[ -z "${OPENAI_API_KEY:-}" ]]; then
  OPENAI_API_KEY=""
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
LLM_PROVIDER=${LLM_PROVIDER}
OPENAI_API_KEY=${OPENAI_API_KEY}
OPENAI_MODEL=${OPENAI_MODEL}
OLLAMA_BASE_URL=${OLLAMA_BASE_URL:-}
OLLAMA_MODEL=${OLLAMA_MODEL:-}
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

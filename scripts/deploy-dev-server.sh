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
SYSTEMD_USER_DIR="${HOME}/.config/systemd/user"
SERVICE_NAME="sally-server.service"
SERVICE_FILE="${SYSTEMD_USER_DIR}/${SERVICE_NAME}"

BIN_DIR="${DEPLOY_ROOT}/bin"
TOOLCHAIN_DIR="${DEPLOY_ROOT}/toolchain"
ENV_FILE="${DEPLOY_ROOT}/server.env"

mkdir -p "${BIN_DIR}" "${TOOLCHAIN_DIR}" "${SYSTEMD_USER_DIR}"

if [[ -z "${XDG_RUNTIME_DIR:-}" ]]; then
  export XDG_RUNTIME_DIR="/run/user/$(id -u)"
fi
if [[ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ]]; then
  export DBUS_SESSION_BUS_ADDRESS="unix:path=${XDG_RUNTIME_DIR}/bus"
fi

required_go_version="$(awk '/^go / {print $2}' "${SERVER_DIR}/go.mod")"
go_bin=""

normalize_go_version() {
  local value="$1"
  value="${value#go}"
  printf '%s' "${value}"
}

bootstrap_go_toolchain() {
  local version="$1"
  local arch
  arch="$(uname -m)"
  case "${arch}" in
    x86_64) arch="amd64" ;;
    aarch64) arch="arm64" ;;
    *)
      echo "Unsupported architecture for Go bootstrap: ${arch}" >&2
      exit 1
      ;;
  esac

  local os
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  local archive="go${version}.${os}-${arch}.tar.gz"
  local url="https://go.dev/dl/${archive}"
  local download_path="${TOOLCHAIN_DIR}/${archive}"
  local install_root="${TOOLCHAIN_DIR}/go${version}"

  if [[ ! -x "${install_root}/go/bin/go" ]]; then
    curl -fsSL "${url}" -o "${download_path}"
    rm -rf "${install_root}"
    mkdir -p "${install_root}"
    tar -C "${install_root}" -xzf "${download_path}"
  fi

  go_bin="${install_root}/go/bin/go"
}

if command -v go >/dev/null 2>&1; then
  current_go_version="$(normalize_go_version "$(go version | awk '{print $3}')")"
  if [[ "${current_go_version}" == "${required_go_version}" ]]; then
    go_bin="$(command -v go)"
  fi
fi

if [[ -z "${go_bin}" ]]; then
  bootstrap_go_toolchain "${required_go_version}"
fi

pushd "${SERVER_DIR}" >/dev/null
"${go_bin}" build -o "${BIN_DIR}/sally-server" ./cmd/sally-server
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

cat >"${SERVICE_FILE}" <<EOF
[Unit]
Description=Sally backend server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=${ENV_FILE}
ExecStart=${BIN_DIR}/sally-server
Restart=always
RestartSec=2
WorkingDirectory=${DEPLOY_ROOT}

[Install]
WantedBy=default.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now "${SERVICE_NAME}"

for _ in {1..20}; do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null; then
    echo "sally-server deployed on port ${PORT}"
    linger_state="$(loginctl show-user "$(id -un)" -p Linger 2>/dev/null | cut -d= -f2 || true)"
    if [[ "${linger_state}" != "yes" ]]; then
      echo "warning: loginctl lingering is not enabled for $(id -un); service may not start on host boot until root runs: sudo loginctl enable-linger $(id -un)" >&2
    fi
    exit 0
  fi
  sleep 0.5
done

echo "sally-server failed health check; inspect with: systemctl --user status ${SERVICE_NAME} --no-pager" >&2
exit 1

#!/usr/bin/env bash

set -euo pipefail

: "${GUEST_HOST:?Set GUEST_HOST, for example wyatt@172.16.0.148}"
: "${GITHUB_RUNNER_TOKEN:?Set GITHUB_RUNNER_TOKEN from the GitHub Actions runner registration API}"
: "${REPO_URL:=https://github.com/wyattfry/sally}"
: "${RUNNER_USER:=wyatt}"
: "${RUNNER_ROOT:=/opt/actions-runner}"
: "${RUNNER_NAME:=sally-dev-vm}"
: "${RUNNER_LABELS:=sally-dev}"
: "${REMOTE_SUDO:=}"

if [[ -z "${REMOTE_SUDO}" && "${GUEST_HOST}" != root@* ]]; then
  REMOTE_SUDO="sudo"
fi

remote_env="GITHUB_RUNNER_TOKEN='${GITHUB_RUNNER_TOKEN}' REPO_URL='${REPO_URL}' RUNNER_USER='${RUNNER_USER}' RUNNER_ROOT='${RUNNER_ROOT}' RUNNER_NAME='${RUNNER_NAME}' RUNNER_LABELS='${RUNNER_LABELS}'"
if [[ -n "${REMOTE_SUDO}" ]]; then
  remote_command="${remote_env} ${REMOTE_SUDO} --preserve-env=GITHUB_RUNNER_TOKEN,REPO_URL,RUNNER_USER,RUNNER_ROOT,RUNNER_NAME,RUNNER_LABELS bash -s"
else
  remote_command="${remote_env} bash -s"
fi

ssh "${GUEST_HOST}" "${remote_command}" <<'EOF'
set -euo pipefail

apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl jq tar gzip sudo

if ! id -u "${RUNNER_USER}" >/dev/null 2>&1; then
  useradd -m -s /bin/bash "${RUNNER_USER}"
fi

install -d -o "${RUNNER_USER}" -g "${RUNNER_USER}" "${RUNNER_ROOT}"

arch="$(uname -m)"
case "${arch}" in
  x86_64) runner_arch="x64" ;;
  aarch64 | arm64) runner_arch="arm64" ;;
  *) echo "Unsupported runner architecture: ${arch}" >&2; exit 1 ;;
esac

latest_json="$(curl -fsSL https://api.github.com/repos/actions/runner/releases/latest)"
version="$(printf '%s' "${latest_json}" | jq -r '.tag_name | sub("^v"; "")')"
download_url="https://github.com/actions/runner/releases/download/v${version}/actions-runner-linux-${runner_arch}-${version}.tar.gz"

if [[ ! -x "${RUNNER_ROOT}/run.sh" || ! -f "${RUNNER_ROOT}/.runner-version" || "$(cat "${RUNNER_ROOT}/.runner-version")" != "${version}" ]]; then
  systemctl stop "actions.runner.$(printf '%s' "${REPO_URL}" | sed 's#https://github.com/##; s#/#-#g').${RUNNER_NAME}.service" >/dev/null 2>&1 || true
  rm -rf "${RUNNER_ROOT:?}/"*
  curl -fsSL "${download_url}" -o /tmp/actions-runner.tar.gz
  tar -xzf /tmp/actions-runner.tar.gz -C "${RUNNER_ROOT}"
  chown -R "${RUNNER_USER}:${RUNNER_USER}" "${RUNNER_ROOT}"
  printf '%s\n' "${version}" > "${RUNNER_ROOT}/.runner-version"
fi

if [[ ! -f "${RUNNER_ROOT}/.runner" ]]; then
  sudo -u "${RUNNER_USER}" bash -lc "cd '${RUNNER_ROOT}' && ./config.sh --url '${REPO_URL}' --token '${GITHUB_RUNNER_TOKEN}' --name '${RUNNER_NAME}' --labels '${RUNNER_LABELS}' --unattended --replace"
fi

cd "${RUNNER_ROOT}"
./svc.sh install "${RUNNER_USER}"
./svc.sh start
EOF

echo "Installed GitHub Actions runner ${RUNNER_NAME} on ${GUEST_HOST}"

#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_PATH="${ROOT_DIR}/scripts/deploy-cloudflared.sh"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

TEST_HOME="${TMP_DIR}/home"
BIN_DIR="${TMP_DIR}/bin"
LOG_DIR="${TMP_DIR}/log"
DEPLOY_ROOT="${TMP_DIR}/deploy"
mkdir -p "${TEST_HOME}/.cloudflared" "${BIN_DIR}" "${LOG_DIR}" "${DEPLOY_ROOT}"

CERT_JSON='{"zoneID":"zone-123","accountID":"account-123","apiToken":"api-token-123"}'
printf '%s' "${CERT_JSON}" | base64 -w0 >"${TEST_HOME}/.cloudflared/cert.pem"

cat >"${BIN_DIR}/cloudflared" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'cloudflared %s\n' "$*" >>"${TEST_LOG_DIR}/commands.log"
if [[ "${1:-}" == "tunnel" && "${2:-}" == "list" ]]; then
  printf '[{"id":"tunnel-123","name":"sally-dev"}]\n'
  exit 0
fi
exit 0
EOF

cat >"${BIN_DIR}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'curl %s\n' "$*" >>"${TEST_LOG_DIR}/commands.log"
for arg in "$@"; do
  printf '%s\n' "$arg" >>"${TEST_LOG_DIR}/curl.args"
done
printf '%s' "${*}" >"${TEST_LOG_DIR}/curl.command"
printf '{"success":true}\n'
EOF

cat >"${BIN_DIR}/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'systemctl %s\n' "$*" >>"${TEST_LOG_DIR}/commands.log"
exit 0
EOF

cat >"${BIN_DIR}/loginctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'Linger=yes\n'
EOF

chmod +x "${BIN_DIR}/cloudflared" "${BIN_DIR}/curl" "${BIN_DIR}/systemctl" "${BIN_DIR}/loginctl"

(
  export HOME="${TEST_HOME}"
  export PATH="${BIN_DIR}:${PATH}"
  export TEST_LOG_DIR="${LOG_DIR}"
  export DEPLOY_ROOT
  export CLOUDFLARED_TUNNEL_TOKEN="test-token"
  export CLOUDFLARED_TUNNEL_NAME="sally-dev"
  export CLOUDFLARED_PUBLIC_URL="https://dev.spexxtool.com"
  export CLOUDFLARED_URL="http://127.0.0.1:8080"
  export XDG_RUNTIME_DIR="${TMP_DIR}/run"
  mkdir -p "${XDG_RUNTIME_DIR}"
  bash "${SCRIPT_PATH}"
)

SERVICE_FILE="${TEST_HOME}/.config/systemd/user/sally-cloudflared.service"

grep -F 'cloudflared tunnel route dns sally-dev dev.spexxtool.com' "${LOG_DIR}/commands.log" >/dev/null
grep -F 'curl -fsS -X PUT' "${LOG_DIR}/commands.log" >/dev/null
grep -F '/accounts/account-123/cfd_tunnel/tunnel-123/configurations' "${LOG_DIR}/curl.command" >/dev/null
grep -F '"hostname": "dev.spexxtool.com"' "${LOG_DIR}/curl.args" >/dev/null
grep -F '"service": "http://127.0.0.1:8080"' "${LOG_DIR}/curl.args" >/dev/null
grep -F 'ExecStart=' "${SERVICE_FILE}" | grep -F 'cloudflared tunnel --no-autoupdate run --token test-token' >/dev/null

echo "deploy-cloudflared test passed"

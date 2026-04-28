#!/usr/bin/env bash

set -euo pipefail

: "${DEPLOY_ROOT:=$HOME/.local/share/sally-dev}"
: "${CLOUDFLARED_URL:=http://127.0.0.1:8080}"
: "${CLOUDFLARED_SERVICE_NAME:=sally-cloudflared.service}"
: "${CLOUDFLARED_QUICK_TUNNEL:=false}"

SYSTEMD_USER_DIR="${HOME}/.config/systemd/user"
SERVICE_FILE="${SYSTEMD_USER_DIR}/${CLOUDFLARED_SERVICE_NAME}"
ENV_FILE="${DEPLOY_ROOT}/cloudflared.env"

mkdir -p "${SYSTEMD_USER_DIR}" "${DEPLOY_ROOT}"

if [[ -z "${XDG_RUNTIME_DIR:-}" ]]; then
  export XDG_RUNTIME_DIR="/run/user/$(id -u)"
fi
if [[ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ]]; then
  export DBUS_SESSION_BUS_ADDRESS="unix:path=${XDG_RUNTIME_DIR}/bus"
fi

if ! command -v cloudflared >/dev/null 2>&1; then
  echo "cloudflared is not installed" >&2
  exit 1
fi

exec_start=()

case "${CLOUDFLARED_QUICK_TUNNEL}" in
  true)
    exec_start=( "$(command -v cloudflared)" tunnel --no-autoupdate --url "${CLOUDFLARED_URL}" )
    ;;
  false)
    if [[ -z "${CLOUDFLARED_TUNNEL_TOKEN:-}" ]]; then
      echo "CLOUDFLARED_TUNNEL_TOKEN is required unless CLOUDFLARED_QUICK_TUNNEL=true" >&2
      exit 1
    fi
    exec_start=( "$(command -v cloudflared)" tunnel --no-autoupdate run --token "${CLOUDFLARED_TUNNEL_TOKEN}" )
    ;;
  *)
    echo "CLOUDFLARED_QUICK_TUNNEL must be true or false" >&2
    exit 1
    ;;
esac

cat >"${ENV_FILE}" <<EOF
CLOUDFLARED_URL=${CLOUDFLARED_URL}
CLOUDFLARED_QUICK_TUNNEL=${CLOUDFLARED_QUICK_TUNNEL}
EOF

cat >"${SERVICE_FILE}" <<EOF
[Unit]
Description=Sally Cloudflare Tunnel
After=network-online.target sally-server.service
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=${ENV_FILE}
ExecStart=${exec_start[*]}
Restart=always
RestartSec=2
WorkingDirectory=${DEPLOY_ROOT}

[Install]
WantedBy=default.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now "${CLOUDFLARED_SERVICE_NAME}"

sleep 3
systemctl --user --no-pager --full status "${CLOUDFLARED_SERVICE_NAME}" || true

linger_state="$(loginctl show-user "$(id -un)" -p Linger 2>/dev/null | cut -d= -f2 || true)"
if [[ "${linger_state}" != "yes" ]]; then
  echo "warning: loginctl lingering is not enabled for $(id -un); service may not start on host boot until root runs: sudo loginctl enable-linger $(id -un)" >&2
fi

if [[ "${CLOUDFLARED_QUICK_TUNNEL}" == "true" ]]; then
  echo "quick tunnel started; inspect URL with: journalctl --user -u ${CLOUDFLARED_SERVICE_NAME} -n 50 --no-pager"
else
  echo "named tunnel started; verify DNS hostname through Cloudflare"
fi

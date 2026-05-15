#!/usr/bin/env bash

set -euo pipefail

: "${DEPLOY_ROOT:=$HOME/.local/share/sally-dev}"
: "${CLOUDFLARED_URL:=http://127.0.0.1:8080}"
: "${CLOUDFLARED_SERVICE_NAME:=sally-cloudflared.service}"
: "${CLOUDFLARED_QUICK_TUNNEL:=false}"
: "${CLOUDFLARED_TUNNEL_NAME:=sally-dev}"
: "${CLOUDFLARED_PUBLIC_URL:=}"

SYSTEMD_USER_DIR="${HOME}/.config/systemd/user"
SERVICE_FILE="${SYSTEMD_USER_DIR}/${CLOUDFLARED_SERVICE_NAME}"
ENV_FILE="${DEPLOY_ROOT}/cloudflared.env"
CERT_FILE="${HOME}/.cloudflared/cert.pem"

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

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required" >&2
  exit 1
fi

public_hostname=""
if [[ -n "${CLOUDFLARED_PUBLIC_URL}" ]]; then
  public_hostname="$(
    python3 - "${CLOUDFLARED_PUBLIC_URL}" <<'PY'
import sys
from urllib.parse import urlparse

url = urlparse(sys.argv[1])
if not url.scheme or not url.hostname:
    raise SystemExit(1)
print(url.hostname)
PY
  )"
fi

# When an origin cert and the named tunnel are present locally, derive the token from
# `cloudflared tunnel token <name>` — that's the source of truth. Doing this even when
# CLOUDFLARED_TUNNEL_TOKEN is already set in the environment protects against stale
# shells / .bashrc exports left over from a different tunnel.
if [[ "${CLOUDFLARED_QUICK_TUNNEL}" != "true" ]]; then
  if [[ -f "${CERT_FILE}" ]] && cloudflared tunnel list 2>/dev/null | awk 'NR>1 {print $2}' | grep -Fxq "${CLOUDFLARED_TUNNEL_NAME}"; then
    derived_token="$(cloudflared tunnel token "${CLOUDFLARED_TUNNEL_NAME}")"
    if [[ -n "${CLOUDFLARED_TUNNEL_TOKEN:-}" && "${CLOUDFLARED_TUNNEL_TOKEN}" != "${derived_token}" ]]; then
      echo "ignoring ambient CLOUDFLARED_TUNNEL_TOKEN (does not match local '${CLOUDFLARED_TUNNEL_NAME}' tunnel)" >&2
    fi
    CLOUDFLARED_TUNNEL_TOKEN="${derived_token}"
  fi
fi

configure_named_tunnel() {
  if [[ -z "${public_hostname}" ]]; then
    return 0
  fi

  if [[ ! -f "${CERT_FILE}" ]]; then
    echo "cloudflared origin cert not found at ${CERT_FILE}" >&2
    exit 1
  fi

  tunnel_list_json="$(cloudflared tunnel list -o json)"

  tunnel_id="$(
    python3 - "${CLOUDFLARED_TUNNEL_NAME}" "${tunnel_list_json}" <<'PY'
import json
import sys

tunnel_name = sys.argv[1]
tunnels = json.loads(sys.argv[2])
for tunnel in tunnels:
    if tunnel.get("name") == tunnel_name or tunnel.get("id") == tunnel_name:
        print(tunnel["id"])
        raise SystemExit(0)
raise SystemExit(1)
PY
  )"

  read -r account_id api_token < <(
    python3 - "${CERT_FILE}" <<'PY'
import base64
import json
import re
import sys

with open(sys.argv[1], "r") as fh:
    text = fh.read()

# cloudflared cert.pem wraps the JSON-with-account-id-and-api-token in a PEM-style
# block: -----BEGIN ARGO TUNNEL TOKEN----- ... -----END ARGO TUNNEL TOKEN-----.
# Strip the markers and any whitespace before base64-decoding.
match = re.search(
    r"-----BEGIN ARGO TUNNEL TOKEN-----(.+?)-----END ARGO TUNNEL TOKEN-----",
    text,
    re.DOTALL,
)
b64 = re.sub(r"\s+", "", match.group(1) if match else text)
payload = base64.b64decode(b64)
data = json.loads(payload)
print(data["accountID"], data["apiToken"])
PY
  )

  desired_ingress_json="$(
    python3 - "${public_hostname}" "${CLOUDFLARED_URL}" <<'PY'
import json
import sys

print(json.dumps([
    {"hostname": sys.argv[1], "service": sys.argv[2]},
    {"service": "http_status:404"},
]))
PY
  )"

  # A fresh tunnel has no config yet; cloudflare returns 404. Drop -f and discard
  # stderr so the missing-config case isn't reported as a script error.
  current_config="$(
    curl -sS \
      -H "Authorization: Bearer ${api_token}" \
      "https://api.cloudflare.com/client/v4/accounts/${account_id}/cfd_tunnel/${tunnel_id}/configurations" \
      2>/dev/null
  )"
  if [[ -z "${current_config}" ]]; then
    current_config="{}"
  fi

  ingress_matches="$(
    python3 - "${current_config}" "${desired_ingress_json}" <<'PY'
import json
import sys

try:
    current = json.loads(sys.argv[1])
except json.JSONDecodeError:
    current = {}
desired = json.loads(sys.argv[2])
got = (((current.get("result") or {}).get("config") or {}).get("ingress") or [])
print("yes" if got == desired else "no")
PY
  )"

  if [[ "${ingress_matches}" == "yes" ]]; then
    echo "tunnel ${CLOUDFLARED_TUNNEL_NAME} ingress already routes ${public_hostname} -> ${CLOUDFLARED_URL}"
  else
    echo "updating tunnel ${CLOUDFLARED_TUNNEL_NAME} ingress: ${public_hostname} -> ${CLOUDFLARED_URL}"
    ingress_payload="$(
      python3 - "${desired_ingress_json}" <<'PY'
import json
import sys

print(json.dumps({"config": {"ingress": json.loads(sys.argv[1])}}))
PY
    )"

    response="$(
      curl -fsS -X PUT \
        -H "Authorization: Bearer ${api_token}" \
        -H "Content-Type: application/json" \
        --data "${ingress_payload}" \
        "https://api.cloudflare.com/client/v4/accounts/${account_id}/cfd_tunnel/${tunnel_id}/configurations"
    )"

    python3 - "${response}" <<'PY'
import json
import sys

response = json.loads(sys.argv[1])
if not response.get("success"):
    print(json.dumps(response), file=sys.stderr)
    raise SystemExit(1)
PY
  fi

  # `cloudflared tunnel route dns` fails if the DNS record already points elsewhere;
  # tolerate that and just print the failure for the operator.
  if ! cloudflared tunnel route dns "${CLOUDFLARED_TUNNEL_NAME}" "${public_hostname}" 2>/tmp/cf-route-dns.err; then
    if grep -q "already exists" /tmp/cf-route-dns.err; then
      echo "DNS route for ${public_hostname} already exists; leaving as-is"
    else
      cat /tmp/cf-route-dns.err >&2
      rm -f /tmp/cf-route-dns.err
      exit 1
    fi
  fi
  rm -f /tmp/cf-route-dns.err
}

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
    configure_named_tunnel
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
CLOUDFLARED_TUNNEL_NAME=${CLOUDFLARED_TUNNEL_NAME}
CLOUDFLARED_PUBLIC_URL=${CLOUDFLARED_PUBLIC_URL}
EOF

new_unit="$(cat <<EOF
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
)"

printf '%s\n' "${new_unit}" >"${SERVICE_FILE}"

systemctl --user daemon-reload
systemctl --user enable "${CLOUDFLARED_SERVICE_NAME}" >/dev/null

# Always restart so the running process picks up the unit's ExecStart. A diff against
# the file on disk isn't enough — the live process can drift from disk if a prior run
# wrote a new unit but only called `enable --now`, which is a no-op when active.
systemctl --user restart "${CLOUDFLARED_SERVICE_NAME}"

sleep 3
systemctl --user --no-pager --full status "${CLOUDFLARED_SERVICE_NAME}" || true

linger_state="$(loginctl show-user "$(id -un)" -p Linger 2>/dev/null | cut -d= -f2 || true)"
if [[ "${linger_state}" != "yes" ]]; then
  echo "warning: loginctl lingering is not enabled for $(id -un); service may not start on host boot until root runs: sudo loginctl enable-linger $(id -un)" >&2
fi

if [[ "${CLOUDFLARED_QUICK_TUNNEL}" == "true" ]]; then
  echo "quick tunnel started; inspect URL with: journalctl --user -u ${CLOUDFLARED_SERVICE_NAME} -n 50 --no-pager"
else
  if [[ -n "${public_hostname}" ]]; then
    echo "named tunnel started for ${public_hostname}"
  else
    echo "named tunnel started; no public hostname configured in deploy script"
  fi
fi

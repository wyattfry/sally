#!/usr/bin/env bash

set -euo pipefail

: "${GUEST_HOST:?Set GUEST_HOST, for example root@172.16.0.150}"
: "${DEPLOY_USER:=wyatt}"
: "${DEPLOY_ROOT:=/opt/sally}"
: "${REPO_URL:=https://github.com/wyattfry/sally.git}"
: "${BRANCH:=main}"
: "${REMOTE_SUDO:=}"
: "${SSH_AUTHORIZED_KEYS_SOURCE:=}"

if [[ -z "${REMOTE_SUDO}" && "${GUEST_HOST}" != root@* ]]; then
  REMOTE_SUDO="sudo"
fi

ssh "${GUEST_HOST}" "${REMOTE_SUDO} bash -s" <<EOF
set -euo pipefail

if command -v cloud-init >/dev/null 2>&1; then
  cloud-init status --wait || true
fi

while fuser /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock >/dev/null 2>&1; do
  sleep 5
done

apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl gnupg git sudo rsync

install -m 0755 -d /etc/apt/keyrings
if [[ ! -f /etc/apt/keyrings/docker.asc ]]; then
  os_id="\$(. /etc/os-release && echo "\${ID}")"
  curl -fsSL "https://download.docker.com/linux/\${os_id}/gpg" -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
fi

os_id="\$(. /etc/os-release && echo "\${ID}")"
codename="\$(. /etc/os-release && echo "\${VERSION_CODENAME}")"
cat >/etc/apt/sources.list.d/docker.list <<DOCKER
deb [arch=\$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/\${os_id} \${codename} stable
DOCKER

if [[ ! -f /usr/share/keyrings/cloudflare-main.gpg ]]; then
  mkdir -p --mode=0755 /usr/share/keyrings
  curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null
fi
echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main" >/etc/apt/sources.list.d/cloudflared.list

apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin cloudflared

if ! id -u "${DEPLOY_USER}" >/dev/null 2>&1; then
  useradd -m -s /bin/bash "${DEPLOY_USER}"
fi

usermod -aG docker "${DEPLOY_USER}"
install -d -o "${DEPLOY_USER}" -g "${DEPLOY_USER}" /home/"${DEPLOY_USER}"/.ssh
authorized_keys_source="${SSH_AUTHORIZED_KEYS_SOURCE}"
if [[ -z "\${authorized_keys_source}" && -n "\${SUDO_USER:-}" && -f "/home/\${SUDO_USER}/.ssh/authorized_keys" ]]; then
  authorized_keys_source="/home/\${SUDO_USER}/.ssh/authorized_keys"
fi
if [[ -z "\${authorized_keys_source}" && -f /root/.ssh/authorized_keys ]]; then
  authorized_keys_source="/root/.ssh/authorized_keys"
fi
if [[ -n "\${authorized_keys_source}" && -f "\${authorized_keys_source}" ]]; then
  cp "\${authorized_keys_source}" /home/"${DEPLOY_USER}"/.ssh/authorized_keys
  chown "${DEPLOY_USER}:${DEPLOY_USER}" /home/"${DEPLOY_USER}"/.ssh/authorized_keys
  chmod 600 /home/"${DEPLOY_USER}"/.ssh/authorized_keys
fi

echo "${DEPLOY_USER} ALL=(ALL) NOPASSWD:ALL" >/etc/sudoers.d/90-sally-deploy
chmod 0440 /etc/sudoers.d/90-sally-deploy

install -d -o "${DEPLOY_USER}" -g "${DEPLOY_USER}" "${DEPLOY_ROOT}"
if [[ -d "${DEPLOY_ROOT}/.git" ]]; then
  sudo -u "${DEPLOY_USER}" git -C "${DEPLOY_ROOT}" remote set-url origin "${REPO_URL}"
  sudo -u "${DEPLOY_USER}" git -C "${DEPLOY_ROOT}" fetch origin "${BRANCH}"
else
  rm -rf "${DEPLOY_ROOT:?}/"*
  sudo -u "${DEPLOY_USER}" git clone --branch "${BRANCH}" "${REPO_URL}" "${DEPLOY_ROOT}"
fi

if [[ -f "${DEPLOY_ROOT}/infra/dev-server/templates/sally-compose.service" ]]; then
  install -m 0644 "${DEPLOY_ROOT}/infra/dev-server/templates/sally-compose.service" /etc/systemd/system/sally-compose.service
  systemctl daemon-reload
  systemctl enable sally-compose.service
else
  echo "Skipping sally-compose.service install; infra/dev-server is not present in ${DEPLOY_ROOT}"
fi

loginctl enable-linger "${DEPLOY_USER}" || true
systemctl enable --now docker
EOF

echo "Bootstrapped ${GUEST_HOST}. Next: ssh ${DEPLOY_USER}@${GUEST_HOST#*@} and configure ${DEPLOY_ROOT}/.env"

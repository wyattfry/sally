# Sally Dev Server Infra

This directory contains the repo-local setup path for a Sally dev server on the
home Proxmox cluster.

The recommended target is `pve-04` at `172.16.0.204`. As of the last inspection,
it had the best mix of free memory and bulk storage:

| Node | Address | CPU | Available RAM | Useful storage |
| --- | --- | ---: | ---: | --- |
| `pve-02` | `172.16.0.202` | 4 cores | ~3.5 GiB | `local-lvm` ~754 GiB, `fastssd` ~110 GiB |
| `pve-03` | `172.16.0.203` | 4 cores | ~2.3 GiB | `local-lvm` ~324 GiB |
| `pve-04` | `172.16.0.204` | 4 cores | ~10.6 GiB | `tank2` ~626 GiB |

The existing `sally-dev` reference host is LXC `127` on `pve-02`, reachable at
`root@172.16.0.147`. It runs Docker, Cloudflared, a GitHub Actions runner, and
user-level Sally systemd services. This scaffold keeps that proven shape but uses
Docker Compose for the app runtime.

## Layout

- `scripts/inspect-proxmox-nodes.sh` compares Proxmox node capacity.
- `scripts/create-proxmox-vm.sh` creates an Ubuntu cloud-init VM suitable for Docker.
- `scripts/bootstrap-guest.sh` installs Docker, Cloudflared, Git, and a deploy user.
- `scripts/install-github-runner.sh` installs the VM-local GitHub Actions runner.
- `templates/docker-compose.sally-dev.yml` runs Sally, Postgres, and Cloudflared.
- `templates/sally-compose.service` keeps the Compose stack running under systemd.
- `templates/deploy-sally-dev.sh` is the host-side deploy script GitHub Actions can call.
- `github-actions/deploy-sally-dev.yml` is a workflow template.
- `env.example` lists the expected host environment.

## Quick Start

From this workstation:

```bash
infra/dev-server/scripts/inspect-proxmox-nodes.sh
```

Create or converge the VM on the selected Proxmox node:

```bash
PROXMOX_HOST=root@172.16.0.204 \
  VMID=128 \
  NAME=sally-dev-vm \
  STORAGE=tank2 \
  DISK_SIZE=32G \
  CORES=2 \
  MEMORY_MB=4096 \
  infra/dev-server/scripts/create-proxmox-vm.sh
```

The script is idempotent for an existing `VMID`: it reapplies CPU, memory,
networking, cloud-init user/SSH key, guest-agent, and on-boot settings. It creates
the boot disk only when the VM does not exist. By default it injects this
machine's `~/.ssh/id_rsa.pub`; override with `SSH_PUBLIC_KEY_FILE=...` if needed.
The default cloud-init login is Ubuntu's `ubuntu` user; the bootstrap script then
creates the long-lived `wyatt` deploy user.

After the VM has an IP address, bootstrap or converge the guest:

```bash
GUEST_HOST=ubuntu@<guest-ip> \
  DEPLOY_USER=wyatt \
  infra/dev-server/scripts/bootstrap-guest.sh
```

Copy `env.example` into `/opt/sally/.env` on the guest and fill in secrets. Then
deploy:

```bash
ssh wyatt@<guest-ip> 'cd /opt/sally && ./infra/dev-server/templates/deploy-sally-dev.sh'
```

## GitHub Actions

The live workflow is `.github/workflows/deploy-sally-dev.yml`; the copy in
`github-actions/` is kept as a portable template.

The deploy workflow runs on a self-hosted runner labeled `sally-dev` on the VM.
This is required because the VM address is private RFC1918 space and is not
reachable from GitHub-hosted runners. To install or converge the runner:

```bash
GITHUB_RUNNER_TOKEN="$(gh api -X POST repos/wyattfry/sally/actions/runners/registration-token -q .token)" \
  GUEST_HOST=wyatt@<guest-ip> \
  infra/dev-server/scripts/install-github-runner.sh
```

The workflow reads these `development` environment secrets:

- `SALLY_DEV_POSTGRES_PASSWORD`
- `CLOUDFLARED_TUNNEL_TOKEN`
- `SESSION_SECRET`
- optional provider/OAuth secrets such as `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`,
  and `GOOGLE_CLIENT_SECRET`

The workflow reads these `development` environment variables:

- `VITE_SALLY_BACKEND_BASE_URL`
- `LLM_PROVIDER`
- optional provider/OAuth variables such as `OPENAI_MODEL`, `OPENAI_BASE_URL`,
  `ANTHROPIC_MODEL`, `GOOGLE_CLIENT_ID`, and `GOOGLE_REDIRECT_URL`

As of this setup, the `development` environment has already been populated from
the working VM at `172.16.0.148`.

## Notes

- The script creates a VM, not an LXC, because Sally's dev server is a Docker
  Compose host and should not depend on nested container behavior inside LXC.
- The scripts are intentionally convergent. Re-running them should download only
  missing packages/templates, reapply desired settings, preserve existing Postgres
  data, and restart the Compose stack from the current `main` branch.
- Cloudflared is run in the Compose stack. Keep the tunnel token in `.env` or in
  GitHub Actions secrets, not in committed files.
- Postgres data lives in a named Docker volume. Add a `pg_dump` backup timer
  before relying on this for production-like data.

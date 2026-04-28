# Deployment Notes

## Environment Split

- Local backend iteration happens on the current development machine with `docker compose`.
- Shared dev lives on a dedicated development host/container.
- The extension's normal dev/test target is the configured development backend URL, not the local compose instance.
- GitHub Actions uses the repository `development` environment on the self-hosted runner to deploy the Go binary directly on that development host.

## Local Compose

Use local compose when you want fast backend iteration without touching the shared dev box.

```bash
docker compose up --build
```

- service: `sally-server`
- published port: `8080`
- expected local health check: `http://localhost:8080/healthz`
- env source: `server/.env`

If `server/.env` is absent, Compose will warn. Create it only when you need non-default provider settings for local iteration.

## Shared Development Host

The extension should normally point at the dedicated development backend URL via:

```text
VITE_SALLY_BACKEND_BASE_URL=http://<development-host>:8080
```

That environment is for shared integration testing. Keep local Compose and the shared development host separate so extension testing stays predictable.

## GitHub Actions

The self-hosted workflow in the `development` environment does three things:

1. runs `go test ./...`
2. builds the `sally-server` binary
3. uploads the server binary artifact, optionally builds a `sally-server:dev` image when Docker is usable on the runner host, and deploys the binary directly on the runner host

The `development` environment should hold the runtime config needed for the selected extractor, including:

- `LLM_PROVIDER`
- optional OpenAI settings:
  - `OPENAI_API_KEY`
  - `OPENAI_MODEL` defaulting to `gpt-5-mini`
- optional Ollama settings:
  - `OLLAMA_BASE_URL`
  - `OLLAMA_MODEL`
- optional `SALLY_SERVER_PORT` defaulting to `8080`
- optional `SALLY_SERVER_DEPLOY_ROOT` defaulting to `~/.local/share/sally-dev`

The direct deployment path is intended for hosts where Docker is inconvenient or unavailable, including Proxmox LXC guests affected by nested-container AppArmor restrictions.

The deploy script installs a user `systemd` service and manages it with `systemctl --user`. For boot-time startup, the target host must also have lingering enabled for that user, for example:

```bash
sudo loginctl enable-linger wyatt
```

## Optional Public Dev Access

Instead of port forwarding, expose the dev backend with `cloudflared`.

- Quick test path:
  - `CLOUDFLARED_QUICK_TUNNEL=true ./scripts/deploy-cloudflared.sh`
  - read the temporary hostname from `journalctl --user -u sally-cloudflared.service`
- Durable path:
  - `cloudflared tunnel login`
  - `cloudflared tunnel create sally-dev`
  - `cloudflared tunnel route dns sally-dev dev.spexxtool.com`
  - `cloudflared tunnel token sally-dev`
  - `CLOUDFLARED_TUNNEL_TOKEN=... ./scripts/deploy-cloudflared.sh`

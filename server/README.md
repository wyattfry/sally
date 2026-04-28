# Sally Server

## Run

```bash
cd /home/wyatt/sally/server
go run ./cmd/sally-server
```

The server listens on `:8080`.

## Health Check

```bash
curl -i http://localhost:8080/healthz
```

Expected:

- `HTTP/1.1 200 OK`
- body: `ok`

For LAN access from another machine, use the host IP instead, for example:

```bash
curl -i http://<development-host>:8080/healthz
```

## Provider Config

Select the extractor with `LLM_PROVIDER`.

### Stub

- `LLM_PROVIDER=stub`

### OpenAI

- `LLM_PROVIDER=openai`
- `OPENAI_API_KEY`
- `OPENAI_MODEL`

### Ollama

- `LLM_PROVIDER=ollama`
- `OLLAMA_BASE_URL`
- `OLLAMA_MODEL`

Example:

```bash
LLM_PROVIDER=ollama OLLAMA_BASE_URL=http://10.0.0.200:11434 OLLAMA_MODEL=qwen2.5:7b go run ./cmd/sally-server
```

## Docker

Build the image directly:

```bash
cd /home/wyatt/sally/server
docker build -t sally-server .
```

Run the backend locally through Compose from the repo root:

```bash
cd /home/wyatt/sally
docker compose up --build
```

Local Compose is for iteration on the current machine. The extension's normal shared dev target should be the backend URL configured in `VITE_SALLY_BACKEND_BASE_URL`.

If the shared dev host is a Proxmox LXC and Docker cannot start nested containers, run the Go binary directly instead:

```bash
cd /home/wyatt/sally
LLM_PROVIDER=ollama OLLAMA_BASE_URL=http://10.0.0.200:11434 OLLAMA_MODEL=qwen2.5:7b PORT=8080 ./scripts/deploy-dev-server.sh
```

`deploy-dev-server.sh` installs a user `systemd` service at `~/.config/systemd/user/sally-server.service` and starts it with `systemctl --user`.

For boot-time startup, the host needs one root-side step:

```bash
sudo loginctl enable-linger wyatt
```

The script warns if lingering is still disabled.

## GitHub Actions Development Environment

The self-hosted workflow runs in the repository `development` environment and deploys the Go binary directly on the runner host.

Store deployment-specific secrets and variables there as needed.

- `LLM_PROVIDER`
- `OPENAI_API_KEY`
- `OPENAI_MODEL`
- `OLLAMA_BASE_URL`
- `OLLAMA_MODEL`
- `SALLY_SERVER_PORT` default: `8080`
- `SALLY_SERVER_DEPLOY_ROOT` default: `~/.local/share/sally-dev`
- `VITE_SALLY_BACKEND_BASE_URL` for the public backend URL, for example `https://dev.spexxtool.com`
- optional `CLOUDFLARED_TUNNEL_NAME` default: `sally-dev`

## Cloudflare Tunnel

For a stable public dev endpoint without port forwarding, run `cloudflared` as a user service on the same host:

```bash
cd /home/wyatt/sally
CLOUDFLARED_QUICK_TUNNEL=true ./scripts/deploy-cloudflared.sh
```

That creates `~/.config/systemd/user/sally-cloudflared.service`.

To read the temporary Quick Tunnel URL:

```bash
journalctl --user -u sally-cloudflared.service -n 50 --no-pager
```

For a durable hostname such as `dev.spexxtool.com`, use a named tunnel instead:

1. One-time Cloudflare auth:
   ```bash
   cloudflared tunnel login
   ```
2. Create the tunnel:
   ```bash
   cloudflared tunnel create sally-dev
   ```
3. Route DNS:
   ```bash
   cloudflared tunnel route dns sally-dev dev.spexxtool.com
   ```
4. Get the tunnel token:
   ```bash
   cloudflared tunnel token sally-dev
   ```
5. Start the service with that token:
   ```bash
   CLOUDFLARED_TUNNEL_TOKEN=... CLOUDFLARED_PUBLIC_URL=https://dev.spexxtool.com ./scripts/deploy-cloudflared.sh
   ```

When `CLOUDFLARED_PUBLIC_URL` is set, the deploy script also:

- ensures the tunnel DNS route exists for that hostname
- pushes the ingress config mapping that hostname to `CLOUDFLARED_URL`

That captures the Cloudflare-side route in code instead of relying on a manual dashboard edit.

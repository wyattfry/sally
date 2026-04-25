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

## OpenAI Config

The real provider is selected only when both of these are set:

- `OPENAI_API_KEY`
- `OPENAI_MODEL`

If neither is set, the server uses the stub extractor.

If only one is set, the server exits at startup.

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
OPENAI_API_KEY=... OPENAI_MODEL=gpt-5-mini PORT=8080 ./scripts/deploy-dev-server.sh
```

## GitHub Actions Development Environment

The self-hosted workflow runs in the repository `development` environment and deploys the Go binary directly on the runner host.

Store deployment-specific secrets there, including:

- `OPENAI_API_KEY`

Optional environment variables:

- `OPENAI_MODEL` default: `gpt-5-mini`
- `SALLY_SERVER_PORT` default: `8080`
- `SALLY_SERVER_DEPLOY_ROOT` default: `~/.local/share/sally-dev`

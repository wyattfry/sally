# Sally & Mothership

This repo contains a Chrome MV3 extension proof of concept for the Sally SPEC capture loop.

The PoC does this:

- injects an always-present green `SPEC` button onto normal webpages
- captures the current page title, URL, visible text, structured product JSON-LD, likely product image, and likely PDF/spec links
- sends the captured page to the local Sally backend and uses the backend extraction response to propose one editable schedule item
- lets the user edit the proposal, choose a zone and category, cancel, view accepted items, or click `OK`
- saves accepted items to `chrome.storage.local`
- shows a short confirmation toast after an item is accepted
- opens a local schedule viewer for thumbnails, project renaming, item removal, and print output
- supports both the green `SPEC` button and the browser context menu invocation

By default, the extension uses backend extraction. Local mock fallback exists only as an explicitly enabled development option for transport-style failures.

The PoC does not include backend storage, auth, a full Mothership dashboard, or product-page detection.

## Go server skeleton

The repo also includes an early Go backend skeleton under `server/`.

Current endpoints:

- `GET /healthz`
- `POST /v1/extract-spec` extraction response using the configured provider

Provider selection is controlled by `LLM_PROVIDER`:

- `stub`: local stub extractor
- `openai`: hosted OpenAI extractor
- `ollama`: Ollama over local or LAN HTTP

Run the backend tests:

```bash
cd server
go test ./...
```

Run the backend locally:

```bash
cd server
go run ./cmd/sally-server
```

By default the server listens on `:8080`. For shared integration testing, point the extension at your development backend host with `VITE_SALLY_BACKEND_BASE_URL`.

For local backend iteration on the current development machine, use Docker Compose:

```bash
docker compose up --build
```

That local container path is separate from the shared development host used for integration testing.
If your shared development host is a Proxmox LXC, Docker-in-LXC may still fail because of host AppArmor policy. In that case, run the Go server directly on the host and use the self-hosted runner to deploy the binary.

## Development

Install dependencies:

```bash
npm install
```

Optional extension env config:

```bash
cp .env.example .env
```

- `VITE_SALLY_BACKEND_BASE_URL` should point at your shared development backend host, for example `http://<development-host>:8080`
- `VITE_SALLY_ALLOW_MOCK_FALLBACK` defaults to `false`
- mock fallback is intended for development only, must be enabled explicitly, and is only used for transport or unreachable-backend failures
- backend provider examples:
  - `LLM_PROVIDER=stub`
  - `LLM_PROVIDER=openai` with `OPENAI_API_KEY` and `OPENAI_MODEL`
  - `LLM_PROVIDER=ollama` with `OLLAMA_BASE_URL` and `OLLAMA_MODEL`

Run tests:

```bash
npm test
```

Build the extension:

```bash
npm run build
```

Load the extension in Chrome:

1. Open `chrome://extensions`.
2. Enable Developer mode.
3. Click Load unpacked.
4. Select the generated `dist/` directory.
5. Open or refresh a normal `http://` or `https://` webpage.
6. Click the green `SPEC` button.

Quick backend check:

```bash
curl -i http://localhost:8080/healthz
```

Expected result:

- `HTTP/1.1 200 OK`
- body: `ok`

## Dev Deployment Split

- Local backend iteration: `docker compose up --build` on the current machine
- Shared dev integration target: the backend URL configured in `VITE_SALLY_BACKEND_BASE_URL`
- GitHub Actions on the self-hosted runner: test, build, and directly deploy the Go server in the repository `development` environment; Docker image artifacts are optional and only built when Docker is usable on that host
- Real extractor runtime on the development host depends on `LLM_PROVIDER`
- Optional GitHub environment variables:
  - `LLM_PROVIDER`
  - `OPENAI_MODEL`
  - `OLLAMA_BASE_URL`
  - `OLLAMA_MODEL`
  - `SALLY_SERVER_PORT`
  - `SALLY_SERVER_DEPLOY_ROOT`

See [docs/plans/2026-04-24-deployment-notes.md](/home/wyatt/sally/docs/plans/2026-04-24-deployment-notes.md) for the concise deployment notes.

The extension stores accepted PoC items locally in the current Chrome profile through `chrome.storage.local`.

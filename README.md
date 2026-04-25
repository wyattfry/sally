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

If `OPENAI_API_KEY` and `OPENAI_MODEL` are both configured, the server uses the hosted OpenAI extractor. If neither is configured, the server uses the stub extractor for local development.

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

By default the server listens on `:8080`. Point the extension at `http://10.0.0.104:8080` when that machine is using `10.0.0.104` on your LAN.

## Development

Install dependencies:

```bash
npm install
```

Optional extension env config:

```bash
cp .env.example .env
```

- `VITE_SALLY_BACKEND_BASE_URL` defaults to `http://10.0.0.104:8080`
- `VITE_SALLY_ALLOW_MOCK_FALLBACK` defaults to `false`
- mock fallback is intended for development only, must be enabled explicitly, and is only used for transport or unreachable-backend failures

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
curl -i http://10.0.0.104:8080/healthz
```

Expected result:

- `HTTP/1.1 200 OK`
- body: `ok`

The extension stores accepted PoC items locally in the current Chrome profile through `chrome.storage.local`.

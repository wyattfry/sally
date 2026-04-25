# Sally & Mothership

This repo contains a Chrome MV3 extension proof of concept for the Sally SPEC capture loop.

The PoC does this:

- injects an always-present green `SPEC` button onto normal webpages
- captures the current page title, URL, visible text, structured product JSON-LD, likely product image, and likely PDF/spec links
- uses a mocked extraction function to propose one editable schedule item
- lets the user edit the proposal, choose a zone and category, cancel, view accepted items, or click `OK`
- saves accepted items to `chrome.storage.local`
- shows a short confirmation toast after an item is accepted
- opens a local schedule viewer for thumbnails, project renaming, item removal, and print output
- supports both the green `SPEC` button and the browser context menu invocation

The PoC does not include backend storage, auth, a full Mothership dashboard, product-page detection, or a real AI call.

## Go server skeleton

The repo also includes an early Go backend skeleton under `server/`.

Current endpoints:

- `GET /healthz`
- `POST /v1/extract-spec` placeholder only

Run the backend tests:

```bash
cd server
go test ./...
```

## Development

Install dependencies:

```bash
npm install
```

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

The extension stores accepted PoC items locally in the current Chrome profile through `chrome.storage.local`.

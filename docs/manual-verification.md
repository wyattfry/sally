# Manual Verification

## Backend

1. On the machine that hosts the backend, make sure the extension `.env` points at `http://10.0.0.104:8080`.
2. To verify the real provider path rather than the stub, make sure both `OPENAI_API_KEY` and `OPENAI_MODEL` are set before starting the server.
3. Make sure `VITE_SALLY_ALLOW_MOCK_FALLBACK=false` for this check so the extension does not mask backend failures with local mock data.
4. In `/home/wyatt/sally/server`, start the server:
   ```bash
   go run ./cmd/sally-server
   ```
5. In another terminal, confirm health:
   ```bash
   curl -i http://10.0.0.104:8080/healthz
   ```
6. Confirm the response is `200 OK` with body `ok`.

The server listens on `:8080`, so it is reachable on the host's LAN IP. If the machine is not currently using `10.0.0.104`, update `.env` and reload the extension with the actual LAN IP.

## Load The Extension

1. In `/home/wyatt/sally`, build the extension:
   ```bash
   npm run build
   ```
2. Open `chrome://extensions`.
3. Enable Developer mode.
4. Click `Load unpacked`.
5. Select `/home/wyatt/sally/dist`.

## Confirm Backend Extraction

1. Open or refresh a normal `http://` or `https://` product page.
2. Click the green `SPEC` button.
3. Confirm Sally opens on the right and briefly shows `Reading page`.
4. Confirm an editable proposal appears.
5. In Chrome DevTools, open `Network` and confirm a `POST` to:
   - `http://10.0.0.104:8080/v1/extract-spec`
6. Confirm the response is `200` and the payload contains `status: "ok"`.
7. Confirm the visible proposal fields in Sally match the `POST /v1/extract-spec` response body.
8. Confirm no fallback toast appears during this check.

## Confirm Real Proposal Flow

1. In Sally, confirm the header shows the project name, initially `My New Project`.
2. Pick a `Zone`.
3. Pick a `Category`.
4. Edit at least `Title`.
5. Click `OK`.
6. Confirm the panel closes and an `Item added` toast appears.
7. Click `SPEC` again, then `View Items`.
8. Confirm the accepted item appears in the viewer with its thumbnail and source link.

## Confirm Failure Behavior

1. Stop the Go server.
2. Refresh the product page.
3. Click `SPEC`.
4. Confirm Sally does not silently produce a proposal.
5. Confirm a visible failure state appears with the backend error and `Retry extraction` / `Dismiss`.
6. Confirm the same failure is also surfaced as a toast.
7. If development fallback is explicitly enabled, confirm fallback only occurs for unreachable or timeout-style backend failures, not backend extraction errors.

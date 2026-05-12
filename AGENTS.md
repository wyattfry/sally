# Sally — Agent Handoff

## What this is

Sally is a Chrome MV3 extension + Go server ("Mothership") that lets architects SPEC product pages
into structured schedule items. Two user personas:

- **Architect** — uses the extension to capture items and the Mothership dashboard to manage projects/schedules
- **Contractor** — receives a share link and views the read-only schedule to buy items

Both paths must work end-to-end before the product is tested with real users.

---

## Repository layout

```
sally/
  src/                   Chrome extension (React + TypeScript, built with Vite)
    App.tsx              Root component — state machine, API calls, event wiring
    components/
      SallyPanel.tsx     Capture/review panel (shown during SPEC flow)
      SpecButton.tsx     Always-visible green SPEC button injected into pages
    lib/
      extractApi.ts      Calls the Go extraction backend via chrome.runtime proxy
      mothershipApi.ts   REST client for the Mothership Go server
      storage.ts         chrome.storage.local helpers (active project/schedule context)
      capturePage.ts     Captures visible text, images, JSON-LD, PDF links from DOM
      types.ts           Shared TypeScript types
  server/                Go server (Mothership dashboard + extraction API)
    cmd/sally-server/
      main.go            Entry point; reads env config; wires extractor + DB + HTTP
    internal/
      config/            Env-var config loader
      db/
        migrate.go       Runs goose migrations from server/migrations/
        generated/       sqlc-generated Go types + query functions (hand-maintained,
                         sqlc is NOT available in dev — see DB section below)
        queries/         SQL source files (*.sql)
      extract/           Shared request/response types for the extraction pipeline
      httpapi/           HTTP handlers for the extraction API (/v1/extract-spec)
      provider/          LLM provider implementations (stub, openai, ollama,
                         chatcompletion, anthropic)
      web/               Mothership server-side HTML (Go templates, net/http)
        project_handlers.go  All route registration + HTML template (single file)
        *_handlers_test.go   Integration tests (require DATABASE_URL)
        static/app.css   Shared stylesheet
  migrations/            goose SQL migration files (applied at server startup)
  docs/                  Design notes and implementation plans
```

When updating Wyatt's project-hours/work-log CSV, see `docs/update-hours.md`.

---

## Extension build

```bash
npm install
npm run build   # outputs to dist/
npm test        # vitest unit tests
```

Load unpacked from `dist/` in Chrome. The extension proxies all HTTP calls through
`chrome.runtime.sendMessage` (PROXY_FETCH messages) to bypass CORS — this is why
`mothershipApi.ts` never uses `fetch` directly.

Relevant env vars (set in `.env`, passed through Vite):

| Variable | Default | Purpose |
|---|---|---|
| `VITE_SALLY_BACKEND_BASE_URL` | — | Extraction + Mothership API base URL |
| `VITE_SALLY_ALLOW_MOCK_FALLBACK` | `false` | Enables mock extraction for transport failures |

---

## Go server

### Run locally

```bash
cd server
go run ./cmd/sally-server
```

Or with Docker Compose from the repo root (starts server + Postgres):

```bash
docker compose up --build
```

### Environment variables

| Variable | Purpose |
|---|---|
| `PORT` | HTTP listen port (default `8080`) |
| `DATABASE_URL` | Postgres DSN; server starts without DB if unset |
| `LLM_PROVIDER` | `stub` \| `openai` \| `ollama` \| `chatcompletion` \| `anthropic` |
| `OPENAI_API_KEY` | Required for `openai` and `chatcompletion` providers |
| `OPENAI_MODEL` | Required for `openai` and `chatcompletion` providers |
| `OPENAI_BASE_URL` | Override base URL for `openai`/`chatcompletion` (e.g. Groq) |
| `OPENAI_TIMEOUT_MS` | Request timeout in milliseconds (default 15000) |
| `CHATCOMPLETION_RESPONSE_FORMAT` | `json_schema` (default) or `json_object` |
| `OLLAMA_BASE_URL` | Required for `ollama` provider |
| `OLLAMA_MODEL` | Required for `ollama` provider |
| `ANTHROPIC_API_KEY` | Required for `anthropic` provider |
| `ANTHROPIC_MODEL` | Required for `anthropic` provider (e.g. `claude-sonnet-4-6`) |
| `GOOGLE_CLIENT_ID` | Required for Google OAuth; omit to run in dev mode (no auth) |
| `GOOGLE_CLIENT_SECRET` | Required for Google OAuth |
| `GOOGLE_REDIRECT_URL` | OAuth callback URL (e.g. `https://dev.spexxtool.com/auth/callback`) |
| `SESSION_SECRET` | HMAC-SHA256 signing key for session cookies; any non-empty string |

### LLM provider notes

- **`stub`** — instant fake response, no API key needed, good for UI work
- **`openai`** — uses the Responses API (`POST /responses`), supports structured output via `json_schema`
- **`chatcompletion`** — OpenAI-compatible chat completions endpoint; works with Groq, Together, etc.
  Use `CHATCOMPLETION_RESPONSE_FORMAT=json_object` for models that don't support `json_schema`
  (e.g. Groq `llama-3.1-8b-instant`). Use a larger model (e.g. `llama-3.3-70b-versatile`) if
  requests exceed the model's TPM limit.
- **`anthropic`** — Anthropic Messages API with tool_use for guaranteed structured JSON.
  Uses `POST /messages` with `x-api-key` + `anthropic-version: 2023-06-01` headers.
  Forces `extract_spec` tool call so output is always structured.

---

## Database

### Schema

Postgres. Migrations use goose (applied automatically on startup via `db/migrate.go`).
Migration files live in `server/migrations/`.

Key tables: `users`, `projects`, `schedules`, `schedule_items`, `project_share_links`.

`project_share_links` stores both `token_hash` (SHA-256 of the plain token, used to look
up public share requests) and `token` (the plain token, stored so the manage page can
always display the copyable URL without requiring the token in the URL query string).

### Working without sqlc

**sqlc is not available in this dev environment.** When adding or modifying SQL queries:

1. Write the SQL in `server/internal/db/queries/<table>.sql`
2. **Manually** add the corresponding Go function and types to
   `server/internal/db/generated/<table>.sql.go`
3. If the query returns rows, update the `Scan(...)` call to match all columns in SELECT order
4. If adding a column, also update the struct in `server/internal/db/generated/models.go`

The generated files have a `// Code generated by sqlc. DO NOT EDIT.` header — ignore it,
we edit them by hand.

### Local dev DB

Docker Compose exposes Postgres at `localhost:5432`:

```
postgres://sally:sally_dev_password@localhost:5432/sally?sslmode=disable
```

Run server tests:

```bash
DATABASE_URL="postgres://sally:sally_dev_password@localhost:5432/sally" \
  go test ./... -v
```

Tests that need the DB call `t.Skip` when `DATABASE_URL` is unset — safe to run without it.

---

## Mothership web (server-side rendered)

Routes and the HTML template all live in `server/internal/web/project_handlers.go`.
Auth handlers live in `server/internal/web/auth_handlers.go`.
Session cookie helpers live in `server/internal/web/session.go`.
The entire UI is one `html/template` block at the bottom of `project_handlers.go`, switching on `page.Kind`.

**Template conventions:**
- Each page kind passes a typed Go struct as template data
- Breadcrumb chains follow the pattern: `Projects / Project Name / Schedule Name`
- The `edit-schedule` and `edit-item` pages include a full breadcrumb back to the Projects list
- `requestBaseURL(r)` builds `scheme://host` for share links (checks `X-Forwarded-Proto` first)
- `$.Project.ID` (dollar-sign) accesses the outer template context inside nested `range` blocks
- Capture loop variables before inner `range`: `{{$s := .Schedule}}` then use `$s.ID` in item rows

**Project page layout:**
- All schedules shown on the project detail page as collapsible `<details>` sections
- Sticky sidebar nav (`schedule-nav`) with anchor links (`#schedule-{id}`) for quick-jumping
- Item titles link to `SourceUrl` when present; `SourceImageUrl` renders as a 56×56px thumbnail
- `GET /projects/{id}/schedules/{scheduleID}` (old per-schedule URL) redirects 303 to `#schedule-{id}`
- All post-action redirects (create/update/delete item or schedule) go to the project page with anchor

**Auth flow (Google OAuth):**
1. When `GOOGLE_CLIENT_ID` is set, all protected routes call `requireUser` which checks `sally_session` cookie
2. Unauthenticated → redirect to `GET /login` which renders the sign-in page
3. User clicks "Sign in with Google" → `GET /auth/google` → sets state cookie, redirects to Google
4. Google → `GET /auth/callback?code=...&state=...` → exchanges code, upserts user, sets signed session cookie
5. `POST /logout` → clears cookie, redirects to `/login`
6. Session cookie value is `email|base64(HMAC-SHA256(secret, email))` — tamper-evident, verified on every request
7. When `GOOGLE_CLIENT_ID` is unset (dev mode), `requireUser` upserts the `DevUserEmail` dev user instead

**Share link flow:**
1. Owner clicks "Enable sharing" → `POST /projects/{id}/share-links`
2. Handler deactivates any existing active link, creates a new one, stores both hash and plain token
3. Redirects to `GET /projects/{id}/share` (no token in URL)
4. Manage page reads active link from DB and shows full copyable URL
5. Public link: `GET /share/{token}` — looks up by SHA-256 hash, renders read-only view
6. "Disable sharing" → `POST /projects/{id}/share-links/deactivate`

---

## Adding a new LLM provider

1. Create `server/internal/provider/<name>.go` — implement `provider.Extractor` interface
   (`Extract` + `Meta` methods). See `anthropic.go` for a clean example.
2. Add config fields to `server/internal/config/config.go`
3. Add a `case "<name>":` to `newExtractor()` in `server/cmd/sally-server/main.go`
4. Add a `validate<Name>Config()` function in the same file
5. Write unit tests in `server/internal/provider/<name>_test.go` using `httptest.NewServer`
6. Document the new env vars in this file and in `README.md` / `server/README.md`

Reusable helpers already in the package:
- `extractionSchema()` — JSON schema for structured output / tool definitions
- `buildUserPrompt(req)` — formats page content + project context into a user message
- `openAIExtractionOutput` struct — target for unmarshalling model output
- `coalesceStrings`, `coalesceFinishMappings` — nil-safe slice helpers
- `isTimeoutError`, `summarizeUpstreamBody`, `truncate` — error/logging helpers

---

## Test strategy

- **Extension**: Vitest unit tests in `src/`. Run with `npm test`. Note: ~12 tests are
  currently failing due to a pre-existing mismatch between the test file and the current
  component API (references to removed features like Zone, local viewer, toast). Do not
  treat these as regressions from recent work.
- **Server**: Standard `go test ./...`. DB-dependent tests skip when `DATABASE_URL` is unset.
  Provider tests use `httptest.NewServer` and do not require API keys.

---

## Key product decisions

- **SPEC is the verb** — not Pin, not Save, not Add
- **AI proposes; human approves** — no silent writes
- `onCreateProject` / `onCreateSchedule` callbacks in SallyPanel return `Promise<string | null>`
  (null = success, string = error message shown inline). They must never call `setPanel` directly
  — that would destroy the in-progress review draft.
- Share tokens: only the SHA-256 hash is used for lookup; the plain token is stored for display

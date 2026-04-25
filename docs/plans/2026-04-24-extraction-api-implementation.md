# Sally Extraction API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Go backend with a `POST /v1/extract-spec` endpoint, wire the extension to call it, and preserve the current Sally review UX while replacing the mock-only extraction path.

**Architecture:** Keep the system split into a TypeScript Chrome extension and a narrow Go HTTP service in the same repo. Implement the backend in stages: first a contract-accurate stub, then a provider interface, then a real hosted model adapter, while the extension switches from local mock extraction to backend extraction with controlled dev fallback.

**Tech Stack:** Go standard library, Go `httptest`, TypeScript, React, Vitest, Chrome MV3, fetch, hosted multimodal model API.

---

### Task 1: Add Backend Skeleton And Config

**Files:**
- Create: `server/go.mod`
- Create: `server/cmd/sally-server/main.go`
- Create: `server/internal/config/config.go`
- Create: `server/internal/httpapi/router.go`
- Create: `server/internal/httpapi/router_test.go`
- Modify: `README.md`

**Step 1: Write the failing router test**

Create `server/internal/httpapi/router_test.go` with a test that constructs the router and sends:

```go
req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
rr := httptest.NewRecorder()
router.ServeHTTP(rr, req)
if rr.Code != http.StatusOK {
	t.Fatalf("expected 200, got %d", rr.Code)
}
```

**Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/httpapi`

Expected: FAIL because the module and router do not exist yet.

**Step 3: Write minimal implementation**

- Create `server/go.mod`
- Implement a minimal router with:
  - `GET /healthz` returning `200 OK`
  - placeholder registration for `POST /v1/extract-spec`
- Add config loading for:
  - `PORT`
  - `OPENAI_API_KEY`
  - `SALLY_ALLOW_MOCK_FALLBACK`

**Step 4: Run test to verify it passes**

Run: `cd server && go test ./internal/httpapi`

Expected: PASS

**Step 5: Commit**

```bash
git add server/go.mod server/cmd/sally-server/main.go server/internal/config/config.go server/internal/httpapi/router.go server/internal/httpapi/router_test.go README.md
git commit -m "feat: add go server skeleton"
```

### Task 2: Define Extraction Contract Types

**Files:**
- Create: `server/internal/extract/types.go`
- Create: `server/internal/extract/types_test.go`
- Modify: `docs/plans/2026-04-24-extraction-api-design.md`

**Step 1: Write the failing contract tests**

Create tests that JSON-unmarshal example request and response payloads and assert:

- required top-level fields are present
- `status` accepts `ok` and `error`
- `proposal.title`, `proposal.sourceUrl`, `meta.provider`, and `meta.promptVersion` decode correctly

Use literal JSON fixtures embedded in the test.

**Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/extract`

Expected: FAIL because the types do not exist yet.

**Step 3: Write minimal implementation**

Create explicit structs for:

- `ExtractSpecRequest`
- `ClientInfo`
- `PagePayload`
- `ProjectContext`
- `ExtractSpecOptions`
- `ExtractSpecResponse`
- `Proposal`
- `FinishModelMapping`
- `Analysis`
- `Confidence`
- `ErrorPayload`
- `ResponseMeta`

Use JSON tags matching the approved contract.

**Step 4: Run test to verify it passes**

Run: `cd server && go test ./internal/extract`

Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/extract/types.go server/internal/extract/types_test.go docs/plans/2026-04-24-extraction-api-design.md
git commit -m "feat: define extraction api types"
```

### Task 3: Add Stub Extract Handler

**Files:**
- Create: `server/internal/httpapi/extract_handler.go`
- Create: `server/internal/httpapi/extract_handler_test.go`
- Modify: `server/internal/httpapi/router.go`

**Step 1: Write the failing handler tests**

Add tests for:

1. valid `POST /v1/extract-spec` returns `200` and `status: "ok"`
2. invalid JSON returns `400`
3. wrong method returns `405`

The happy-path test should assert that the response includes:

- same `requestId` as request
- populated `proposal`
- populated `meta`

**Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/httpapi`

Expected: FAIL because the handler does not exist.

**Step 3: Write minimal implementation**

Implement a stub handler that:

- decodes `ExtractSpecRequest`
- validates required fields lightly
- returns a hard-coded but contract-valid proposal derived from request fields
- sets `meta.provider` to `"stub"`
- sets `meta.promptVersion` to `"extract-spec-v1"`

Do not call any model provider yet.

**Step 4: Run test to verify it passes**

Run: `cd server && go test ./internal/httpapi`

Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/httpapi/extract_handler.go server/internal/httpapi/extract_handler_test.go server/internal/httpapi/router.go
git commit -m "feat: add stub extraction handler"
```

### Task 4: Add Extension-Side API Client

**Files:**
- Create: `src/lib/extractApi.ts`
- Create: `src/lib/extractApi.test.ts`
- Modify: `src/lib/types.ts`
- Modify: `src/App.tsx`

**Step 1: Write the failing client tests**

Add tests for:

1. building the request from captured page plus project context
2. decoding a successful response into a draft proposal
3. throwing on non-OK backend responses
4. honoring timeout behavior with `AbortController`

Mock `fetch`.

**Step 2: Run test to verify it fails**

Run: `npm test -- src/lib/extractApi.test.ts`

Expected: FAIL because `extractApi.ts` does not exist.

**Step 3: Write minimal implementation**

Create:

- extension-side request/response types mirroring the backend contract
- `extractScheduleItem(...)`
- request builder using:
  - `capturePage(...)`
  - `projectName`
  - known zones
  - known categories
- timeout handling at 18 seconds

In `src/App.tsx`, replace the direct `mockExtractScheduleItem(captured)` path with the API client, but keep the current review panel behavior.

**Step 4: Run test to verify it passes**

Run: `npm test -- src/lib/extractApi.test.ts src/App.test.tsx`

Expected: PASS

**Step 5: Commit**

```bash
git add src/lib/extractApi.ts src/lib/extractApi.test.ts src/lib/types.ts src/App.tsx
git commit -m "feat: call extraction backend from extension"
```

### Task 5: Add Dev Config And Controlled Mock Fallback

**Files:**
- Modify: `src/App.tsx`
- Modify: `src/lib/extractApi.ts`
- Modify: `public/manifest.json`
- Modify: `README.md`
- Create: `.env.example`

**Step 1: Write the failing tests**

Add tests covering:

1. backend failure in dev uses mock extraction when fallback is enabled
2. backend failure in production-facing mode does not silently fall back
3. failed extraction leaves the panel recoverable and user-visible

**Step 2: Run test to verify it fails**

Run: `npm test -- src/App.test.tsx`

Expected: FAIL because fallback and error behavior are not implemented yet.

**Step 3: Write minimal implementation**

- Add extension config for backend base URL, defaulting to `http://10.0.0.104:<port>`
- add `host_permissions` in the manifest for the backend origin
- allow mock fallback only behind explicit dev config
- show a clear user-facing error state or toast when extraction fails without fallback

Do not bury errors.

**Step 4: Run test to verify it passes**

Run: `npm test -- src/App.test.tsx`

Expected: PASS

**Step 5: Commit**

```bash
git add src/App.tsx src/lib/extractApi.ts public/manifest.json README.md .env.example
git commit -m "feat: add backend config and dev fallback"
```

### Task 6: Add Provider Interface In Go

**Files:**
- Create: `server/internal/provider/provider.go`
- Create: `server/internal/provider/provider_test.go`
- Modify: `server/internal/httpapi/extract_handler.go`

**Step 1: Write the failing provider tests**

Create tests for a provider interface that returns:

- a successful proposal
- a timeout-style error
- a provider failure

The handler test should verify that provider errors are converted into the contract error response.

**Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/provider ./internal/httpapi`

Expected: FAIL because the provider abstraction does not exist yet.

**Step 3: Write minimal implementation**

Create an interface like:

```go
type Extractor interface {
	Extract(ctx context.Context, req extract.ExtractSpecRequest) (extract.ExtractSpecResponse, error)
}
```

Wire the handler to depend on that interface instead of inline stub logic. Keep the existing stub as one implementation.

**Step 4: Run test to verify it passes**

Run: `cd server && go test ./internal/provider ./internal/httpapi`

Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/provider/provider.go server/internal/provider/provider_test.go server/internal/httpapi/extract_handler.go
git commit -m "refactor: add provider interface for extraction"
```

### Task 7: Implement Hosted Model Adapter

**Files:**
- Create: `server/internal/provider/openai.go`
- Create: `server/internal/provider/openai_test.go`
- Modify: `server/internal/config/config.go`
- Modify: `server/cmd/sally-server/main.go`

**Step 1: Write the failing adapter tests**

Mock the upstream model HTTP API and verify:

1. request includes model name and prompt version
2. request includes page text and image URL when present
3. structured model output is converted into the approved contract
4. upstream timeout maps to `MODEL_TIMEOUT`

**Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/provider`

Expected: FAIL because the hosted adapter does not exist yet.

**Step 3: Write minimal implementation**

Implement the first real provider adapter against the chosen hosted model API using:

- environment-configured API key
- environment-configured model name
- fixed prompt version string
- request timeout from config

The adapter should:

- build a schema-constrained extraction request
- map provider output into `proposal`, `analysis`, and `meta`
- preserve source URL/title/image/PDF links from the original page payload

**Step 4: Run test to verify it passes**

Run: `cd server && go test ./internal/provider`

Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/provider/openai.go server/internal/provider/openai_test.go server/internal/config/config.go server/cmd/sally-server/main.go
git commit -m "feat: add hosted extraction provider"
```

### Task 8: End-To-End Local Verification

**Files:**
- Modify: `docs/manual-verification.md`
- Modify: `README.md`
- Create: `server/README.md`

**Step 1: Write the manual verification checklist updates**

Add exact run steps for:

- starting the Go server on the shared dev host reachable at `10.0.0.104`
- loading the unpacked extension
- confirming the extension calls the backend
- confirming a real proposal appears in Sally
- confirming failure behavior when the backend is unavailable

**Step 2: Run tests and build**

Run:

```bash
cd server && go test ./...
cd /home/wyatt/sally && npm test
cd /home/wyatt/sally && npm run build
```

Expected:

- Go tests pass
- Vitest suite passes
- extension build passes

**Step 3: Run the server locally**

Run:

```bash
cd server && go run ./cmd/sally-server
```

Expected:

- server listens on configured port
- `GET /healthz` returns `200 OK`

**Step 4: Manually verify the extension flow**

Confirm:

- `SPEC` opens Sally
- backend request occurs
- proposal returns from backend
- user edits and accepts as before
- accepted item still saves locally

**Step 5: Commit**

```bash
git add docs/manual-verification.md README.md server/README.md
git commit -m "docs: add extraction backend setup and verification"
```

### Task 9: Local Compose And Self-Hosted Dev Deployment Prep

**Files:**
- Create: `docker-compose.yml`
- Create: `server/Dockerfile`
- Create: `.github/workflows/server-build.yml`
- Create: `docs/plans/2026-04-24-deployment-notes.md`
- Modify: `README.md`
- Modify: `server/README.md`

**Step 1: Define the environment split**

Document the intended workflow:

- local backend iteration happens on the current development machine via Docker Compose
- the shared dev environment lives at `10.0.0.104`
- the extension's normal dev/test target is `10.0.0.104`
- GitHub Actions builds and deploys the server to `10.0.0.104`

Keep this split explicit so local fast iteration and shared integration testing do not get conflated.

**Step 2: Write the failing workflow or build expectation**

Define the expected outputs:

- build Go binary
- build Docker image
- support `docker compose up` for local backend development
- make the server artifact available to the self-hosted dev environment at `10.0.0.104`

**Step 3: Implement minimal local/dev deployment assets**

- local `docker-compose.yml` for the backend service
- multi-stage Dockerfile for the Go server
- GitHub Actions workflow that runs on the self-hosted runner, executes `go test ./...`, builds the server artifact, and prepares deployment to `10.0.0.104`
- docs describing:
  - local compose usage
  - expected env file handling
  - how `10.0.0.104` differs from local compose

Keep deployment simple. Do not automate production rollout yet.

**Step 4: Verify locally where possible**

Run:

```bash
docker compose up --build
cd server && docker build -t sally-server .
```

Expected:

- compose starts the local backend container successfully
- container image builds successfully

**Step 5: Commit**

```bash
git add docker-compose.yml server/Dockerfile .github/workflows/server-build.yml docs/plans/2026-04-24-deployment-notes.md README.md server/README.md
git commit -m "chore: add local and dev deployment assets"
```

# Sally Mother Ship Design

## Goal

Build the first durable Mother Ship web app: a conventional CRUD application where architects manage projects, schedules, schedule items, and project share links.

## Product Shape

Sally remains the capture assistant. The Mother Ship becomes the system of record.

The extension captures product-page data, asks the backend to extract a proposed schedule item, and eventually saves accepted items into a selected project schedule. Browser storage should become a local cache or fallback, not persistence.

The Mother Ship app should be conventional:

- users have projects
- projects have schedules
- schedules have items
- projects can generate share links
- contractors can open share links without accounts and see the exact items to procure

## Service Boundary

Use one Go application for the next phase.

Routes:

- Dashboard and CRUD pages live at `https://dev.spexxtool.com/`
- API routes live under `https://dev.spexxtool.com/api/v1/...`
- The existing extension extraction endpoint moves from `/v1/extract-spec` to `/api/v1/extract-spec`
- The old `/v1/extract-spec` route can remain temporarily as a compatibility alias

This keeps deployment, database access, auth/session handling, and provider configuration in one binary. The app can be split into separate services later if traffic, ownership, or operational constraints justify it.

## Recommended Stack

- Go HTTP server
- Postgres
- `sqlc` for typed SQL queries
- `goose` for database migrations
- `templ` for server-rendered pages/components
- HTMX for small progressive interactions where useful
- Docker Compose for local and dev VM deployment

This is deliberately boring. Sally's Mother Ship is mostly records, forms, tables, print views, and share pages.

## Initial Data Model

### users

Represents an authenticated architect account.

Initial local development can use a single seeded/dev user before Google OAuth is wired in.

Fields:

- id
- email
- name
- created_at
- updated_at

### projects

Highest-level user-owned entity.

Fields:

- id
- owner_user_id
- name
- address
- created_at
- updated_at

### schedules

Named groups of project items, such as Bath, Kitchen, Lighting, Appliances, Hardware, or Paint.

Fields:

- id
- project_id
- name
- position
- created_at
- updated_at

### schedule_items

Editable procurement/specification rows.

Fields:

- id
- schedule_id
- code
- title
- description
- manufacturer
- model_number
- finish
- finish_model_number
- notes
- source_url
- source_title
- source_image_url
- source_pdf_links
- position
- created_at
- updated_at

### project_share_links

Opaque public links for contractors.

Fields:

- id
- project_id
- token_hash
- label
- active
- created_at
- updated_at
- last_viewed_at

Use project-level share links first. Schedule-specific links can be added later.

## Initial Pages

Authenticated:

- `GET /` redirects to `/projects`
- `GET /projects`
- `GET /projects/new`
- `POST /projects`
- `GET /projects/{projectID}`
- `GET /projects/{projectID}/edit`
- `POST /projects/{projectID}`
- `POST /projects/{projectID}/delete`
- `GET /projects/{projectID}/schedules/new`
- `POST /projects/{projectID}/schedules`
- `GET /projects/{projectID}/schedules/{scheduleID}`
- `GET /projects/{projectID}/schedules/{scheduleID}/items/new`
- `POST /projects/{projectID}/schedules/{scheduleID}/items`
- `GET /projects/{projectID}/share`
- `POST /projects/{projectID}/share-links`

Public:

- `GET /share/{token}`

API:

- `GET /healthz`
- `POST /api/v1/extract-spec`
- `POST /v1/extract-spec` compatibility alias

## Testing Strategy

Use normal Go tests for most behavior.

- Handler tests use `httptest`
- Database tests run against Postgres
- Query behavior is tested through `sqlc` generated code
- HTML responses are parsed with `goquery` where useful
- Share-token generation and lookup get focused unit/integration tests
- Playwright can be added later for browser-level flows

Avoid SQLite for tests because production behavior should match Postgres.

## Deployment

Local:

- Docker Compose with Go app and Postgres

Dev:

- Proxmox full Linux VM
- Docker Compose
- `dev.spexxtool.com` points at the Mother Ship app
- API lives at `dev.spexxtool.com/api/v1`

Later:

- add backups
- add reverse proxy or tunnel hardening
- consider Kubernetes only when deployment complexity actually warrants it


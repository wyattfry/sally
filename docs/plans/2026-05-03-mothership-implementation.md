# Sally Mother Ship Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the first persistent Mother Ship CRUD web app for projects, schedules, items, and project share links.

**Architecture:** Extend the existing Go server into a unified web/API application. Dashboard pages are server-rendered at `/`; extension and integration endpoints live under `/api/v1`; persistence uses Postgres with migrations and typed SQL queries.

**Tech Stack:** Go, Postgres, Docker Compose, goose migrations, sqlc query generation, templ server-rendered components, HTMX where useful, Go `httptest`, Postgres-backed integration tests.

---

### Task 1: Route Namespace And Compatibility Alias

**Files:**
- Modify: `server/internal/httpapi/router.go`
- Modify: `server/internal/httpapi/router_test.go`
- Modify: `src/lib/extractApi.ts`
- Modify: `src/lib/extractApi.test.ts`

**Steps:**

1. Add a failing Go router test proving `POST /api/v1/extract-spec` is registered.
2. Add a failing Go router test proving `POST /v1/extract-spec` still works as a compatibility alias.
3. Update `NewRouterWithExtractor` so both paths use `NewExtractHandler(extractor)`.
4. Run `cd server && go test ./internal/httpapi`.
5. Update the extension API path builder to call `/api/v1/extract-spec`.
6. Keep a frontend test proving configured backend base URLs combine correctly with `/api/v1/extract-spec`.
7. Run `npm test -- src/lib/extractApi.test.ts`.

### Task 2: Add Postgres To Docker Compose

**Files:**
- Modify: `docker-compose.yml`
- Modify: `server/README.md`
- Modify: `README.md`

**Steps:**

1. Add a `postgres` service with database, user, password, and named volume.
2. Add `DATABASE_URL` to the `sally-server` service environment.
3. Add `depends_on` from `sally-server` to `postgres`.
4. Document local startup and database URL.
5. Run `docker compose config` to validate Compose syntax.

### Task 3: Add Migrations

**Files:**
- Create: `server/migrations/00001_create_mothership_tables.sql`
- Modify: `server/go.mod`
- Create: `server/internal/db/migrate.go`
- Create: `server/internal/db/migrate_test.go`

**Steps:**

1. Add `goose` dependency.
2. Write migration with `users`, `projects`, `schedules`, `schedule_items`, and `project_share_links`.
3. Add indexes for owner/project/schedule lookups and active share links.
4. Add `RunMigrations(ctx, db, migrationsDir)` helper.
5. Add a Postgres-backed migration test if `DATABASE_URL` is set; skip clearly if not.
6. Run `cd server && go test ./internal/db`.

### Task 4: Add sqlc Queries

**Files:**
- Create: `server/sqlc.yaml`
- Create: `server/internal/db/queries/projects.sql`
- Create: `server/internal/db/queries/schedules.sql`
- Create: `server/internal/db/queries/items.sql`
- Create: `server/internal/db/queries/share_links.sql`
- Generated: `server/internal/db/generated/*`

**Steps:**

1. Add `sqlc` configuration.
2. Write project CRUD queries.
3. Write schedule CRUD queries.
4. Write item CRUD queries.
5. Write share-link create/list/lookup queries.
6. Generate code with `cd server && sqlc generate`.
7. Add focused query tests against Postgres if `DATABASE_URL` is set.

### Task 5: Server Configuration And Database Wiring

**Files:**
- Modify: `server/internal/config/config.go`
- Modify: `server/internal/config/config_test.go`
- Modify: `server/cmd/sally-server/main.go`
- Modify: `server/internal/httpapi/router.go`

**Steps:**

1. Add `DatabaseURL` to config.
2. Test `DATABASE_URL` loading.
3. In `main`, open a Postgres pool when `DATABASE_URL` exists.
4. Run migrations on startup in development.
5. Pass app dependencies into the router through a small `Deps` struct.
6. Keep extraction working when no database is configured for tests.
7. Run `cd server && go test ./...`.

### Task 6: Basic Layout And Project Pages

**Files:**
- Create: `server/internal/web/templates/layout.templ`
- Create: `server/internal/web/templates/projects.templ`
- Create: `server/internal/web/project_handlers.go`
- Create: `server/internal/web/project_handlers_test.go`
- Modify: `server/internal/httpapi/router.go`

**Steps:**

1. Add templ dependency and generation setup.
2. Create minimal layout with project navigation.
3. Add `GET /` redirecting to `/projects`.
4. Add `GET /projects`, `GET /projects/new`, and `POST /projects`.
5. In local dev, use a deterministic dev user until auth lands.
6. Test project creation redirects to the created project page.
7. Run `cd server && go test ./...`.

### Task 7: Schedule And Item CRUD Pages

**Files:**
- Create: `server/internal/web/templates/project_detail.templ`
- Create: `server/internal/web/templates/schedule_detail.templ`
- Create: `server/internal/web/schedule_handlers.go`
- Create: `server/internal/web/item_handlers.go`
- Create: `server/internal/web/schedule_handlers_test.go`
- Create: `server/internal/web/item_handlers_test.go`

**Steps:**

1. Add project detail page listing schedules.
2. Add schedule creation.
3. Add schedule detail page listing items.
4. Add item creation and editing forms.
5. Preserve editable fields from current `ScheduleItem` shape.
6. Test project -> schedule -> item creation end to end through handlers.
7. Run `cd server && go test ./...`.

### Task 8: Project Share Links

**Files:**
- Create: `server/internal/share/tokens.go`
- Create: `server/internal/share/tokens_test.go`
- Create: `server/internal/web/share_handlers.go`
- Create: `server/internal/web/templates/share_project.templ`
- Create: `server/internal/web/share_handlers_test.go`

**Steps:**

1. Generate high-entropy opaque tokens.
2. Store only token hashes.
3. Add project share-link management page.
4. Add public `GET /share/{token}` page.
5. Public page shows schedules and items without edit controls.
6. Test invalid, inactive, and valid share tokens.
7. Run `cd server && go test ./...`.

### Task 9: Print-Friendly Views

**Files:**
- Modify: `server/internal/web/templates/share_project.templ`
- Modify: `server/internal/web/templates/schedule_detail.templ`
- Create: `server/internal/web/static/app.css`

**Steps:**

1. Add restrained, table-first CSS.
2. Add print CSS for schedule and share views.
3. Keep typography bold and legible.
4. Test static CSS route.
5. Manually print-preview the sample pages.

### Task 10: Verification And Docs

**Files:**
- Modify: `README.md`
- Modify: `server/README.md`
- Modify: `docs/manual-verification.md`

**Steps:**

1. Run `npm test`.
2. Run `npm run build`.
3. Run `cd server && go test ./...`.
4. Run `docker compose up --build`.
5. Verify `GET /healthz`.
6. Verify `GET /projects`.
7. Verify project/schedule/item creation.
8. Verify `POST /api/v1/extract-spec`.
9. Verify project share link renders as guest.
10. Update docs with the final local workflow.


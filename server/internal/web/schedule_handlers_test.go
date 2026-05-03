package web

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	appdb "sally/server/internal/db"
	queries "sally/server/internal/db/generated"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestSchedulePagesCreateAndShowSchedule(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	if err := appdb.RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	q := queries.New(conn)
	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "schedule-pages-test@example.com",
		Name:  "Schedule Pages Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Schedule Test Project " + time.Now().Format("150405.000000"),
		Address:     "24 School St.",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "schedule-pages-test@example.com",
		DevUserName:  "Schedule Pages Test",
	})

	form := url.Values{}
	form.Set("name", "Bath")

	createReq := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/schedules", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createResp := httptest.NewRecorder()

	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusSeeOther {
		t.Fatalf("expected create to redirect with 303, got %d", createResp.Code)
	}
	location := createResp.Header().Get("Location")
	if !strings.HasPrefix(location, "/projects/"+project.ID+"/schedules/") {
		t.Fatalf("expected redirect to schedule detail, got %q", location)
	}

	showReq := httptest.NewRequest(http.MethodGet, location, nil)
	showResp := httptest.NewRecorder()

	router.ServeHTTP(showResp, showReq)

	if showResp.Code != http.StatusOK {
		t.Fatalf("expected show status 200, got %d", showResp.Code)
	}
	body := showResp.Body.String()
	if !strings.Contains(body, "Bath") || !strings.Contains(body, "No items yet.") {
		t.Fatalf("expected schedule detail with empty item state, got %s", body)
	}
}

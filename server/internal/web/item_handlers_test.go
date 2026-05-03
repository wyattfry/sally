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

func TestItemPagesCreateAndListItem(t *testing.T) {
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
		Email: "item-pages-test@example.com",
		Name:  "Item Pages Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Item Test Project " + time.Now().Format("150405.000000"),
		Address:     "24 School St.",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	schedule, err := q.CreateSchedule(context.Background(), queries.CreateScheduleParams{
		ProjectID: project.ID,
		Name:      "Bath",
		Position:  1,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "item-pages-test@example.com",
		DevUserName:  "Item Pages Test",
	})

	form := url.Values{}
	form.Set("code", "B-01")
	form.Set("title", "Wall Faucet")
	form.Set("description", "Wall-mounted faucet with rough valve.")
	form.Set("manufacturer", "Example Co.")
	form.Set("model_number", "WF-200")
	form.Set("finish", "Polished Chrome")
	form.Set("notes", "Verify rough-in.")
	form.Set("source_url", "https://example.com/products/wf-200")

	path := "/projects/" + project.ID + "/schedules/" + schedule.ID + "/items"
	createReq := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createResp := httptest.NewRecorder()

	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusSeeOther {
		t.Fatalf("expected create to redirect with 303, got %d", createResp.Code)
	}
	location := createResp.Header().Get("Location")
	expectedLocation := "/projects/" + project.ID + "/schedules/" + schedule.ID
	if location != expectedLocation {
		t.Fatalf("expected redirect to %q, got %q", expectedLocation, location)
	}

	showReq := httptest.NewRequest(http.MethodGet, location, nil)
	showResp := httptest.NewRecorder()

	router.ServeHTTP(showResp, showReq)

	if showResp.Code != http.StatusOK {
		t.Fatalf("expected show status 200, got %d", showResp.Code)
	}
	body := showResp.Body.String()
	for _, expected := range []string{"B-01", "Wall Faucet", "Example Co.", "Polished Chrome", "Verify rough-in."} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected schedule detail to include %q, got %s", expected, body)
		}
	}
}

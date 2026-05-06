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

func newColumnTestRouter(t *testing.T) (*queries.Queries, http.Handler) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := appdb.RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	q := queries.New(conn)
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "column-test@example.com",
		DevUserName:  "Column Test",
	})
	return q, router
}

func createColumnTestFixture(t *testing.T, q *queries.Queries) (queries.Project, queries.Schedule, []queries.ScheduleColumn) {
	t.Helper()
	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "column-test@example.com",
		Name:  "Column Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Column Test Project " + time.Now().Format("150405.000000"),
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	schedule, err := q.CreateSchedule(context.Background(), queries.CreateScheduleParams{
		ProjectID: project.ID,
		Name:      "Test Schedule",
		Kind:      "items",
		Position:  1,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	cols := []struct{ key, label string }{
		{"alpha", "Alpha"},
		{"beta", "Beta"},
		{"gamma", "Gamma"},
	}
	var created []queries.ScheduleColumn
	for i, c := range cols {
		col, err := q.CreateScheduleColumn(context.Background(), queries.CreateScheduleColumnParams{
			ScheduleID: schedule.ID,
			Key:        c.key,
			Label:      c.label,
			Kind:       "text",
			Position:   int32(i + 1),
		})
		if err != nil {
			t.Fatalf("create column %s: %v", c.key, err)
		}
		created = append(created, col)
	}
	return project, schedule, created
}

func TestReorderScheduleColumns(t *testing.T) {
	q, router := newColumnTestRouter(t)
	project, schedule, cols := createColumnTestFixture(t, q)

	// Reverse order: gamma, beta, alpha
	form := url.Values{}
	form.Add("ids", cols[2].ID) // gamma → position 1
	form.Add("ids", cols[1].ID) // beta  → position 2
	form.Add("ids", cols[0].ID) // alpha → position 3

	req := httptest.NewRequest(http.MethodPost,
		"/projects/"+project.ID+"/schedules/"+schedule.ID+"/columns/reorder",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", resp.Code, resp.Body.String())
	}

	updated, err := q.ListScheduleColumns(context.Background(), schedule.ID)
	if err != nil {
		t.Fatalf("list columns: %v", err)
	}
	if len(updated) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(updated))
	}
	want := []string{"gamma", "beta", "alpha"}
	for i, col := range updated {
		if col.Key != want[i] {
			t.Errorf("position %d: expected key %q, got %q", i+1, want[i], col.Key)
		}
	}
}

func TestAddColumnAppearsInList(t *testing.T) {
	q, router := newColumnTestRouter(t)
	project, schedule, _ := createColumnTestFixture(t, q)

	form := url.Values{}
	form.Set("label", "Delta")
	req := httptest.NewRequest(http.MethodPost,
		"/projects/"+project.ID+"/schedules/"+schedule.ID+"/columns",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 fragment response, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "Delta") {
		t.Errorf("expected 'Delta' in fragment, got: %s", body)
	}
	if !strings.Contains(body, "col-modal-move") {
		t.Errorf("expected move buttons in fragment, got: %s", body)
	}
}

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
	if !strings.HasPrefix(location, "/projects/"+project.ID+"#schedule-") {
		t.Fatalf("expected redirect to project page with schedule anchor, got %q", location)
	}

	// Fragment is stripped by the router; GET the project page directly.
	showReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID, nil)
	showResp := httptest.NewRecorder()

	router.ServeHTTP(showResp, showReq)

	if showResp.Code != http.StatusOK {
		t.Fatalf("expected show status 200, got %d", showResp.Code)
	}
	body := showResp.Body.String()
	if !strings.Contains(body, "Bath") || !strings.Contains(body, "No items yet.") {
		t.Fatalf("expected project detail with new schedule, got %s", body)
	}
}

func TestEditSchedulePageShowsFullBreadcrumb(t *testing.T) {
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
		Email: "schedule-breadcrumb-test@example.com",
		Name:  "Schedule Breadcrumb Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Breadcrumb Project " + time.Now().Format("150405.000000"),
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
		DevUserEmail: "schedule-breadcrumb-test@example.com",
		DevUserName:  "Schedule Breadcrumb Test",
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID+"/schedules/"+schedule.ID+"/edit", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, project.Name) {
		t.Fatalf("expected edit-schedule breadcrumb to include project name %q, got:\n%s", project.Name, body)
	}
	if !strings.Contains(body, "Bath") {
		t.Fatalf("expected edit-schedule breadcrumb to include schedule name, got:\n%s", body)
	}
	if !strings.Contains(body, `/projects"`) {
		t.Fatalf("expected edit-schedule breadcrumb to include /projects link, got:\n%s", body)
	}
}

func TestSchedulePagesUpdateAndDeleteSchedule(t *testing.T) {
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
		Email: "schedule-update-test@example.com",
		Name:  "Schedule Update Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Schedule Update Project " + time.Now().Format("150405.000000"),
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
		DevUserEmail: "schedule-update-test@example.com",
		DevUserName:  "Schedule Update Test",
	})

	form := url.Values{}
	form.Set("name", "Primary Bath")
	form.Set("position", "2")
	updateReq := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/schedules/"+schedule.ID+"/edit", strings.NewReader(form.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)

	if updateResp.Code != http.StatusSeeOther {
		t.Fatalf("expected update to redirect with 303, got %d", updateResp.Code)
	}

	projectReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID, nil)
	projectResp := httptest.NewRecorder()
	router.ServeHTTP(projectResp, projectReq)
	if !strings.Contains(projectResp.Body.String(), "Primary Bath") {
		t.Fatalf("expected project page to contain updated schedule name, got %s", projectResp.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/schedules/"+schedule.ID+"/delete", nil)
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusSeeOther || deleteResp.Header().Get("Location") != "/projects/"+project.ID {
		t.Fatalf("expected delete redirect to project, got status=%d location=%q", deleteResp.Code, deleteResp.Header().Get("Location"))
	}

	deletedShowReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID+"/schedules/"+schedule.ID, nil)
	deletedResp := httptest.NewRecorder()
	router.ServeHTTP(deletedResp, deletedShowReq)
	if deletedResp.Code != http.StatusNotFound {
		t.Fatalf("expected deleted schedule to return 404, got %d", deletedResp.Code)
	}
}

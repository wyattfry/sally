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

func TestProjectPagesCreateAndShowProject(t *testing.T) {
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

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      queries.New(conn),
		DevUserEmail: "project-pages-test@example.com",
		DevUserName:  "Project Pages Test",
	})

	form := url.Values{}
	form.Set("name", "Project Pages "+time.Now().Format("150405.000000"))
	form.Set("address", "24 School St. Metuchen NJ")

	createReq := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createResp := httptest.NewRecorder()

	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusSeeOther {
		t.Fatalf("expected create to redirect with 303, got %d", createResp.Code)
	}
	location := createResp.Header().Get("Location")
	if !strings.HasPrefix(location, "/projects/") {
		t.Fatalf("expected redirect to project detail, got %q", location)
	}

	showReq := httptest.NewRequest(http.MethodGet, location, nil)
	showResp := httptest.NewRecorder()

	router.ServeHTTP(showResp, showReq)

	if showResp.Code != http.StatusOK {
		t.Fatalf("expected show status 200, got %d", showResp.Code)
	}
	if !strings.Contains(showResp.Body.String(), "24 School St. Metuchen NJ") {
		t.Fatalf("expected response to include project address, got %s", showResp.Body.String())
	}
}

func TestProjectIndexListsProjects(t *testing.T) {
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
		Email: "project-index-test@example.com",
		Name:  "Project Index Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Listed Project " + time.Now().Format("150405.000000"),
		Address:     "307 W38th St.",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "project-index-test@example.com",
		DevUserName:  "Project Index Test",
	})

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), project.Name) {
		t.Fatalf("expected response to include project name, got %s", resp.Body.String())
	}
}

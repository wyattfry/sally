package web

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	appdb "sally/server/internal/db"
	queries "sally/server/internal/db/generated"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestProjectShareLinkRendersPublicProject(t *testing.T) {
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
		Email: "share-pages-test@example.com",
		Name:  "Share Pages Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Share Test Project " + time.Now().Format("150405.000000"),
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
	_, err = q.CreateScheduleItem(context.Background(), queries.CreateScheduleItemParams{
		ScheduleID:        schedule.ID,
		Code:              "B-01",
		Title:             "Wall Faucet",
		Description:       "Wall-mounted faucet.",
		Manufacturer:      "Example Co.",
		ModelNumber:       "WF-200",
		Finish:            "Polished Chrome",
		FinishModelNumber: "",
		Notes:             "Verify rough-in.",
		SourceUrl:         "https://example.com/products/wf-200",
		SourceTitle:       "",
		SourceImageUrl:    "",
		SourcePdfLinks:    []string{},
		Position:          1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "share-pages-test@example.com",
		DevUserName:  "Share Pages Test",
	})

	createReq := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/share-links", nil)
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusSeeOther {
		t.Fatalf("expected share link create to redirect with 303, got %d", createResp.Code)
	}
	location := createResp.Header().Get("Location")
	if !strings.HasPrefix(location, "/projects/"+project.ID+"/share?token=") {
		t.Fatalf("expected redirect with share token, got %q", location)
	}
	token := strings.TrimPrefix(location, "/projects/"+project.ID+"/share?token=")

	publicReq := httptest.NewRequest(http.MethodGet, "/share/"+token, nil)
	publicResp := httptest.NewRecorder()
	router.ServeHTTP(publicResp, publicReq)

	if publicResp.Code != http.StatusOK {
		t.Fatalf("expected public share status 200, got %d", publicResp.Code)
	}
	body := publicResp.Body.String()
	for _, expected := range []string{project.Name, "Bath", "Wall Faucet", "Example Co.", "https://example.com/products/wf-200"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected public share page to include %q, got %s", expected, body)
		}
	}
	if strings.Contains(body, "Add Item") || strings.Contains(body, "New Schedule") {
		t.Fatalf("expected public share page to omit edit controls, got %s", body)
	}
}

func TestInvalidProjectShareTokenReturnsNotFound(t *testing.T) {
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
	RegisterRoutes(router, Deps{Queries: queries.New(conn)})

	req := httptest.NewRequest(http.MethodGet, "/share/not-a-real-token", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected invalid token to return 404, got %d", resp.Code)
	}
}

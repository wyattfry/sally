package web

import (
	"context"
	"database/sql"
	"encoding/json"
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

func newShareTestRouter(t *testing.T) (*queries.Queries, http.Handler) {
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
		DevUserEmail: "share-test@example.com",
		DevUserName:  "Share Test",
	})
	return q, router
}

func createShareTestProject(t *testing.T, q *queries.Queries) queries.Project {
	t.Helper()
	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "share-test@example.com",
		Name:  "Share Test",
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
	return project
}

// extractShareToken finds the token from the project page's actions-menu
// data-share-url="http(s)://host/share/TOKEN" attribute.
func extractShareToken(t *testing.T, body string) string {
	t.Helper()
	const needle = "/share/"
	idx := strings.Index(body, needle)
	if idx == -1 {
		t.Fatalf("could not find share URL in page body:\n%s", body)
	}
	rest := body[idx+len(needle):]
	end := strings.IndexAny(rest, `"'& `)
	if end == -1 {
		end = len(rest)
	}
	token := rest[:end]
	if token == "" {
		t.Fatalf("extracted empty token from page body:\n%s", body)
	}
	return token
}

// loadProjectPage renders the architect view of a project (which auto-
// creates a share link on first view) and returns the response body.
func loadProjectPage(t *testing.T, router http.Handler, projectID string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected project page 200, got %d", resp.Code)
	}
	return resp.Body.String()
}

func TestProjectShareLinkRendersPublicProject(t *testing.T) {
	q, router := newShareTestRouter(t)
	project := createShareTestProject(t, q)

	schedule, err := q.CreateSchedule(context.Background(), queries.CreateScheduleParams{
		ProjectID: project.ID,
		Name:      "Bath",
		Kind:      "items",
		Position:  1,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	if err := seedColumns(context.Background(), q, schedule.ID, "general"); err != nil {
		t.Fatalf("seed columns: %v", err)
	}
	itemData, _ := json.Marshal(map[string]string{
		"code":         "B-01",
		"title":        "Wall Faucet",
		"description":  "Wall-mounted faucet.",
		"manufacturer": "Example Co.",
		"model_number": "WF-200",
		"finish":       "Polished Chrome",
		"notes":        "Verify rough-in.",
	})
	_, err = q.CreateScheduleItem(context.Background(), queries.CreateScheduleItemParams{
		ScheduleID:      schedule.ID,
		Data:            itemData,
		SourceImageUrls: []string{},
		SourcePdfLinks:  []string{},
		Position:        1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	// First load auto-creates the share link.
	token := extractShareToken(t, loadProjectPage(t, router, project.ID))

	publicReq := httptest.NewRequest(http.MethodGet, "/share/"+token, nil)
	publicResp := httptest.NewRecorder()
	router.ServeHTTP(publicResp, publicReq)

	if publicResp.Code != http.StatusOK {
		t.Fatalf("expected public share status 200, got %d", publicResp.Code)
	}
	body := publicResp.Body.String()
	for _, expected := range []string{project.Name, "Bath"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected public share page to include %q, got %s", expected, body)
		}
	}
	if strings.Contains(body, "Add Item") || strings.Contains(body, "New Schedule") {
		t.Fatalf("expected public share page to omit edit controls, got %s", body)
	}
}

func TestProjectPageReturnsSameTokenAcrossLoads(t *testing.T) {
	q, router := newShareTestRouter(t)
	project := createShareTestProject(t, q)

	token1 := extractShareToken(t, loadProjectPage(t, router, project.ID))
	token2 := extractShareToken(t, loadProjectPage(t, router, project.ID))
	if token1 != token2 {
		t.Fatalf("expected token to be stable across project page loads, got %q then %q", token1, token2)
	}
}

func TestInvalidProjectShareTokenReturnsNotFound(t *testing.T) {
	_, router := newShareTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/share/not-a-real-token", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected invalid token to return 404, got %d", resp.Code)
	}
}

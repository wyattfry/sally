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

func TestProjectShareLinkRendersPublicProject(t *testing.T) {
	q, router := newShareTestRouter(t)
	project := createShareTestProject(t, q)

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
	_, router := newShareTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/share/not-a-real-token", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected invalid token to return 404, got %d", resp.Code)
	}
}

func TestShareManagePage_NoActiveLink(t *testing.T) {
	q, router := newShareTestRouter(t)
	project := createShareTestProject(t, q)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID+"/share", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "Enable sharing") {
		t.Fatalf("expected 'Enable sharing' button when no active link, got:\n%s", body)
	}
	if strings.Contains(body, "Disable sharing") {
		t.Fatalf("expected no 'Disable sharing' when no active link, got:\n%s", body)
	}
}

func TestShareManagePage_ShowsFullURLAfterCreation(t *testing.T) {
	q, router := newShareTestRouter(t)
	project := createShareTestProject(t, q)

	createReq := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/share-links", nil)
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)

	location := createResp.Header().Get("Location")
	token := strings.TrimPrefix(location, "/projects/"+project.ID+"/share?token=")

	pageReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID+"/share?token="+token, nil)
	pageReq.Host = "sally.example.com"
	pageResp := httptest.NewRecorder()
	router.ServeHTTP(pageResp, pageReq)

	if pageResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", pageResp.Code)
	}
	body := pageResp.Body.String()
	expectedURL := "http://sally.example.com/share/" + token
	if !strings.Contains(body, expectedURL) {
		t.Fatalf("expected full share URL %q in page, got:\n%s", expectedURL, body)
	}
	if !strings.Contains(body, "Copy") {
		t.Fatalf("expected Copy button, got:\n%s", body)
	}
	if !strings.Contains(body, "Disable sharing") {
		t.Fatalf("expected 'Disable sharing' button, got:\n%s", body)
	}
}

func TestShareManagePage_DeactivateRemovesLink(t *testing.T) {
	q, router := newShareTestRouter(t)
	project := createShareTestProject(t, q)

	// Enable sharing
	createReq := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/share-links", nil)
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 from create, got %d", createResp.Code)
	}

	location := createResp.Header().Get("Location")
	token := strings.TrimPrefix(location, "/projects/"+project.ID+"/share?token=")

	// Confirm the public link works
	publicReq := httptest.NewRequest(http.MethodGet, "/share/"+token, nil)
	publicResp := httptest.NewRecorder()
	router.ServeHTTP(publicResp, publicReq)
	if publicResp.Code != http.StatusOK {
		t.Fatalf("expected public share to be accessible before disable, got %d", publicResp.Code)
	}

	// Disable sharing
	disableReq := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/share-links/deactivate", nil)
	disableResp := httptest.NewRecorder()
	router.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 from deactivate, got %d", disableResp.Code)
	}

	// Confirm the public link no longer works
	publicReq2 := httptest.NewRequest(http.MethodGet, "/share/"+token, nil)
	publicResp2 := httptest.NewRecorder()
	router.ServeHTTP(publicResp2, publicReq2)
	if publicResp2.Code != http.StatusNotFound {
		t.Fatalf("expected public share to return 404 after disable, got %d", publicResp2.Code)
	}

	// Confirm the manage page shows Enable button
	pageReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID+"/share", nil)
	pageResp := httptest.NewRecorder()
	router.ServeHTTP(pageResp, pageReq)
	body := pageResp.Body.String()
	if !strings.Contains(body, "Enable sharing") {
		t.Fatalf("expected 'Enable sharing' after disable, got:\n%s", body)
	}
}

func TestShareManagePage_ReEnableReplacesOldLink(t *testing.T) {
	q, router := newShareTestRouter(t)
	project := createShareTestProject(t, q)

	// First enable
	r1 := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/share-links", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	token1 := strings.TrimPrefix(w1.Header().Get("Location"), "/projects/"+project.ID+"/share?token=")

	// Re-enable (replaces old token)
	r2 := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/share-links", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	token2 := strings.TrimPrefix(w2.Header().Get("Location"), "/projects/"+project.ID+"/share?token=")

	if token1 == token2 {
		t.Fatal("expected new token to differ from old token after re-enable")
	}

	// Old token should be gone
	r3 := httptest.NewRequest(http.MethodGet, "/share/"+token1, nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)
	if w3.Code != http.StatusNotFound {
		t.Fatalf("expected old token to return 404 after re-enable, got %d", w3.Code)
	}

	// New token should work
	r4 := httptest.NewRequest(http.MethodGet, "/share/"+token2, nil)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, r4)
	if w4.Code != http.StatusOK {
		t.Fatalf("expected new token to return 200, got %d", w4.Code)
	}
}

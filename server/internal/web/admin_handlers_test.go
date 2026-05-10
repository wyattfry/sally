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
	"sally/server/internal/share"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/oauth2"
)

func TestAdminDashboardRendersChartSeriesAsJavaScriptArrays(t *testing.T) {
	resp := httptest.NewRecorder()

	render(resp, adminPage{
		Kind:  "admin",
		Title: "Admin",
		ItemHourlyJSON: mustJSON([]queries.DailyPoint{
			{Date: "05-10 16:00", Count: 2},
		}),
		ItemDailyJSON: mustJSON([]queries.DailyPoint{
			{Date: "2026-05-10", Count: 3},
		}),
		ExtractHourlyJSON: mustJSON([]queries.DailyPoint{
			{Date: "05-10 16:00", Count: 4, Extra: 1},
		}),
		ExtractDailyJSON: mustJSON([]queries.DailyPoint{
			{Date: "2026-05-10", Count: 5, Extra: 2},
		}),
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, want := range []string{
		`var itemHourly  = [{"Date":"05-10 16:00","Count":2,"Extra":0}];`,
		`var itemDaily   = [{"Date":"2026-05-10","Count":3,"Extra":0}];`,
		`var exHourly    = [{"Date":"05-10 16:00","Count":4,"Extra":1}];`,
		`var exDaily     = [{"Date":"2026-05-10","Count":5,"Extra":2}];`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected chart JSON to render as a JavaScript array containing %q, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, `var itemHourly  = "[`) {
		t.Fatalf("chart JSON rendered as a quoted string")
	}
}

func newAdminTestSetup(t *testing.T) (*queries.Queries, *sql.DB, http.Handler) {
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
	// Dev mode (no OAuthConfig) + DB wired so requireAdmin passes.
	RegisterRoutes(router, Deps{
		Queries: q,
		DB:      conn,
	})
	return q, conn, router
}

func TestAdminExtractionLogsJSONAcceptsAPIToken(t *testing.T) {
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
	email := "admin-api-token-" + time.Now().Format("150405000") + "@example.com"
	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: email,
		Name:  "Admin API Token",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	rawToken := "admin-api-token-test-" + time.Now().Format("150405000")
	if _, err := q.CreateAPIToken(context.Background(), user.ID, "notebook", share.HashToken(rawToken)); err != nil {
		t.Fatalf("create api token: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries: q,
		DB:      conn,
		OAuthConfig: &oauth2.Config{
			ClientID:    "test-client",
			RedirectURL: "http://localhost/auth/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.test/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.test/token",
			},
		},
		SessionSecret: []byte("test-secret"),
		AdminEmail:    email,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/api/extraction-logs", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected JSON response, got content-type %q and body %s", got, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), `"logs"`) {
		t.Fatalf("expected logs payload, got %s", resp.Body.String())
	}
}

func TestAdminCreateUser_CreatesUserAndReturnsLoginURL(t *testing.T) {
	_, _, router := newAdminTestSetup(t)

	email := "admin-create-user-" + time.Now().Format("150405000") + "@example.com"
	form := url.Values{}
	form.Set("email", email)
	form.Set("name", "Admin Created User")

	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d: %s", resp.Code, resp.Body.String())
	}
	loc := resp.Header().Get("Location")
	if !strings.Contains(loc, "login_url=") {
		t.Errorf("expected login_url in redirect, got %q", loc)
	}
	if !strings.Contains(loc, "/auth/token?t=") {
		t.Errorf("expected /auth/token in login_url, got %q", loc)
	}
}

func TestAdminCreateUser_RequiresEmail(t *testing.T) {
	_, _, router := newAdminTestSetup(t)

	form := url.Values{}
	form.Set("name", "No Email User")

	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when email is missing, got %d", resp.Code)
	}
}

func TestAdminCreateLoginLink_ReturnsLoginURL(t *testing.T) {
	q, _, router := newAdminTestSetup(t)

	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "admin-link-user-" + time.Now().Format("150405000") + "@example.com",
		Name:  "Link User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+user.ID+"/login-link", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d: %s", resp.Code, resp.Body.String())
	}
	loc := resp.Header().Get("Location")
	if !strings.Contains(loc, "login_url=") {
		t.Errorf("expected login_url in redirect, got %q", loc)
	}
	if !strings.Contains(loc, "/auth/token?t=") {
		t.Errorf("expected /auth/token in login_url, got %q", loc)
	}
}

func TestAdminCreateLoginLink_UnknownUser(t *testing.T) {
	_, _, router := newAdminTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/00000000-0000-0000-0000-000000000000/login-link", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown user, got %d", resp.Code)
	}
}

func TestAdminUsersPageShowsNewLoginURL(t *testing.T) {
	_, _, router := newAdminTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/users?login_url=http%3A%2F%2Flocalhost%2Fauth%2Ftoken%3Ft%3Dabc&for=test%40example.com", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "http://localhost/auth/token?t=abc") {
		t.Errorf("expected login URL rendered in page, got:\n%s", body)
	}
	if !strings.Contains(body, "test@example.com") {
		t.Errorf("expected 'for' name in page, got:\n%s", body)
	}
}

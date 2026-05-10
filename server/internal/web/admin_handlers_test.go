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

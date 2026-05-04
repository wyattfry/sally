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

	"golang.org/x/oauth2"
	appdb "sally/server/internal/db"
	queries "sally/server/internal/db/generated"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// --- Session helper unit tests (no HTTP, no DB) ---

func TestSessionCookieRoundTrip(t *testing.T) {
	secret := []byte("test-secret")
	email := "user@example.com"

	signed := signedCookieValue(secret, email)
	got, ok := verifySignedCookieValue(secret, signed)
	if !ok {
		t.Fatal("expected verification to succeed")
	}
	if got != email {
		t.Fatalf("expected %q, got %q", email, got)
	}
}

func TestSessionCookieRejectsTamperedValue(t *testing.T) {
	secret := []byte("test-secret")
	signed := signedCookieValue(secret, "user@example.com")

	_, ok := verifySignedCookieValue(secret, signed+"x")
	if ok {
		t.Fatal("expected tampered value to fail verification")
	}
}

func TestSessionCookieRejectsWrongSecret(t *testing.T) {
	signed := signedCookieValue([]byte("secret-a"), "user@example.com")
	_, ok := verifySignedCookieValue([]byte("secret-b"), signed)
	if ok {
		t.Fatal("expected wrong secret to fail verification")
	}
}

// --- Auth handler tests ---

func TestLoginRedirectsToProjectsInDevMode(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{}) // no OAuthConfig → dev mode

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", resp.Code)
	}
	if resp.Header().Get("Location") != "/projects" {
		t.Fatalf("expected redirect to /projects, got %q", resp.Header().Get("Location"))
	}
}

func TestLoginRendersSignInPageWhenOAuthConfigured(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		OAuthConfig: &oauth2.Config{
			ClientID:    "test-client",
			RedirectURL: "http://localhost/auth/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.test/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.test/token",
			},
		},
		SessionSecret: []byte("test-secret"),
	})

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "/auth/google") {
		t.Fatalf("expected sign-in page with /auth/google link, got:\n%s", resp.Body.String())
	}
}

func TestLoginRedirectsToProjectsWhenAlreadySignedIn(t *testing.T) {
	secret := []byte("test-secret")
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		OAuthConfig: &oauth2.Config{
			ClientID:    "test-client",
			RedirectURL: "http://localhost/auth/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.test/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.test/token",
			},
		},
		SessionSecret: secret,
	})

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: signedCookieValue(secret, "already@signed.in"),
	})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", resp.Code)
	}
	if resp.Header().Get("Location") != "/projects" {
		t.Fatalf("expected redirect to /projects, got %q", resp.Header().Get("Location"))
	}
}

func TestStartGoogleOAuthRedirectsToProvider(t *testing.T) {
	authEndpoint := "https://accounts.google.test/o/oauth2/auth"
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		OAuthConfig: &oauth2.Config{
			ClientID:    "test-client",
			RedirectURL: "http://localhost/auth/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  authEndpoint,
				TokenURL: "https://oauth2.googleapis.test/token",
			},
		},
		SessionSecret: []byte("test-secret"),
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", resp.Code)
	}
	if !strings.HasPrefix(resp.Header().Get("Location"), authEndpoint) {
		t.Fatalf("expected redirect to Google auth endpoint, got %q", resp.Header().Get("Location"))
	}
	// State cookie must be set
	var hasStateCookie bool
	for _, c := range resp.Result().Cookies() {
		if c.Name == oauthStateCookieName {
			hasStateCookie = true
		}
	}
	if !hasStateCookie {
		t.Fatal("expected oauth state cookie to be set")
	}
}

func TestOAuthCallbackRejectsMissingState(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		OAuthConfig: &oauth2.Config{
			ClientID: "test-client",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.test/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.test/token",
			},
		},
		SessionSecret: []byte("test-secret"),
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state=wrong", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing state cookie, got %d", resp.Code)
	}
}

func TestOAuthCallbackRejectsMismatchedState(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		OAuthConfig: &oauth2.Config{
			ClientID: "test-client",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.test/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.test/token",
			},
		},
		SessionSecret: []byte("test-secret"),
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state=state-from-query", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: "different-state"})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for mismatched state, got %d", resp.Code)
	}
}

func TestLogoutClearsSessionAndRedirects(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{SessionSecret: []byte("test-secret")})

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: signedCookieValue([]byte("test-secret"), "user@example.com"),
	})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", resp.Code)
	}
	if resp.Header().Get("Location") != "/login" {
		t.Fatalf("expected redirect to /login, got %q", resp.Header().Get("Location"))
	}
	var cleared bool
	for _, c := range resp.Result().Cookies() {
		if c.Name == sessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected session cookie to be cleared")
	}
}

// --- requireUser DB tests ---

func TestRequireUserDevModeUsesDevCredentials(t *testing.T) {
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
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "dev-require-test@example.com",
		DevUserName:  "Dev Require Test",
	})

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 in dev mode, got %d", resp.Code)
	}
}

func TestRequireUserOAuthModeRedirectsWithoutSession(t *testing.T) {
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
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries: q,
		OAuthConfig: &oauth2.Config{
			ClientID: "test",
			Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.test/auth", TokenURL: "https://tok"},
		},
		SessionSecret: []byte("test-secret"),
	})

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther || resp.Header().Get("Location") != "/login" {
		t.Fatalf("expected redirect to /login, got status=%d location=%q", resp.Code, resp.Header().Get("Location"))
	}
}

func TestRequireUserOAuthModeAcceptsValidSession(t *testing.T) {
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
	email := "oauth-session-test-" + time.Now().Format("150405000") + "@example.com"
	if _, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: email,
		Name:  "OAuth Session Test",
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	secret := []byte("test-secret")
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries: q,
		OAuthConfig: &oauth2.Config{
			ClientID: "test",
			Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.test/auth", TokenURL: "https://tok"},
		},
		SessionSecret: secret,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: signedCookieValue(secret, email),
	})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid session, got %d", resp.Code)
	}
}

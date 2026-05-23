package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMarketingPagesRenderWithoutAuth(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{})

	tests := []struct {
		path     string
		expected string
	}{
		{path: "/press", expected: "Press"},
		{path: "/privacy", expected: "Privacy Policy"},
		{path: "/contact", expected: "Contact"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", resp.Code)
			}
			if !strings.Contains(resp.Body.String(), tt.expected) {
				t.Fatalf("expected page to contain %q, got:\n%s", tt.expected, resp.Body.String())
			}
		})
	}
}

func TestFooterLinksMarketingPages(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{})

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	body := resp.Body.String()
	for _, href := range []string{`href="/press"`, `href="/privacy"`, `href="/contact"`} {
		if !strings.Contains(body, href) {
			t.Fatalf("expected footer to contain %s, got:\n%s", href, body)
		}
	}
}

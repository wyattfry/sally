package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStaticCSSRouteServesAppStyles(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{})

	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "@media print") {
		t.Fatalf("expected print styles in css, got %s", resp.Body.String())
	}
}

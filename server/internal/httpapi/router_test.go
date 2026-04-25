package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sally/server/internal/config"
)

func TestRouterHealthzReturnsOK(t *testing.T) {
	router := NewRouter(config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestRouterExtractSpecReturnsBadRequestForMissingBody(t *testing.T) {
	router := NewRouter(config.Config{})

	req := httptest.NewRequest(http.MethodPost, "/v1/extract-spec", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRouterExtractSpecHandlesCORSPreflight(t *testing.T) {
	router := NewRouter(config.Config{})

	req := httptest.NewRequest(http.MethodOptions, "/v1/extract-spec", nil)
	req.Header.Set("Origin", "https://www.voltagerestaurantsupply.com")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://www.voltagerestaurantsupply.com" {
		t.Fatalf("expected allow-origin header, got %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "content-type" {
		t.Fatalf("expected allow-headers header, got %q", got)
	}
}

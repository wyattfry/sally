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

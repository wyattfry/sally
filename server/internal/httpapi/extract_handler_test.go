package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sally/server/internal/config"
	"sally/server/internal/extract"
)

func TestExtractSpecHandlerReturnsOKForValidRequest(t *testing.T) {
	router := NewRouter(config.Config{})

	body := `{
		"requestId": "123e4567-e89b-12d3-a456-426614174000",
		"sentAt": "2026-04-24T18:30:00Z",
		"client": {
			"app": "sally-extension",
			"version": "0.1.0",
			"chromeVersion": "136.0.0.0"
		},
		"page": {
			"title": "Example Co. WF-200 Wall Faucet",
			"url": "https://example.com/products/wf-200",
			"visibleText": "Wall-mounted faucet rough-in dimensions and installation notes.",
			"mainImageUrl": "https://example.com/faucet.jpg",
			"structuredData": [],
			"pdfLinks": ["https://example.com/spec-sheet.pdf"]
		},
		"projectContext": {
			"projectName": "My New Project",
			"knownZones": ["Primary Bath"],
			"knownCategories": ["Plumbing Fixture"]
		},
		"options": {
			"includeDebug": true,
			"returnAlternatives": false
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/extract-spec", strings.NewReader(body))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp extract.ExtractSpecResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected ok status in body, got %q", resp.Status)
	}
	if resp.RequestID != "123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("expected same requestId in body, got %q", resp.RequestID)
	}
	if resp.Proposal == nil || resp.Proposal.Title == "" || resp.Proposal.SourceURL == "" {
		t.Fatalf("expected populated proposal in body, got %#v", resp.Proposal)
	}
	if resp.Meta.Provider == "" || resp.Meta.PromptVersion == "" {
		t.Fatalf("expected populated meta in body, got %#v", resp.Meta)
	}
}

func TestExtractSpecHandlerReturnsBadRequestForInvalidJSON(t *testing.T) {
	router := NewRouter(config.Config{})

	req := httptest.NewRequest(http.MethodPost, "/v1/extract-spec", strings.NewReader(`{`))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestExtractSpecHandlerReturnsBadRequestForMissingRequiredField(t *testing.T) {
	router := NewRouter(config.Config{})

	body := `{
		"requestId": "123e4567-e89b-12d3-a456-426614174000",
		"sentAt": "2026-04-24T18:30:00Z",
		"client": {
			"app": "sally-extension",
			"version": "0.1.0",
			"chromeVersion": "136.0.0.0"
		},
		"page": {
			"title": "",
			"url": "https://example.com/products/wf-200",
			"visibleText": "Wall-mounted faucet rough-in dimensions and installation notes.",
			"mainImageUrl": "https://example.com/faucet.jpg",
			"structuredData": [],
			"pdfLinks": ["https://example.com/spec-sheet.pdf"]
		},
		"projectContext": {
			"projectName": "My New Project",
			"knownZones": ["Primary Bath"],
			"knownCategories": ["Plumbing Fixture"]
		},
		"options": {
			"includeDebug": true,
			"returnAlternatives": false
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/extract-spec", strings.NewReader(body))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestExtractSpecHandlerReturnsMethodNotAllowedForWrongMethod(t *testing.T) {
	router := NewRouter(config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/v1/extract-spec", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

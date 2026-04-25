package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sally/server/internal/config"
	"sally/server/internal/extract"
)

// parseSSEData returns the JSON data payload of the first SSE event matching eventType.
func parseSSEData(body []byte, eventType string) ([]byte, error) {
	var currentEvent string
	for _, line := range strings.Split(string(body), "\n") {
		switch {
		case strings.HasPrefix(line, "event: "):
			currentEvent = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: ") && currentEvent == eventType:
			return []byte(strings.TrimPrefix(line, "data: ")), nil
		}
	}
	return nil, fmt.Errorf("no %q event in SSE body: %s", eventType, body)
}

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
	if ct := rr.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}

	data, err := parseSSEData(rr.Body.Bytes(), "done")
	if err != nil {
		t.Fatalf("parse SSE: %v", err)
	}
	var resp extract.ExtractSpecResponse
	if err := json.Unmarshal(data, &resp); err != nil {
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

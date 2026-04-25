package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sally/server/internal/extract"
	"sally/server/internal/httpapi"
	"sally/server/internal/provider"
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

func TestStubExtractorReturnsProposal(t *testing.T) {
	extractor := provider.NewStubExtractor()

	resp, err := extractor.Extract(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("extract returned error: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("expected ok status, got %q", resp.Status)
	}
	if resp.Proposal == nil {
		t.Fatal("expected proposal")
	}
	if resp.Proposal.Title != "Example Co. WF-200 Wall Faucet" {
		t.Fatalf("expected title from page payload, got %q", resp.Proposal.Title)
	}
	if resp.Meta.Provider != "stub" {
		t.Fatalf("expected stub provider meta, got %q", resp.Meta.Provider)
	}
}

func TestProviderSentinelErrorsMatchKinds(t *testing.T) {
	if !errors.Is(provider.ErrTimeout, provider.ErrTimeout) {
		t.Fatal("expected timeout sentinel to match itself")
	}
	if !errors.Is(provider.ErrFailure, provider.ErrFailure) {
		t.Fatal("expected failure sentinel to match itself")
	}
}

func TestExtractHandlerMapsProviderTimeoutToContractError(t *testing.T) {
	handler := httpapi.NewExtractHandler(&fakeExtractor{
		err: provider.ErrTimeout,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/extract-spec", strings.NewReader(validRequestJSON))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	data, err := parseSSEData(rr.Body.Bytes(), "error")
	if err != nil {
		t.Fatalf("parse SSE: %v", err)
	}
	var resp extract.ExtractSpecResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}

	if resp.Status != "error" {
		t.Fatalf("expected error status, got %q", resp.Status)
	}
	if resp.Error == nil || resp.Error.Code != "MODEL_TIMEOUT" {
		t.Fatalf("expected MODEL_TIMEOUT error payload, got %#v", resp.Error)
	}
	if resp.Meta.Provider != "fake-provider" || resp.Meta.PromptVersion != "fake-v1" {
		t.Fatalf("expected provider meta from extractor, got %#v", resp.Meta)
	}
}

func TestExtractHandlerMapsProviderFailureToContractError(t *testing.T) {
	handler := httpapi.NewExtractHandler(&fakeExtractor{
		err: errors.Join(provider.ErrFailure, errors.New("upstream status 400: bad request")),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/extract-spec", strings.NewReader(validRequestJSON))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	data, err := parseSSEData(rr.Body.Bytes(), "error")
	if err != nil {
		t.Fatalf("parse SSE: %v", err)
	}
	var resp extract.ExtractSpecResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}

	if resp.Status != "error" {
		t.Fatalf("expected error status, got %q", resp.Status)
	}
	if resp.Error == nil || resp.Error.Code != "PROVIDER_ERROR" {
		t.Fatalf("expected PROVIDER_ERROR payload, got %#v", resp.Error)
	}
	if !strings.Contains(resp.Error.Message, "upstream status 400") {
		t.Fatalf("expected detailed provider error message, got %#v", resp.Error)
	}
	if resp.Meta.Provider != "fake-provider" || resp.Meta.Model != "fake-model" {
		t.Fatalf("expected provider meta from extractor, got %#v", resp.Meta)
	}
}

func TestExtractHandlerPassesRequestContextToProvider(t *testing.T) {
	ctxKey := contextKey("request-id")
	extractor := fakeExtractor{
		resp: extract.ExtractSpecResponse{
			RequestID: "123e4567-e89b-12d3-a456-426614174000",
			Status:    "ok",
			Proposal: &extract.Proposal{
				Title:          "Wall Faucet",
				SourceURL:      "https://example.com/products/wf-200",
				SourceTitle:    "Example Co. WF-200 Wall Faucet",
				SourcePDFLinks: []string{"https://example.com/spec-sheet.pdf"},
			},
			Meta: extract.ResponseMeta{
				Provider:      "fake-provider",
				Model:         "fake-model",
				PromptVersion: "fake-v1",
			},
		},
	}
	handler := httpapi.NewExtractHandler(&extractor)

	req := httptest.NewRequest(http.MethodPost, "/v1/extract-spec", strings.NewReader(validRequestJSON))
	req = req.WithContext(context.WithValue(req.Context(), ctxKey, "ctx-value"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := extractor.gotCtx.Value(ctxKey); got != "ctx-value" {
		t.Fatalf("expected request context to reach provider, got %#v", got)
	}
}

type fakeExtractor struct {
	resp   extract.ExtractSpecResponse
	err    error
	gotCtx context.Context
}

func (f *fakeExtractor) Extract(ctx context.Context, _ extract.ExtractSpecRequest) (extract.ExtractSpecResponse, error) {
	f.gotCtx = ctx
	return f.resp, f.err
}

func (f fakeExtractor) Meta() extract.ResponseMeta {
	return extract.ResponseMeta{
		Provider:      "fake-provider",
		Model:         "fake-model",
		PromptVersion: "fake-v1",
		DurationMS:    17,
	}
}

type contextKey string

const validRequestJSON = `{
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

func validRequest() extract.ExtractSpecRequest {
	return extract.ExtractSpecRequest{
		RequestID: "123e4567-e89b-12d3-a456-426614174000",
		SentAt:    "2026-04-24T18:30:00Z",
		Client: extract.ClientInfo{
			App:           "sally-extension",
			Version:       "0.1.0",
			ChromeVersion: "136.0.0.0",
		},
		Page: extract.PagePayload{
			Title:          "Example Co. WF-200 Wall Faucet",
			URL:            "https://example.com/products/wf-200",
			VisibleText:    "Wall-mounted faucet rough-in dimensions and installation notes.",
			MainImageURL:   "https://example.com/faucet.jpg",
			StructuredData: []json.RawMessage{},
			PDFLinks:       []string{"https://example.com/spec-sheet.pdf"},
		},
		ProjectContext: extract.ProjectContext{
			ProjectName:     "My New Project",
			KnownZones:      []string{"Primary Bath"},
			KnownCategories: []string{"Plumbing Fixture"},
		},
		Options: extract.ExtractSpecOptions{
			IncludeDebug:       true,
			ReturnAlternatives: false,
		},
	}
}

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaExtractorMapsJSONResponseIntoContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/generate" {
			t.Fatalf("expected /api/generate, got %s", r.URL.Path)
		}

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if reqBody["model"] != "qwen2.5:7b" {
			t.Fatalf("expected model in request, got %#v", reqBody["model"])
		}
		if reqBody["stream"] != false {
			t.Fatalf("expected non-streaming request, got %#v", reqBody["stream"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": `{
				"title":"Wall Faucet",
				"manufacturer":"Example Co.",
				"modelNumber":"WF-200",
				"category":"Plumbing Fixture",
				"description":"Wall-mounted faucet with rough-in requirements.",
				"finish":"Polished Chrome",
				"finishModelNumber":"WF-200-PC",
				"availableFinishes":["Polished Chrome","Brushed Nickel"],
				"finishModelMappings":[
					{"finish":"Polished Chrome","modelNumber":"WF-200-PC"},
					{"finish":"Brushed Nickel","modelNumber":"WF-200-BN"}
				],
				"requiredAddOns":["Rough valve body"],
				"optionalCompanions":["Drain assembly"],
				"zone":"Primary Bath",
				"analysis":{
					"missingFields":[],
					"warnings":["Verify finish mapping."],
					"confidence":{
						"overall":0.88,
						"title":0.95,
						"manufacturer":0.94,
						"modelNumber":0.92,
						"category":0.84,
						"description":0.8,
						"finish":0.77,
						"requiredAddOns":0.7
					}
				}
			}`,
		})
	}))
	defer server.Close()

	extractor := NewOllamaExtractor("qwen2.5:7b", server.URL, server.Client())

	resp, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if err != nil {
		t.Fatalf("extract returned error: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected ok status, got %q", resp.Status)
	}
	if resp.Proposal == nil || resp.Proposal.Title != "Wall Faucet" {
		t.Fatalf("expected mapped proposal, got %#v", resp.Proposal)
	}
	if resp.Proposal.SourceURL != "https://example.com/products/wf-200" {
		t.Fatalf("expected original source url, got %q", resp.Proposal.SourceURL)
	}
	if resp.Analysis == nil || resp.Analysis.Confidence.Overall != 0.88 {
		t.Fatalf("expected mapped analysis, got %#v", resp.Analysis)
	}
	if resp.Meta.Provider != "ollama" || resp.Meta.Model != "qwen2.5:7b" {
		t.Fatalf("expected ollama meta, got %#v", resp.Meta)
	}
}

func TestOllamaExtractorReturnsProviderFailureForInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": `not-json`,
		})
	}))
	defer server.Close()

	extractor := NewOllamaExtractor("qwen2.5:7b", server.URL, server.Client())

	_, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if !errors.Is(err, ErrFailure) {
		t.Fatalf("expected ErrFailure, got %v", err)
	}
	if !strings.Contains(err.Error(), "decode structured output") {
		t.Fatalf("expected decode structured output failure, got %v", err)
	}
}

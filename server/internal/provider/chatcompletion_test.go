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

func TestChatCompletionExtractorUsesCorrectEndpointAndAuth(t *testing.T) {
	var capturedMethod, capturedPath, capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		writeChatCompletionResponse(w, goodExtractionPayload())
	}))
	defer server.Close()

	extractor := NewChatCompletionExtractor("test-key", "llama-3.3-70b-versatile", server.URL, "json_schema", server.Client())
	if _, err := extractor.Extract(context.Background(), validOpenAIRequest()); err != nil {
		t.Fatalf("extract returned error: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if capturedPath != "/chat/completions" {
		t.Errorf("expected /chat/completions, got %s", capturedPath)
	}
	if capturedAuth != "Bearer test-key" {
		t.Errorf("expected bearer auth, got %q", capturedAuth)
	}
}

func TestChatCompletionExtractorSendsJsonSchemaResponseFormat(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		writeChatCompletionResponse(w, goodExtractionPayload())
	}))
	defer server.Close()

	extractor := NewChatCompletionExtractor("k", "llama-3.1-8b-instant", server.URL, "json_schema", server.Client())
	if _, err := extractor.Extract(context.Background(), validOpenAIRequest()); err != nil {
		t.Fatalf("extract returned error: %v", err)
	}

	rf := capturedBody["response_format"].(map[string]any)
	if rf["type"] != "json_schema" {
		t.Fatalf("expected json_schema response format, got %#v", rf["type"])
	}
	schema := rf["json_schema"].(map[string]any)
	if schema["name"] != "sally_extraction" {
		t.Fatalf("expected schema name, got %#v", schema["name"])
	}
	if schema["strict"] != true {
		t.Fatalf("expected strict schema, got %#v", schema["strict"])
	}
}

func TestChatCompletionExtractorEmbedsFieldListInSystemPromptForJsonObjectMode(t *testing.T) {
	var capturedMessages []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, m := range body["messages"].([]any) {
			capturedMessages = append(capturedMessages, m.(map[string]any))
		}
		writeChatCompletionResponse(w, goodExtractionPayload())
	}))
	defer server.Close()

	extractor := NewChatCompletionExtractor("k", "llama-3.3-70b-versatile", server.URL, "json_object", server.Client())
	if _, err := extractor.Extract(context.Background(), validOpenAIRequest()); err != nil {
		t.Fatalf("extract returned error: %v", err)
	}

	if len(capturedMessages) == 0 {
		t.Fatal("expected messages in request")
	}
	systemMsg := capturedMessages[0]
	if systemMsg["role"] != "system" {
		t.Fatalf("expected system message first, got %q", systemMsg["role"])
	}
	content := systemMsg["content"].(string)
	for _, field := range []string{"title", "manufacturer", "modelNumber", "requiredAddOns", "analysis"} {
		if !strings.Contains(content, field) {
			t.Errorf("expected field %q in system prompt for json_object mode, prompt=%q", field, content[:min(200, len(content))])
		}
	}
}

func TestChatCompletionExtractorMapsResponseIntoContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletionResponse(w, goodExtractionPayload())
	}))
	defer server.Close()

	extractor := NewChatCompletionExtractor("k", "llama-3.3-70b-versatile", server.URL, "json_schema", server.Client())
	resp, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if err != nil {
		t.Fatalf("extract returned error: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("expected ok status, got %q", resp.Status)
	}
	p := resp.Proposal
	if p == nil {
		t.Fatal("expected proposal")
	}
	if p.Title != "Wall Faucet" {
		t.Errorf("expected title, got %q", p.Title)
	}
	if p.SourceURL != "https://example.com/products/wf-200" {
		t.Errorf("expected source URL from page payload, got %q", p.SourceURL)
	}
	if resp.Meta.Provider != "chatcompletion" || resp.Meta.Model != "llama-3.3-70b-versatile" {
		t.Errorf("expected provider meta, got %#v", resp.Meta)
	}
}

func TestChatCompletionExtractorNormalizesNullArrayFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Model returns null for optional array fields — common in json_object mode.
		writeChatCompletionResponse(w, map[string]any{
			"title": "Wall Faucet", "manufacturer": "Example Co.", "modelNumber": "WF-200",
			"category": "Plumbing Fixture", "description": "A faucet.", "finish": "Chrome",
			"finishModelNumber": "", "zone": "",
			"availableFinishes":   nil,
			"finishModelMappings": nil,
			"requiredAddOns":      nil,
			"optionalCompanions":  nil,
			"analysis": map[string]any{
				"missingFields": nil,
				"warnings":      nil,
				"confidence": map[string]any{
					"overall": 0.9, "title": 0.9, "manufacturer": 0.9, "modelNumber": 0.9,
					"category": 0.9, "description": 0.9, "finish": 0.9, "requiredAddOns": 0.9,
				},
			},
		})
	}))
	defer server.Close()

	extractor := NewChatCompletionExtractor("k", "model", server.URL, "json_object", server.Client())
	resp, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if err != nil {
		t.Fatalf("expected no error for null arrays, got: %v", err)
	}
	if resp.Proposal.RequiredAddOns == nil {
		t.Error("expected non-nil RequiredAddOns slice")
	}
	if resp.Proposal.OptionalCompanions == nil {
		t.Error("expected non-nil OptionalCompanions slice")
	}
	if resp.Proposal.AvailableFinishes == nil {
		t.Error("expected non-nil AvailableFinishes slice")
	}
	if resp.Proposal.FinishModelMappings == nil {
		t.Error("expected non-nil FinishModelMappings slice")
	}
}

func TestChatCompletionExtractorMapsTimeoutToProviderTimeout(t *testing.T) {
	extractor := NewChatCompletionExtractor("k", "model", "https://api.groq.test", "json_schema", timeoutClient{})

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	_, err := extractor.Extract(ctx, validOpenAIRequest())
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestChatCompletionExtractorIncludesUpstreamErrorBodyInFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "This model does not support response format `json_schema`.",
			},
		})
	}))
	defer server.Close()

	extractor := NewChatCompletionExtractor("k", "llama-3.3-70b-versatile", server.URL, "json_schema", server.Client())
	_, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if !errors.Is(err, ErrFailure) {
		t.Fatalf("expected ErrFailure, got %v", err)
	}
	if !strings.Contains(err.Error(), "json_schema") {
		t.Fatalf("expected upstream error body in message, got %v", err)
	}
}

func writeChatCompletionResponse(w http.ResponseWriter, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"role": "assistant", "content": mustJSON(payload)}},
		},
	})
}

func goodExtractionPayload() map[string]any {
	return map[string]any{
		"title": "Wall Faucet", "manufacturer": "Example Co.", "modelNumber": "WF-200",
		"category": "Plumbing Fixture", "description": "Wall-mounted faucet with rough-in requirements.",
		"finish": "Polished Chrome", "finishModelNumber": "WF-200-PC",
		"availableFinishes":   []string{"Polished Chrome", "Brushed Nickel"},
		"finishModelMappings": []map[string]any{{"finish": "Polished Chrome", "modelNumber": "WF-200-PC"}},
		"requiredAddOns":      []string{"Rough valve body"},
		"optionalCompanions":  []string{"Drain assembly"},
		"zone":                "Primary Bath",
		"analysis": map[string]any{
			"missingFields": []string{},
			"warnings":      []string{},
			"confidence": map[string]any{
				"overall": 0.92, "title": 0.99, "manufacturer": 0.98, "modelNumber": 0.97,
				"category": 0.88, "description": 0.84, "finish": 0.82, "requiredAddOns": 0.8,
			},
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

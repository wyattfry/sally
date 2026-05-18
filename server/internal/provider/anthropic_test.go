package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

)

func TestAnthropicExtractorRequestShape(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("expected x-api-key header, got %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicVersion {
			t.Fatalf("expected anthropic-version header, got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		writeAnthropicResponse(w, validExtractionPayload())
	}))
	defer server.Close()

	extractor := NewAnthropicExtractor("test-key", "claude-sonnet-4-6", server.URL, server.Client())
	_, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if err != nil {
		t.Fatalf("extract returned error: %v", err)
	}

	if capturedBody["model"] != "claude-sonnet-4-6" {
		t.Fatalf("expected model in request, got %#v", capturedBody["model"])
	}

	// System prompt must include the prompt version
	system, _ := capturedBody["system"].(string)
	if !strings.Contains(system, PromptVersion) {
		t.Fatalf("expected prompt version in system prompt, got %q", system)
	}

	// Messages: single user turn
	messages := capturedBody["messages"].([]any)
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	userMsg := messages[0].(map[string]any)
	if userMsg["role"] != "user" {
		t.Fatalf("expected user message, got %#v", userMsg["role"])
	}
	content := userMsg["content"].([]any)
	textBlock := content[0].(map[string]any)
	if textBlock["type"] != "text" {
		t.Fatalf("expected text block, got %#v", textBlock["type"])
	}
	if !strings.Contains(textBlock["text"].(string), "Wall-mounted faucet rough-in dimensions") {
		t.Fatalf("expected page text in user content, got %#v", textBlock["text"])
	}
	if len(content) != 1 {
		t.Fatalf("expected only text block in content (no image), got %d blocks", len(content))
	}

	// Tool definition must use the extraction schema
	tools := capturedBody["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "extract_spec" {
		t.Fatalf("expected extract_spec tool, got %#v", tool["name"])
	}
	if tool["input_schema"] == nil {
		t.Fatal("expected input_schema in tool definition")
	}

	// Tool choice must force the specific tool
	toolChoice := capturedBody["tool_choice"].(map[string]any)
	if toolChoice["type"] != "tool" || toolChoice["name"] != "extract_spec" {
		t.Fatalf("expected forced tool_choice, got %#v", toolChoice)
	}
}

func TestAnthropicExtractorMapsStructuredOutputIntoContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAnthropicResponse(w, validExtractionPayload())
	}))
	defer server.Close()

	extractor := NewAnthropicExtractor("test-key", "claude-sonnet-4-6", server.URL, server.Client())
	resp, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if err != nil {
		t.Fatalf("extract returned error: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("expected ok status, got %q", resp.Status)
	}
	if resp.Proposal == nil {
		t.Fatal("expected proposal")
	}
	if resp.Proposal.Title != "Wall Faucet" {
		t.Fatalf("expected title from model output, got %q", resp.Proposal.Title)
	}
	// Source fields come from the original request, not the model output
	if resp.Proposal.SourceURL != "https://example.com/products/wf-200" {
		t.Fatalf("expected source URL from page payload, got %q", resp.Proposal.SourceURL)
	}
	if resp.Proposal.SourceImageURL != "https://example.com/faucet.jpg" {
		t.Fatalf("expected source image from page payload, got %q", resp.Proposal.SourceImageURL)
	}
	if len(resp.Proposal.SourcePDFLinks) != 1 || resp.Proposal.SourcePDFLinks[0] != "https://example.com/spec-sheet.pdf" {
		t.Fatalf("expected source PDF links from page payload, got %#v", resp.Proposal.SourcePDFLinks)
	}
	if resp.Analysis == nil || resp.Analysis.Confidence.Overall != 0.92 {
		t.Fatalf("expected mapped analysis, got %#v", resp.Analysis)
	}
	if resp.Meta.Provider != "anthropic" || resp.Meta.Model != "claude-sonnet-4-6" || resp.Meta.PromptVersion != PromptVersion {
		t.Fatalf("expected provider meta, got %#v", resp.Meta)
	}
}

func TestAnthropicExtractorFailsWhenNoToolUseBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Sorry, I cannot help with that."},
			},
		})
	}))
	defer server.Close()

	extractor := NewAnthropicExtractor("test-key", "claude-sonnet-4-6", server.URL, server.Client())
	_, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if !errors.Is(err, ErrFailure) {
		t.Fatalf("expected ErrFailure, got %v", err)
	}
	if !strings.Contains(err.Error(), "no tool_use block") {
		t.Fatalf("expected clear error message, got %v", err)
	}
}

func TestAnthropicExtractorMapsUpstreamErrorIntoFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":    "error",
			"error":   map[string]any{"type": "invalid_request_error", "message": "model not found"},
		})
	}))
	defer server.Close()

	extractor := NewAnthropicExtractor("test-key", "claude-bad-model", server.URL, server.Client())
	_, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if !errors.Is(err, ErrFailure) {
		t.Fatalf("expected ErrFailure, got %v", err)
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("expected upstream error body in message, got %v", err)
	}
}

func TestAnthropicExtractorMapsTimeoutToProviderTimeout(t *testing.T) {
	extractor := NewAnthropicExtractor("test-key", "claude-sonnet-4-6", "https://api.anthropic.test", timeoutClient{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, err := extractor.Extract(ctx, validOpenAIRequest())
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func writeAnthropicResponse(w http.ResponseWriter, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"content": []map[string]any{
			{
				"type":  "tool_use",
				"id":    "toolu_01ABC",
				"name":  "extract_spec",
				"input": payload,
			},
		},
		"stop_reason": "tool_use",
	})
}

func validExtractionPayload() map[string]any {
	return map[string]any{
		"title":             "Wall Faucet",
		"manufacturer":      "Example Co.",
		"modelNumber":       "WF-200",
		"category":          "Plumbing Fixture",
		"description":       "Wall-mounted faucet with rough-in requirements.",
		"finish":            "Polished Chrome",
		"finishModelNumber": "WF-200-PC",
		"availableFinishes": []string{"Polished Chrome"},
		"finishModelMappings": []map[string]any{
			{"finish": "Polished Chrome", "modelNumber": "WF-200-PC"},
		},
		"requiredAddOns":     []string{"Rough valve body"},
		"optionalCompanions": []string{"Drain assembly"},
		"room":               "Primary Bath",
		"analysis": map[string]any{
			"missingFields": []string{},
			"warnings":      []string{},
			"confidence": map[string]any{
				"overall":        0.92,
				"title":          0.99,
				"manufacturer":   0.98,
				"modelNumber":    0.97,
				"category":       0.88,
				"description":    0.84,
				"finish":         0.82,
				"requiredAddOns": 0.8,
			},
		},
	}
}

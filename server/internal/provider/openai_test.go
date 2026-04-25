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

	"sally/server/internal/extract"
)

func TestOpenAIExtractorRequestIncludesModelPromptVersionAndInputs(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected bearer auth header, got %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}

		writeOpenAIResponse(w, map[string]any{
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
			"zone":               "Primary Bath",
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
		})
	}))
	defer server.Close()

	extractor := NewOpenAIExtractor("test-key", "gpt-4.1-mini", server.URL, server.Client())

	_, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if err != nil {
		t.Fatalf("extract returned error: %v", err)
	}

	if capturedBody["model"] != "gpt-4.1-mini" {
		t.Fatalf("expected model in request, got %#v", capturedBody["model"])
	}

	input := capturedBody["input"].([]any)
	developer := input[0].(map[string]any)
	user := input[1].(map[string]any)
	if developer["role"] != "developer" {
		t.Fatalf("expected developer message, got %#v", developer["role"])
	}
	developerContent := developer["content"].([]any)
	if !strings.Contains(developerContent[0].(map[string]any)["text"].(string), PromptVersion) {
		t.Fatalf("expected prompt version in developer prompt, got %#v", developerContent[0])
	}

	userContent := user["content"].([]any)
	if !strings.Contains(userContent[0].(map[string]any)["text"].(string), "Wall-mounted faucet rough-in dimensions") {
		t.Fatalf("expected page text in user prompt, got %#v", userContent[0])
	}
	if userContent[1].(map[string]any)["image_url"] != "https://example.com/faucet.jpg" {
		t.Fatalf("expected image url in user content, got %#v", userContent[1])
	}

	textConfig := capturedBody["text"].(map[string]any)
	format := textConfig["format"].(map[string]any)
	if format["type"] != "json_schema" {
		t.Fatalf("expected json_schema format, got %#v", format["type"])
	}
}

func TestOpenAIExtractorMapsStructuredOutputIntoContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeOpenAIResponse(w, map[string]any{
			"title":             "Wall Faucet",
			"manufacturer":      "Example Co.",
			"modelNumber":       "WF-200",
			"category":          "Plumbing Fixture",
			"description":       "Wall-mounted faucet with rough-in requirements.",
			"finish":            "Polished Chrome",
			"finishModelNumber": "WF-200-PC",
			"availableFinishes": []string{"Polished Chrome", "Brushed Nickel"},
			"finishModelMappings": []map[string]any{
				{"finish": "Polished Chrome", "modelNumber": "WF-200-PC"},
				{"finish": "Brushed Nickel", "modelNumber": "WF-200-BN"},
			},
			"requiredAddOns":     []string{"Rough valve body"},
			"optionalCompanions": []string{"Drain assembly"},
			"zone":               "Primary Bath",
			"sourceUrl":          "https://ignored.example.com/other",
			"sourceTitle":        "Ignored Source",
			"sourceImageUrl":     "https://ignored.example.com/other.jpg",
			"sourcePdfLinks":     []string{"https://ignored.example.com/spec.pdf"},
			"analysis": map[string]any{
				"missingFields": []string{},
				"warnings":      []string{"Verify finish mapping."},
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
		})
	}))
	defer server.Close()

	extractor := NewOpenAIExtractor("test-key", "gpt-4.1-mini", server.URL, server.Client())

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
	if resp.Proposal.SourceURL != "https://example.com/products/wf-200" {
		t.Fatalf("expected source URL from original page payload, got %q", resp.Proposal.SourceURL)
	}
	if resp.Proposal.SourceTitle != "Example Co. WF-200 Wall Faucet" {
		t.Fatalf("expected source title from original page payload, got %q", resp.Proposal.SourceTitle)
	}
	if resp.Proposal.SourceImageURL != "https://example.com/faucet.jpg" {
		t.Fatalf("expected source image from original page payload, got %q", resp.Proposal.SourceImageURL)
	}
	if len(resp.Proposal.SourcePDFLinks) != 1 || resp.Proposal.SourcePDFLinks[0] != "https://example.com/spec-sheet.pdf" {
		t.Fatalf("expected source pdf links from original page payload, got %#v", resp.Proposal.SourcePDFLinks)
	}
	if resp.Analysis == nil || resp.Analysis.Confidence.Overall != 0.92 {
		t.Fatalf("expected mapped analysis, got %#v", resp.Analysis)
	}
	if resp.Meta.Provider != "openai" || resp.Meta.Model != "gpt-4.1-mini" || resp.Meta.PromptVersion != PromptVersion {
		t.Fatalf("expected provider meta, got %#v", resp.Meta)
	}
}

func TestOpenAIExtractorMapsTimeoutToProviderTimeout(t *testing.T) {
	extractor := NewOpenAIExtractor("test-key", "gpt-4.1-mini", "https://api.openai.test", timeoutClient{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, err := extractor.Extract(ctx, validOpenAIRequest())
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestOpenAIExtractorFailsClearlyWhenStructuredOutputTextIsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{
					"type":    "message",
					"role":    "assistant",
					"content": []map[string]any{},
				},
			},
		})
	}))
	defer server.Close()

	extractor := NewOpenAIExtractor("test-key", "gpt-4.1-mini", server.URL, server.Client())

	_, err := extractor.Extract(context.Background(), validOpenAIRequest())
	if !errors.Is(err, ErrFailure) {
		t.Fatalf("expected ErrFailure, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing structured output") {
		t.Fatalf("expected clear missing structured output message, got %v", err)
	}
}

func TestExtractionSchemaDefinesNestedObjectsStrictly(t *testing.T) {
	schema := extractionSchema()

	properties := schema["properties"].(map[string]any)
	finishMappings := properties["finishModelMappings"].(map[string]any)
	finishMappingItems := finishMappings["items"].(map[string]any)
	if finishMappingItems["type"] != "object" {
		t.Fatalf("expected finishModelMappings items to be object, got %#v", finishMappingItems["type"])
	}
	if finishMappingItems["additionalProperties"] != false {
		t.Fatalf("expected finishModelMappings items to disallow extra fields, got %#v", finishMappingItems["additionalProperties"])
	}
	requiredFinishFields := finishMappingItems["required"].([]string)
	if len(requiredFinishFields) != 2 || requiredFinishFields[0] != "finish" || requiredFinishFields[1] != "modelNumber" {
		t.Fatalf("expected required finish mapping fields, got %#v", requiredFinishFields)
	}

	analysis := properties["analysis"].(map[string]any)
	if analysis["type"] != "object" || analysis["additionalProperties"] != false {
		t.Fatalf("expected strict analysis schema, got %#v", analysis)
	}
	analysisProperties := analysis["properties"].(map[string]any)
	confidence := analysisProperties["confidence"].(map[string]any)
	if confidence["type"] != "object" || confidence["additionalProperties"] != false {
		t.Fatalf("expected strict confidence schema, got %#v", confidence)
	}
	requiredConfidenceFields := confidence["required"].([]string)
	if len(requiredConfidenceFields) != 8 {
		t.Fatalf("expected all confidence fields to be required, got %#v", requiredConfidenceFields)
	}
}

type timeoutClient struct{}

func (timeoutClient) Do(*http.Request) (*http.Response, error) {
	return nil, context.DeadlineExceeded
}

func writeOpenAIResponse(w http.ResponseWriter, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"output": []map[string]any{
			{
				"type": "message",
				"role": "assistant",
				"content": []map[string]any{
					{
						"type": "output_text",
						"text": mustJSON(payload),
					},
				},
			},
		},
	})
}

func mustJSON(value any) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func validOpenAIRequest() extract.ExtractSpecRequest {
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
			StructuredData: []json.RawMessage{json.RawMessage(`{"@type":"Product","name":"Wall Faucet"}`)},
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

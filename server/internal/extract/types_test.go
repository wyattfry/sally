package extract

import (
	"encoding/json"
	"testing"
)

func TestExtractSpecRequestJSONDecodesApprovedContract(t *testing.T) {
	payload := []byte(`{
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
			"visibleText": "....",
			"mainImageUrl": "https://example.com/faucet.jpg",
			"structuredData": [],
			"pdfLinks": [
				"https://example.com/spec-sheet.pdf"
			]
		},
		"projectContext": {
			"projectName": "My New Project",
			"knownZones": ["Primary Bath", "Powder Room"],
			"knownCategories": [
				"Plumbing Fixture",
				"Lighting",
				"Appliance",
				"Hardware",
				"Finish",
				"Furniture",
				"Accessory"
			]
		},
		"options": {
			"includeDebug": true,
			"returnAlternatives": false
		}
	}`)

	var req ExtractSpecRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	if req.RequestID == "" {
		t.Fatal("expected requestId to decode")
	}
	if req.SentAt == "" {
		t.Fatal("expected sentAt to decode")
	}
	if req.Client.App != "sally-extension" {
		t.Fatalf("expected client.app to decode, got %q", req.Client.App)
	}
	if req.Page.Title != "Example Co. WF-200 Wall Faucet" {
		t.Fatalf("expected page.title to decode, got %q", req.Page.Title)
	}
	if req.Page.URL != "https://example.com/products/wf-200" {
		t.Fatalf("expected page.url to decode, got %q", req.Page.URL)
	}
	if len(req.Page.StructuredData) != 0 {
		t.Fatalf("expected empty structuredData slice, got %d items", len(req.Page.StructuredData))
	}
	if len(req.Page.PDFLinks) != 1 || req.Page.PDFLinks[0] != "https://example.com/spec-sheet.pdf" {
		t.Fatalf("expected pdfLinks to decode, got %#v", req.Page.PDFLinks)
	}
	if req.ProjectContext.ProjectName != "My New Project" {
		t.Fatalf("expected projectContext.projectName to decode, got %q", req.ProjectContext.ProjectName)
	}
	if len(req.ProjectContext.KnownCategories) != 7 {
		t.Fatalf("expected knownCategories to decode, got %d items", len(req.ProjectContext.KnownCategories))
	}
	if !req.Options.IncludeDebug {
		t.Fatal("expected options.includeDebug to decode")
	}
	if req.Options.ReturnAlternatives {
		t.Fatal("expected options.returnAlternatives to decode as false")
	}
}

func TestExtractSpecResponseJSONDecodesApprovedContract(t *testing.T) {
	payload := []byte(`{
		"requestId": "123e4567-e89b-12d3-a456-426614174000",
		"status": "ok",
		"proposal": {
			"title": "Wall Faucet",
			"manufacturer": "Example Co.",
			"modelNumber": "WF-200",
			"category": "Plumbing Fixture",
			"description": "Wall-mounted faucet with rough-in requirements and installation constraints noted from the page.",
			"finish": "Polished Chrome",
			"finishModelNumber": "",
			"availableFinishes": ["Polished Chrome", "Brushed Nickel"],
			"finishModelMappings": [
				{
					"finish": "Polished Chrome",
					"modelNumber": "WF-200-PC"
				},
				{
					"finish": "Brushed Nickel",
					"modelNumber": "WF-200-BN"
				}
			],
			"requiredAddOns": ["Rough valve body"],
			"optionalCompanions": ["Drain assembly"],
			"zone": "",
			"sourceUrl": "https://example.com/products/wf-200",
			"sourceTitle": "Example Co. WF-200 Wall Faucet",
			"sourceImageUrl": "https://example.com/faucet.jpg",
			"sourcePdfLinks": [
				"https://example.com/spec-sheet.pdf"
			]
		},
		"analysis": {
			"missingFields": [],
			"warnings": [
				"Finish-to-model mapping inferred from page copy, verify before approval."
			],
			"confidence": {
				"overall": 0.84,
				"title": 0.96,
				"manufacturer": 0.94,
				"modelNumber": 0.92,
				"category": 0.78,
				"description": 0.73,
				"finish": 0.81,
				"requiredAddOns": 0.67
			}
		},
		"meta": {
			"provider": "openai",
			"model": "gpt-5-mini",
			"promptVersion": "extract-spec-v1",
			"durationMs": 1820
		}
	}`)

	var resp ExtractSpecResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal success response: %v", err)
	}

	if resp.RequestID == "" {
		t.Fatal("expected requestId to decode")
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %q", resp.Status)
	}
	if resp.Proposal == nil {
		t.Fatal("expected proposal to decode")
	}
	if resp.Proposal.Title != "Wall Faucet" {
		t.Fatalf("expected proposal.title to decode, got %q", resp.Proposal.Title)
	}
	if resp.Proposal.SourceURL != "https://example.com/products/wf-200" {
		t.Fatalf("expected proposal.sourceUrl to decode, got %q", resp.Proposal.SourceURL)
	}
	if len(resp.Proposal.FinishModelMappings) != 2 {
		t.Fatalf("expected finishModelMappings to decode, got %d items", len(resp.Proposal.FinishModelMappings))
	}
	if resp.Meta.Provider != "openai" {
		t.Fatalf("expected meta.provider to decode, got %q", resp.Meta.Provider)
	}
	if resp.Meta.PromptVersion != "extract-spec-v1" {
		t.Fatalf("expected meta.promptVersion to decode, got %q", resp.Meta.PromptVersion)
	}
	if resp.Analysis == nil {
		t.Fatal("expected analysis to decode")
	}
	if resp.Analysis.Confidence.Overall != 0.84 {
		t.Fatalf("expected analysis.confidence.overall to decode, got %v", resp.Analysis.Confidence.Overall)
	}
}

func TestExtractSpecErrorResponseJSONDecodesApprovedContract(t *testing.T) {
	payload := []byte(`{
		"requestId": "123e4567-e89b-12d3-a456-426614174000",
		"status": "error",
		"error": {
			"code": "MODEL_TIMEOUT",
			"message": "Extraction did not complete in time."
		},
		"meta": {
			"provider": "openai",
			"model": "gpt-5-mini",
			"promptVersion": "extract-spec-v1",
			"durationMs": 15000
		}
	}`)

	var resp ExtractSpecResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}

	if resp.Status != "error" {
		t.Fatalf("expected status error, got %q", resp.Status)
	}
	if resp.Error == nil {
		t.Fatal("expected error payload to decode")
	}
	if resp.Error.Code != "MODEL_TIMEOUT" {
		t.Fatalf("expected error.code to decode, got %q", resp.Error.Code)
	}
	if resp.Meta.Provider != "openai" {
		t.Fatalf("expected meta.provider to decode, got %q", resp.Meta.Provider)
	}
}

func TestExtractSpecSuccessResponseJSONOmitsError(t *testing.T) {
	resp := ExtractSpecResponse{
		RequestID: "123e4567-e89b-12d3-a456-426614174000",
		Status:    "ok",
		Proposal: &Proposal{
			Title:               "Wall Faucet",
			SourceURL:           "https://example.com/products/wf-200",
			SourcePDFLinks:      []string{"https://example.com/spec-sheet.pdf"},
			AvailableFinishes:   []string{},
			FinishModelMappings: []FinishModelMapping{},
			RequiredAddOns:      []string{},
			OptionalCompanions:  []string{},
		},
		Meta: ResponseMeta{
			Provider:      "openai",
			Model:         "gpt-5-mini",
			PromptVersion: "extract-spec-v1",
			DurationMS:    1820,
		},
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal success response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode marshaled success response: %v", err)
	}

	if _, exists := decoded["error"]; exists {
		t.Fatal("expected error field to be omitted on success response")
	}
}

func TestExtractSpecErrorResponseJSONOmitsProposalAndAnalysis(t *testing.T) {
	resp := ExtractSpecResponse{
		RequestID: "123e4567-e89b-12d3-a456-426614174000",
		Status:    "error",
		Error: &ErrorPayload{
			Code:    "MODEL_TIMEOUT",
			Message: "Extraction did not complete in time.",
		},
		Meta: ResponseMeta{
			Provider:      "openai",
			Model:         "gpt-5-mini",
			PromptVersion: "extract-spec-v1",
			DurationMS:    15000,
		},
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode marshaled error response: %v", err)
	}

	if _, exists := decoded["proposal"]; exists {
		t.Fatal("expected proposal field to be omitted on error response")
	}
	if _, exists := decoded["analysis"]; exists {
		t.Fatal("expected analysis field to be omitted on error response")
	}
}

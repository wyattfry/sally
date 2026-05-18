package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"sally/server/internal/extract"
)

const PromptVersion = "extract-spec-v4"

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type OpenAIExtractor struct {
	apiKey  string
	model   string
	baseURL string
	client  httpDoer
}

func NewOpenAIExtractor(apiKey string, model string, baseURL string, client httpDoer) OpenAIExtractor {
	return OpenAIExtractor{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (o OpenAIExtractor) Meta() extract.ResponseMeta {
	return extract.ResponseMeta{
		Provider:      "openai",
		Model:         o.model,
		PromptVersion: PromptVersion,
		DurationMS:    0,
	}
}

func (o OpenAIExtractor) Extract(ctx context.Context, req extract.ExtractSpecRequest) (extract.ExtractSpecResponse, error) {
	start := time.Now()

	builtReq := buildOpenAIRequest(req, o.model)
	requestBody, err := json.Marshal(builtReq)
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: marshal request: %v", ErrFailure, err)
	}
	promptText := capLog(string(requestBody))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/responses", bytes.NewReader(requestBody))
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: build request: %v", ErrFailure, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := o.client.Do(httpReq)
	if err != nil {
		if isTimeoutError(err) {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream request failed: %v", ErrFailure, err)
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: read response: %v", ErrFailure, err)
	}
	if httpResp.StatusCode >= 400 {
		if httpResp.StatusCode == http.StatusGatewayTimeout || httpResp.StatusCode == http.StatusRequestTimeout {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrTimeout, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
		}
		if httpResp.StatusCode == http.StatusTooManyRequests || httpResp.StatusCode == 529 {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrOverloaded, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
		}
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrFailure, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
	}

	var upstream openAIResponse
	if err := json.Unmarshal(responseBody, &upstream); err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: decode response: %v", ErrFailure, err)
	}

	outputText := strings.TrimSpace(upstream.OutputText())
	if outputText == "" {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: missing structured output text", ErrFailure)
	}

	var output openAIExtractionOutput
	if err := json.Unmarshal([]byte(outputText), &output); err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: decode structured output: %v", ErrFailure, err)
	}

	meta := o.Meta()
	meta.DurationMS = int(time.Since(start).Milliseconds())
	meta.PromptTokens = upstream.Usage.InputTokens
	meta.CompletionTokens = upstream.Usage.OutputTokens

	return extract.ExtractSpecResponse{
		RequestID: req.RequestID,
		Status:    "ok",
		Proposal: &extract.Proposal{
			Title:                 output.Title,
			Manufacturer:          output.Manufacturer,
			ModelNumber:           output.ModelNumber,
			Category:              output.Category,
			Description:           output.Description,
			Finish:                output.Finish,
			FinishModelNumber:     output.FinishModelNumber,
			AvailableFinishes:     output.AvailableFinishes,
			FinishModelMappings:   output.FinishModelMappings,
			RequiredAddOns:        output.RequiredAddOns,
			OptionalCompanions:    output.OptionalCompanions,
			Room:                  sanitizeRoom(output.Room),
			SuggestedScheduleName: output.SuggestedScheduleName,
			SourceURL:             req.Page.URL,
			SourceTitle:           req.Page.Title,
			SourceImageURL:        req.Page.MainImageURL,
			SourcePDFLinks:        req.Page.PDFLinks,
			CustomFields:          output.CustomFields,
		},
		Analysis:     output.Analysis,
		Meta:         meta,
		PromptText:   promptText,
		ResponseText: capLog(string(responseBody)),
	}, nil
}

type openAIRequest struct {
	Model string            `json:"model"`
	Input []openAIMessage   `json:"input"`
	Text  openAITextOptions `json:"text"`
}

type openAIMessage struct {
	Role    string               `json:"role"`
	Content []openAIContentBlock `json:"content"`
}

type openAIContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type openAITextOptions struct {
	Format openAIFormat `json:"format"`
}

type openAIFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type openAIUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type openAIResponse struct {
	Output []openAIOutput `json:"output"`
	Usage  openAIUsage    `json:"usage"`
}

type openAIOutput struct {
	Content []openAIOutputContent `json:"content"`
}

type openAIOutputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (r openAIResponse) OutputText() string {
	for _, output := range r.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" && content.Text != "" {
				return content.Text
			}
		}
	}
	return ""
}

type openAIExtractionOutput struct {
	Title                 string                       `json:"title"`
	Manufacturer          string                       `json:"manufacturer"`
	ModelNumber           string                       `json:"modelNumber"`
	Category              string                       `json:"category"`
	Description           string                       `json:"description"`
	Finish                string                       `json:"finish"`
	FinishModelNumber     string                       `json:"finishModelNumber"`
	AvailableFinishes     []string                     `json:"availableFinishes"`
	FinishModelMappings   []extract.FinishModelMapping `json:"finishModelMappings"`
	RequiredAddOns        []string                     `json:"requiredAddOns"`
	OptionalCompanions    []string                     `json:"optionalCompanions"`
	Room                  string                       `json:"room"`
	SuggestedScheduleName string                       `json:"suggestedScheduleName"`
	Analysis              *extract.Analysis            `json:"analysis"`
	CustomFields          map[string]string            `json:"customFields,omitempty"`
}

func buildOpenAIRequest(req extract.ExtractSpecRequest, model string) openAIRequest {
	userText := buildUserPrompt(req)
	userContent := []openAIContentBlock{
		{Type: "input_text", Text: userText},
	}
	if req.Page.MainImageURL != "" {
		userContent = append(userContent, openAIContentBlock{
			Type:     "input_image",
			ImageURL: req.Page.MainImageURL,
		})
	}

	return openAIRequest{
		Model: model,
		Input: []openAIMessage{
			{
				Role: "developer",
				Content: []openAIContentBlock{
					{
						Type: "input_text",
						Text: "You are Sally. Extract one architectural schedule proposal as strict JSON. " +
							"Leave any field as an empty string if the value is not present or cannot be determined — " +
							"never use placeholder values such as '<UNKNOWN>', 'N/A', 'Unknown', or similar. " +
							"Prompt version: " + PromptVersion,
					},
				},
			},
			{
				Role:    "user",
				Content: userContent,
			},
		},
		Text: openAITextOptions{
			Format: openAIFormat{
				Type:   "json_schema",
				Name:   "sally_extraction",
				Strict: true,
				Schema: extractionSchema(req.CustomColumns),
			},
		},
	}
}

func buildUserPrompt(req extract.ExtractSpecRequest) string {
	structuredData, _ := json.Marshal(req.Page.StructuredData)
	pdfLinks, _ := json.Marshal(req.Page.PDFLinks)
	knownCategories, _ := json.Marshal(req.ProjectContext.KnownCategories)

	var scheduleContext string
	candidates := filterPromptableSchedules(req.ProjectContext.Schedules)
	if len(candidates) > 0 {
		lines := make([]string, 0, len(candidates))
		for _, s := range candidates {
			roomsJSON, _ := json.Marshal(s.Rooms)
			lines = append(lines, fmt.Sprintf("  - %s: rooms in use: %s", s.Name, roomsJSON))
		}
		scheduleContext = "Existing schedules in this project — pick the suggestedScheduleName that best fits this item by category. " +
			"If the item does not belong with any of these, propose a NEW descriptive name (e.g. \"Plumbing Fixtures\" for a toilet, \"Lighting\" for a light fixture). " +
			"Do not force the item onto an existing schedule when the categories don't match.\n" +
			strings.Join(lines, "\n") +
			"\nFor room: use a plain room or area name (e.g. 'Kitchen', 'Master Bath'). Pick from existing rooms above if the item fits, or leave empty. Never output XML tags or markup in this field."
	} else {
		knownRooms, _ := json.Marshal(req.ProjectContext.KnownRooms)
		knownScheduleNames, _ := json.Marshal(req.ProjectContext.KnownScheduleNames)
		scheduleContext = "Known rooms: " + string(knownRooms) + "\n" +
			"Known schedule names: " + string(knownScheduleNames) + " — pick the best match for suggestedScheduleName, or propose a new descriptive name if none fit."
	}

	parts := []string{
		"Project: " + req.ProjectContext.ProjectName,
		scheduleContext,
		"Known categories: " + string(knownCategories),
	}

	if len(req.CustomColumns) > 0 {
		lines := make([]string, 0, len(req.CustomColumns))
		for _, col := range req.CustomColumns {
			lines = append(lines, fmt.Sprintf("  - %s (key: %s)", col.Label, col.Key))
		}
		parts = append(parts, "Custom columns to extract for this schedule (populate customFields in output):\n"+strings.Join(lines, "\n"))
	}

	parts = append(parts,
		"Page title: "+req.Page.Title,
		"Page URL: "+req.Page.URL,
		"Visible text: "+req.Page.VisibleText,
		"Structured data: "+string(structuredData),
		"PDF links: "+string(pdfLinks),
	)
	if req.Page.PDFText != "" {
		parts = append(parts, "Specification document text (extracted from linked PDFs — prioritize this for technical specs, dimensions, and finish codes):\n"+req.Page.PDFText)
	}
	return strings.Join(parts, "\n")
}

func extractionSchema(customColumns []extract.ColumnDefinition) map[string]any {
	properties := map[string]any{
		"title":             map[string]any{"type": "string"},
		"manufacturer":      map[string]any{"type": "string"},
		"modelNumber":       map[string]any{"type": "string"},
		"category":          map[string]any{"type": "string"},
		"description":       map[string]any{"type": "string"},
		"finish":            map[string]any{"type": "string"},
		"finishModelNumber": map[string]any{"type": "string"},
		"availableFinishes": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"finishModelMappings": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"finish":      map[string]any{"type": "string"},
					"modelNumber": map[string]any{"type": "string"},
				},
				"required":             []string{"finish", "modelNumber"},
				"additionalProperties": false,
			},
		},
		"requiredAddOns":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"optionalCompanions":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"room": map[string]any{
				"type":        "string",
				"description": "A plain room or area name (e.g. 'Kitchen', 'Master Bath', 'Living Room'). Use an existing room from the schedule context if provided, or leave empty. Plain text only — no XML, no markup, no JSON.",
			},
		"suggestedScheduleName": map[string]any{"type": "string"},
		"analysis": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"missingFields": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
				"warnings": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
				"confidence": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"overall":        map[string]any{"type": "number"},
						"title":          map[string]any{"type": "number"},
						"manufacturer":   map[string]any{"type": "number"},
						"modelNumber":    map[string]any{"type": "number"},
						"category":       map[string]any{"type": "number"},
						"description":    map[string]any{"type": "number"},
						"finish":         map[string]any{"type": "number"},
						"requiredAddOns": map[string]any{"type": "number"},
					},
					"required": []string{
						"overall",
						"title",
						"manufacturer",
						"modelNumber",
						"category",
						"description",
						"finish",
						"requiredAddOns",
					},
					"additionalProperties": false,
				},
			},
			"required":             []string{"missingFields", "warnings", "confidence"},
			"additionalProperties": false,
		},
	}

	required := []string{
		"title",
		"manufacturer",
		"modelNumber",
		"category",
		"description",
		"finish",
		"finishModelNumber",
		"availableFinishes",
		"finishModelMappings",
		"requiredAddOns",
		"optionalCompanions",
		"room",
		"suggestedScheduleName",
		"analysis",
	}

	customProps := make(map[string]any, len(customColumns)+1)
	for _, col := range customColumns {
		customProps[col.Key] = map[string]any{"type": "string", "description": col.Label}
	}
	// "color" is always extractable so paint items can carry their color
	// regardless of whether the schedule already has a Color column. The
	// backend auto-adds the column on save when this is non-empty.
	if _, present := customProps["color"]; !present {
		customProps["color"] = map[string]any{
			"type":        "string",
			"description": "Paint color name (e.g., 'Ultra Pure White'). Required for paint items. Leave empty for non-paint items.",
		}
	}
	properties["customFields"] = map[string]any{
		"type":                 "object",
		"properties":           customProps,
		"additionalProperties": false,
	}
	required = append(required, "customFields")

	return map[string]any{
		"type":                 "object",
		"properties":          properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func summarizeUpstreamBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "empty response body"
	}
	if len(text) > 500 {
		return text[:500] + "..."
	}
	return text
}

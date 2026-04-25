package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"sally/server/internal/extract"
)

const PromptVersion = "extract-spec-v1"

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

	requestBody, err := json.Marshal(buildOpenAIRequest(req, o.model))
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: marshal request: %v", ErrFailure, err)
	}

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

	var upstream openAIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&upstream); err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: decode response: %v", ErrFailure, err)
	}
	if httpResp.StatusCode >= 400 {
		if httpResp.StatusCode == http.StatusGatewayTimeout || httpResp.StatusCode == http.StatusRequestTimeout {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d", ErrTimeout, httpResp.StatusCode)
		}
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d", ErrFailure, httpResp.StatusCode)
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

	return extract.ExtractSpecResponse{
		RequestID: req.RequestID,
		Status:    "ok",
		Proposal: &extract.Proposal{
			Title:               output.Title,
			Manufacturer:        output.Manufacturer,
			ModelNumber:         output.ModelNumber,
			Category:            output.Category,
			Description:         output.Description,
			Finish:              output.Finish,
			FinishModelNumber:   output.FinishModelNumber,
			AvailableFinishes:   output.AvailableFinishes,
			FinishModelMappings: output.FinishModelMappings,
			RequiredAddOns:      output.RequiredAddOns,
			OptionalCompanions:  output.OptionalCompanions,
			Zone:                output.Zone,
			SourceURL:           req.Page.URL,
			SourceTitle:         req.Page.Title,
			SourceImageURL:      req.Page.MainImageURL,
			SourcePDFLinks:      req.Page.PDFLinks,
		},
		Analysis: output.Analysis,
		Meta:     meta,
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

type openAIResponse struct {
	Output []openAIOutput `json:"output"`
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
	Title               string                       `json:"title"`
	Manufacturer        string                       `json:"manufacturer"`
	ModelNumber         string                       `json:"modelNumber"`
	Category            string                       `json:"category"`
	Description         string                       `json:"description"`
	Finish              string                       `json:"finish"`
	FinishModelNumber   string                       `json:"finishModelNumber"`
	AvailableFinishes   []string                     `json:"availableFinishes"`
	FinishModelMappings []extract.FinishModelMapping `json:"finishModelMappings"`
	RequiredAddOns      []string                     `json:"requiredAddOns"`
	OptionalCompanions  []string                     `json:"optionalCompanions"`
	Zone                string                       `json:"zone"`
	Analysis            *extract.Analysis            `json:"analysis"`
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
						Text: "You are Sally. Extract one architectural schedule proposal as strict JSON. Prompt version: " + PromptVersion,
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
				Schema: extractionSchema(),
			},
		},
	}
}

func buildUserPrompt(req extract.ExtractSpecRequest) string {
	structuredData, _ := json.Marshal(req.Page.StructuredData)
	pdfLinks, _ := json.Marshal(req.Page.PDFLinks)
	knownZones, _ := json.Marshal(req.ProjectContext.KnownZones)
	knownCategories, _ := json.Marshal(req.ProjectContext.KnownCategories)

	return strings.Join([]string{
		"Project: " + req.ProjectContext.ProjectName,
		"Known zones: " + string(knownZones),
		"Known categories: " + string(knownCategories),
		"Page title: " + req.Page.Title,
		"Page URL: " + req.Page.URL,
		"Visible text: " + req.Page.VisibleText,
		"Structured data: " + string(structuredData),
		"PDF links: " + string(pdfLinks),
	}, "\n")
}

func extractionSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
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
			"requiredAddOns":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"optionalCompanions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"zone":               map[string]any{"type": "string"},
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
		},
		"required": []string{
			"title",
			"manufacturer",
			"modelNumber",
			"category",
			"description",
			"finish",
			"availableFinishes",
			"finishModelMappings",
			"requiredAddOns",
			"optionalCompanions",
			"zone",
			"analysis",
		},
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

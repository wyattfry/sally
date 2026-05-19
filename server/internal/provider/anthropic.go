package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"sally/server/internal/extract"
)

const anthropicVersion = "2023-06-01"
const anthropicDefaultBaseURL = "https://api.anthropic.com/v1"
const anthropicMaxTokens = 4096

type AnthropicExtractor struct {
	apiKey  string
	model   string
	baseURL string
	client  httpDoer
}

func NewAnthropicExtractor(apiKey, model, baseURL string, client httpDoer) AnthropicExtractor {
	if baseURL == "" {
		baseURL = anthropicDefaultBaseURL
	}
	return AnthropicExtractor{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (a AnthropicExtractor) Meta() extract.ResponseMeta {
	return extract.ResponseMeta{
		Provider:      "anthropic",
		Model:         a.model,
		PromptVersion: PromptVersion,
		DurationMS:    0,
	}
}

func (a AnthropicExtractor) Extract(ctx context.Context, req extract.ExtractSpecRequest) (extract.ExtractSpecResponse, error) {
	start := time.Now()

	builtReq := buildAnthropicRequest(req, a.model)
	body, err := json.Marshal(builtReq)
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: marshal request: %v", ErrFailure, err)
	}
	promptText := capLog(string(body))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: build request: %v", ErrFailure, err)
	}
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("Content-Type", "application/json")

	log.Printf("[anthropic] %s: POST %s/messages model=%s body_bytes=%d", req.RequestID, a.baseURL, a.model, len(body))
	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		if isTimeoutError(err) {
			log.Printf("[anthropic] %s: timeout after %dms: %v", req.RequestID, time.Since(start).Milliseconds(), err)
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		log.Printf("[anthropic] %s: request failed after %dms: %v", req.RequestID, time.Since(start).Milliseconds(), err)
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream request failed: %v", ErrFailure, err)
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: read response: %v", ErrFailure, err)
	}
	log.Printf("[anthropic] %s: response status=%d body_bytes=%d elapsed=%dms", req.RequestID, httpResp.StatusCode, len(responseBody), time.Since(start).Milliseconds())

	if httpResp.StatusCode >= 400 {
		if httpResp.StatusCode == http.StatusGatewayTimeout || httpResp.StatusCode == http.StatusRequestTimeout {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrTimeout, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
		}
		if httpResp.StatusCode == http.StatusTooManyRequests || httpResp.StatusCode == 529 {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrOverloaded, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
		}
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrFailure, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
	}

	var upstream anthropicResponse
	if err := json.Unmarshal(responseBody, &upstream); err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: decode response: %v", ErrFailure, err)
	}

	toolInput := upstream.ToolInput()
	if toolInput == nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: no tool_use block in response", ErrFailure)
	}

	// Re-marshal the tool input map to JSON then unmarshal into the output struct.
	// Anthropic returns tool inputs as a parsed object, not a string.
	toolJSON, err := json.Marshal(toolInput)
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: marshal tool input: %v", ErrFailure, err)
	}
	log.Printf("[anthropic] %s: tool_input=%q", req.RequestID, truncate(string(toolJSON), 200))

	var output openAIExtractionOutput
	if err := json.Unmarshal(toolJSON, &output); err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: decode tool input: %v", ErrFailure, err)
	}

	meta := a.Meta()
	meta.DurationMS = int(time.Since(start).Milliseconds())
	meta.PromptTokens = upstream.Usage.InputTokens
	meta.CompletionTokens = upstream.Usage.OutputTokens

	// Surface cache activity so we can verify prompt caching is working.
	// `input_tokens` here is the uncached remainder only — total prompt size is
	// input + cache_creation + cache_read. If cache_read stays at 0 across
	// repeat extractions with the same custom-columns schema, the system+tools
	// prefix is below the model's minimum cacheable size (4096 for Haiku 4.5).
	log.Printf("[anthropic] %s: usage input=%d output=%d cache_create=%d cache_read=%d",
		req.RequestID, upstream.Usage.InputTokens, upstream.Usage.OutputTokens,
		upstream.Usage.CacheCreationInputTokens, upstream.Usage.CacheReadInputTokens)

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
			AvailableFinishes:     coalesceStrings(output.AvailableFinishes),
			FinishModelMappings:   coalesceFinishMappings(output.FinishModelMappings),
			RequiredAddOns:        coalesceStrings(output.RequiredAddOns),
			OptionalCompanions:    coalesceStrings(output.OptionalCompanions),
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

func buildAnthropicRequest(req extract.ExtractSpecRequest, model string) anthropicRequest {
	userContent := []anthropicContent{
		{Type: "text", Text: buildUserPrompt(req)},
	}

	// Single cache_control marker on the system block caches `tools + system`
	// as a unit (render order: tools → system → messages). Hits when the same
	// schedule's custom columns drive the tool schema across requests — i.e.
	// spec'ing multiple items into the same schedule. Heads up: Haiku 4.5
	// requires a 4096-token minimum cacheable prefix; below that the cache
	// silently won't activate (no error). The few-shot example + tool schema
	// is borderline; the marker is a no-op until the prompt grows past 4K.
	return anthropicRequest{
		Model:     model,
		MaxTokens: anthropicMaxTokens,
		System: []anthropicSystemBlock{{
			Type:         "text",
			Text:         "You are Sally. Extract one architectural schedule proposal as strict JSON. Prompt version: " + PromptVersion + fewShotExample,
			CacheControl: &anthropicCacheControl{Type: "ephemeral"},
		}},
		Messages: []anthropicMessage{{Role: "user", Content: userContent}},
		Tools: []anthropicTool{{
			Name:        "extract_spec",
			Description: "Extract one architectural schedule item from the product page content",
			InputSchema: extractionSchema(req.CustomColumns),
		}},
		ToolChoice: anthropicToolChoice{Type: "tool", Name: "extract_spec"},
	}
}

type anthropicRequest struct {
	Model      string                 `json:"model"`
	MaxTokens  int                    `json:"max_tokens"`
	System     []anthropicSystemBlock `json:"system"`
	Messages   []anthropicMessage     `json:"messages"`
	Tools      []anthropicTool        `json:"tools"`
	ToolChoice anthropicToolChoice    `json:"tool_choice"`
}

type anthropicSystemBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicCacheControl struct {
	Type string `json:"type"` // "ephemeral" — default 5-minute TTL
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type anthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type anthropicResponse struct {
	Content []anthropicResponseContent `json:"content"`
	Usage   anthropicUsage             `json:"usage"`
}

type anthropicResponseContent struct {
	Type  string         `json:"type"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

func (r anthropicResponse) ToolInput() map[string]any {
	for _, block := range r.Content {
		if block.Type == "tool_use" && block.Name == "extract_spec" {
			return block.Input
		}
	}
	return nil
}

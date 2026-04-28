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

// ChatCompletionExtractor works with any OpenAI-compatible chat completions API:
// Groq, Together AI, OpenRouter, Mistral, or OpenAI itself.
// Configure via LLM_PROVIDER=chatcompletion + OPENAI_API_KEY / OPENAI_MODEL / OPENAI_BASE_URL.
type ChatCompletionExtractor struct {
	apiKey         string
	model          string
	baseURL        string
	responseFormat string // "json_schema" or "json_object"
	client         httpDoer
}

func NewChatCompletionExtractor(apiKey, model, baseURL, responseFormat string, client httpDoer) ChatCompletionExtractor {
	return ChatCompletionExtractor{
		apiKey:         apiKey,
		model:          model,
		baseURL:        strings.TrimRight(baseURL, "/"),
		responseFormat: responseFormat,
		client:         client,
	}
}

func (c ChatCompletionExtractor) Meta() extract.ResponseMeta {
	return extract.ResponseMeta{
		Provider:      "chatcompletion",
		Model:         c.model,
		PromptVersion: PromptVersion,
		DurationMS:    0,
	}
}

func (c ChatCompletionExtractor) Extract(ctx context.Context, req extract.ExtractSpecRequest) (extract.ExtractSpecResponse, error) {
	start := time.Now()

	body, err := json.Marshal(buildChatCompletionRequest(req, c.model, c.responseFormat))
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: marshal request: %v", ErrFailure, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: build request: %v", ErrFailure, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	log.Printf("[chatcompletion] %s: POST %s/chat/completions model=%s body_bytes=%d", req.RequestID, c.baseURL, c.model, len(body))
	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		if isTimeoutError(err) {
			log.Printf("[chatcompletion] %s: timeout after %dms: %v", req.RequestID, time.Since(start).Milliseconds(), err)
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		log.Printf("[chatcompletion] %s: request failed after %dms: %v", req.RequestID, time.Since(start).Milliseconds(), err)
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream request failed: %v", ErrFailure, err)
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: read response: %v", ErrFailure, err)
	}
	log.Printf("[chatcompletion] %s: response status=%d body_bytes=%d elapsed=%dms", req.RequestID, httpResp.StatusCode, len(responseBody), time.Since(start).Milliseconds())

	if httpResp.StatusCode >= 400 {
		if httpResp.StatusCode == http.StatusGatewayTimeout || httpResp.StatusCode == http.StatusRequestTimeout {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrTimeout, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
		}
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrFailure, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
	}

	var upstream chatCompletionResponse
	if err := json.Unmarshal(responseBody, &upstream); err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: decode response: %v", ErrFailure, err)
	}

	outputText := strings.TrimSpace(upstream.Content())
	log.Printf("[chatcompletion] %s: output_text=%q", req.RequestID, truncate(outputText, 200))
	if outputText == "" {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: missing structured output text", ErrFailure)
	}

	var output openAIExtractionOutput
	if err := json.Unmarshal([]byte(outputText), &output); err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: decode structured output: %v", ErrFailure, err)
	}

	meta := c.Meta()
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
			AvailableFinishes:   coalesceStrings(output.AvailableFinishes),
			FinishModelMappings: coalesceFinishMappings(output.FinishModelMappings),
			RequiredAddOns:      coalesceStrings(output.RequiredAddOns),
			OptionalCompanions:  coalesceStrings(output.OptionalCompanions),
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

func buildChatCompletionRequest(req extract.ExtractSpecRequest, model, responseFormat string) chatCompletionRequest {
	var format *chatResponseFormat
	systemPrompt := "You are Sally. Extract one architectural schedule proposal as strict JSON. Return valid JSON only, with no markdown or commentary. Prompt version: " + PromptVersion

	switch responseFormat {
	case "json_schema":
		format = &chatResponseFormat{
			Type: "json_schema",
			JSONSchema: &chatJSONSchema{
				Name:   "sally_extraction",
				Strict: true,
				Schema: extractionSchema(),
			},
		}
	default: // "json_object" or any unrecognised value — embed the field list so the model knows what to return
		format = &chatResponseFormat{Type: "json_object"}
		systemPrompt += "\n\nReturn a JSON object with exactly these fields:\n" +
			`{"title":"string","manufacturer":"string","modelNumber":"string","category":"string",` +
			`"description":"string","finish":"string","finishModelNumber":"string",` +
			`"availableFinishes":["string"],"finishModelMappings":[{"finish":"string","modelNumber":"string"}],` +
			`"requiredAddOns":["string"],"optionalCompanions":["string"],"zone":"string",` +
			`"analysis":{"missingFields":["string"],"warnings":["string"],` +
			`"confidence":{"overall":0,"title":0,"manufacturer":0,"modelNumber":0,"category":0,"description":0,"finish":0,"requiredAddOns":0}}}`
	}

	return chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildUserPrompt(req)},
		},
		ResponseFormat: format,
	}
}

type chatCompletionRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *chatJSONSchema `json:"json_schema,omitempty"`
}

type chatJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

func (r chatCompletionResponse) Content() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

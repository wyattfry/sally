package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sally/server/internal/extract"
)

type OllamaExtractor struct {
	model   string
	baseURL string
	client  httpDoer
}

func NewOllamaExtractor(model string, baseURL string, client httpDoer) OllamaExtractor {
	return OllamaExtractor{
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (o OllamaExtractor) Meta() extract.ResponseMeta {
	return extract.ResponseMeta{
		Provider:      "ollama",
		Model:         o.model,
		PromptVersion: PromptVersion,
		DurationMS:    0,
	}
}

func (o OllamaExtractor) Extract(ctx context.Context, req extract.ExtractSpecRequest) (extract.ExtractSpecResponse, error) {
	start := time.Now()

	body, err := json.Marshal(map[string]any{
		"model":  o.model,
		"prompt": buildOllamaPrompt(req),
		"stream": false,
		"format": extractionSchema(),
	})
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: marshal request: %v", ErrFailure, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: build request: %v", ErrFailure, err)
	}
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
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrFailure, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
	}

	var upstream struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(responseBody, &upstream); err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: decode response: %v", ErrFailure, err)
	}
	if strings.TrimSpace(upstream.Response) == "" {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: missing structured output text", ErrFailure)
	}

	var output openAIExtractionOutput
	if err := json.Unmarshal([]byte(upstream.Response), &output); err != nil {
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

func buildOllamaPrompt(req extract.ExtractSpecRequest) string {
	return strings.Join([]string{
		"You are Sally. Extract one architectural schedule proposal as strict JSON.",
		"Return valid JSON only, with no markdown or commentary.",
		"Prompt version: " + PromptVersion,
		buildUserPrompt(req),
	}, "\n\n")
}

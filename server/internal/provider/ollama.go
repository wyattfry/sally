package provider

import (
	"bufio"
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

	log.Printf("[ollama] %s: POST %s/api/generate model=%s body_bytes=%d", req.RequestID, o.baseURL, o.model, len(body))
	httpResp, err := o.client.Do(httpReq)
	if err != nil {
		if isTimeoutError(err) {
			log.Printf("[ollama] %s: timeout after %dms: %v", req.RequestID, time.Since(start).Milliseconds(), err)
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		log.Printf("[ollama] %s: request failed after %dms: %v", req.RequestID, time.Since(start).Milliseconds(), err)
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream request failed: %v", ErrFailure, err)
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: read response: %v", ErrFailure, err)
	}
	log.Printf("[ollama] %s: response status=%d body_bytes=%d elapsed=%dms", req.RequestID, httpResp.StatusCode, len(responseBody), time.Since(start).Milliseconds())
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

func (o OllamaExtractor) ExtractStreaming(ctx context.Context, req extract.ExtractSpecRequest, onProgress ProgressFunc) (extract.ExtractSpecResponse, error) {
	start := time.Now()

	body, err := json.Marshal(map[string]any{
		"model":  o.model,
		"prompt": buildOllamaPrompt(req),
		"stream": true,
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

	log.Printf("[ollama] %s: POST %s/api/generate model=%s (streaming)", req.RequestID, o.baseURL, o.model)
	httpResp, err := o.client.Do(httpReq)
	if err != nil {
		if isTimeoutError(err) {
			log.Printf("[ollama] %s: timeout after %dms: %v", req.RequestID, time.Since(start).Milliseconds(), err)
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		log.Printf("[ollama] %s: request failed after %dms: %v", req.RequestID, time.Since(start).Milliseconds(), err)
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream request failed: %v", ErrFailure, err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		responseBody, _ := io.ReadAll(httpResp.Body)
		if httpResp.StatusCode == http.StatusGatewayTimeout || httpResp.StatusCode == http.StatusRequestTimeout {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrTimeout, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
		}
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrFailure, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
	}

	var fullResponse strings.Builder
	chunkCount := 0

	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 512*1024), 512*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			log.Printf("[ollama] %s: skipping unparseable chunk: %v", req.RequestID, err)
			continue
		}

		if chunk.Response != "" {
			fullResponse.WriteString(chunk.Response)
			chunkCount++
			onProgress(chunkCount)
		}

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		if isTimeoutError(err) {
			log.Printf("[ollama] %s: streaming timeout after %dms", req.RequestID, time.Since(start).Milliseconds())
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: streaming timeout: %v", ErrTimeout, err)
		}
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: read streaming response: %v", ErrFailure, err)
	}

	log.Printf("[ollama] %s: streaming complete %d chunks elapsed=%dms", req.RequestID, chunkCount, time.Since(start).Milliseconds())

	responseText := strings.TrimSpace(fullResponse.String())
	if responseText == "" {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: missing structured output text", ErrFailure)
	}

	var output openAIExtractionOutput
	if err := json.Unmarshal([]byte(responseText), &output); err != nil {
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

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

	builtReq := buildChatCompletionRequest(req, c.model, c.responseFormat)
	body, err := json.Marshal(builtReq)
	if err != nil {
		return extract.ExtractSpecResponse{}, fmt.Errorf("%w: marshal request: %v", ErrFailure, err)
	}
	promptText := capLog(string(body))

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
		if httpResp.StatusCode == http.StatusTooManyRequests || httpResp.StatusCode == 529 {
			return extract.ExtractSpecResponse{}, fmt.Errorf("%w: upstream status %d: %s", ErrOverloaded, httpResp.StatusCode, summarizeUpstreamBody(responseBody))
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
	meta.PromptTokens = upstream.Usage.PromptTokens
	meta.CompletionTokens = upstream.Usage.CompletionTokens

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
			Room:                  output.Room,
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

// fewShotExample is embedded in the system prompt so both json_schema and
// json_object paths get the same calibration example.
const fewShotExample = `
EXAMPLE — use this as a calibration reference:
Product page title: Andersen 400 Series 36 in. x 48 in. Casement Window, White Exterior
Visible text excerpt: Andersen 400 Series. Casement window. White exterior and interior. Low-E4 SmartSun glass. Model C24. Rough opening: 36-3/8 in. W x 48-1/2 in. H. Overall frame: 35-3/8 in. W x 47-1/2 in. H. Available in White and Pine interior. Accessories: insect screen, grille options.
Specification document text excerpt: C24 CASEMENT WINDOW — Technical Specifications. Frame Material: Fibrex composite. Glass: Low-E4 SmartSun double-pane. Rough Opening Width: 36-3/8 in. Rough Opening Height: 48-1/2 in. Overall Jamb Width: 35-3/8 in. Overall Jamb Height: 47-1/2 in. Swing: Left hand.

Expected output (customFields populated from spec document, not left empty):
{"title":"400 Series Casement Window","manufacturer":"Andersen","modelNumber":"C24","category":"Window","description":"Casement window with Low-E4 SmartSun glass, Fibrex composite frame, white exterior and interior.","finish":"White","finishModelNumber":"","availableFinishes":["White","Pine"],"finishModelMappings":[],"requiredAddOns":[],"optionalCompanions":["Insect Screen","Grille"],"room":"","suggestedScheduleName":"Window Schedule","analysis":{"missingFields":[],"warnings":[],"confidence":{"overall":0.95,"title":0.99,"manufacturer":0.99,"modelNumber":0.95,"category":0.99,"description":0.9,"finish":0.99,"requiredAddOns":0.8}},"customFields":{"rough_opening":"36-3/8 in. W x 48-1/2 in. H","overall_jamb":"35-3/8 in. W x 47-1/2 in. H","swing":"Left hand"}}

Key rules illustrated:
- customFields are populated from spec document text, not left empty.
- room is a room name (e.g. "Kitchen") or empty — never XML or markup.
- suggestedScheduleName matches an existing schedule name exactly when the item fits, or is a short descriptive name.
- title omits the manufacturer name (it has its own field).
END EXAMPLE

PAINT-SPECIFIC RULES — when the product is paint:
- "finish" is the SHEEN, one of: Flat, Matte, Eggshell, Satin, Semi-Gloss, Hi-Gloss (or close equivalent). It is NEVER the color.
- Put the paint's color in customFields.color (e.g., "Ultra Pure White", "Spiced Beige", "Behr N250-2"). "color" is always available in customFields for paint, even if it is not listed as a column.
- "availableFinishes" lists sheens, not colors.

EXAMPLE 2 — paint product:
Product page title: BEHR MARQUEE 1 gal. #N250-2 Spiced Beige Eggshell Enamel Interior Paint and Primer
Visible text excerpt: BEHR MARQUEE. Color: N250-2 Spiced Beige. Sheen: Eggshell. Interior Paint and Primer. Stain-resistant. One-coat hide. Available in Flat, Eggshell, Satin, Semi-Gloss, Hi-Gloss.

Expected output (finish is the sheen, color is in customFields):
{"title":"MARQUEE Interior Paint and Primer","manufacturer":"Behr","modelNumber":"N250-2","category":"Paint","description":"Stain-resistant one-coat interior paint and primer with eggshell sheen.","finish":"Eggshell","finishModelNumber":"","availableFinishes":["Flat","Eggshell","Satin","Semi-Gloss","Hi-Gloss"],"finishModelMappings":[],"requiredAddOns":[],"optionalCompanions":[],"room":"","suggestedScheduleName":"Paint","analysis":{"missingFields":[],"warnings":[],"confidence":{"overall":0.95,"title":0.95,"manufacturer":0.99,"modelNumber":0.95,"category":0.99,"description":0.9,"finish":0.99,"requiredAddOns":0.9}},"customFields":{"color":"Spiced Beige"}}
END EXAMPLE 2

FINISH VARIANTS — when a product page lists multiple finishes/colors with distinct model numbers:
- Populate "availableFinishes" with every finish offered.
- Populate "finishModelMappings" with one entry per finish, pairing the finish to its specific SKU.
- Set "finish" to the currently-selected finish on the page (or the first listed if none is selected).
- Set "finishModelNumber" to the SKU matching that selected finish.

EXAMPLE 3 — product with multiple finishes:
Product page title: KOHLER Cardale Single Handle Pull-Down Kitchen Faucet
Visible text excerpt: KOHLER K-35908-4 Cardale Pull-Down Kitchen Faucet. Finish: Vibrant Brushed Nickel. Available finishes: Polished Chrome (K-35908-4-CP), Vibrant Polished Nickel (K-35908-4-SN), Vibrant Brushed Moderne Brass (K-35908-4-2MB), Vibrant Brushed Nickel (K-35908-4-BN).

Expected output (mappings populated):
{"title":"Cardale Pull-Down Kitchen Faucet","manufacturer":"Kohler","modelNumber":"K-35908-4","category":"Faucet","description":"Pull-down kitchen faucet with single-handle control.","finish":"Vibrant Brushed Nickel","finishModelNumber":"K-35908-4-BN","availableFinishes":["Polished Chrome","Vibrant Polished Nickel","Vibrant Brushed Moderne Brass","Vibrant Brushed Nickel"],"finishModelMappings":[{"finish":"Polished Chrome","modelNumber":"K-35908-4-CP"},{"finish":"Vibrant Polished Nickel","modelNumber":"K-35908-4-SN"},{"finish":"Vibrant Brushed Moderne Brass","modelNumber":"K-35908-4-2MB"},{"finish":"Vibrant Brushed Nickel","modelNumber":"K-35908-4-BN"}],"requiredAddOns":[],"optionalCompanions":[],"room":"","suggestedScheduleName":"Plumbing","analysis":{"missingFields":[],"warnings":[],"confidence":{"overall":0.95,"title":0.95,"manufacturer":0.99,"modelNumber":0.95,"category":0.99,"description":0.9,"finish":0.95,"requiredAddOns":0.9}},"customFields":{}}
END EXAMPLE 3`

func buildChatCompletionRequest(req extract.ExtractSpecRequest, model, responseFormat string) chatCompletionRequest {
	var format *chatResponseFormat
	systemPrompt := "You are Sally. Extract one architectural schedule proposal as strict JSON. Return valid JSON only, with no markdown or commentary. Prompt version: " + PromptVersion + fewShotExample

	switch responseFormat {
	case "json_schema":
		format = &chatResponseFormat{
			Type: "json_schema",
			JSONSchema: &chatJSONSchema{
				Name:   "sally_extraction",
				Strict: true,
				Schema: extractionSchema(req.CustomColumns),
			},
		}
	default: // "json_object" or any unrecognised value — embed the field list so the model knows what to return
		format = &chatResponseFormat{Type: "json_object"}
		systemPrompt += "\n\nReturn a JSON object with exactly these fields:\n" +
			`{"title":"string","manufacturer":"string","modelNumber":"string","category":"string",` +
			`"description":"string","finish":"string","finishModelNumber":"string",` +
			`"availableFinishes":["string"],"finishModelMappings":[{"finish":"string","modelNumber":"string"}],` +
			`"requiredAddOns":["string"],"optionalCompanions":["string"],"room":"string","suggestedScheduleName":"string",` +
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

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
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

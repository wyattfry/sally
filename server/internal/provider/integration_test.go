package provider

import (
	"bufio"
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestChatCompletionExtractorRealAPIHappyPath makes a real call to the configured LLM API.
// It skips unless OPENAI_API_KEY is set (either in environment or root .env file).
func TestChatCompletionExtractorRealAPIHappyPath(t *testing.T) {
	loadDotEnv(t, "../../../.env")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set — skipping real API integration test")
	}

	model := envOrDefault("OPENAI_MODEL", "llama-3.3-70b-versatile")
	baseURL := envOrDefault("OPENAI_BASE_URL", "https://api.groq.com/openai/v1")
	responseFormat := envOrDefault("CHATCOMPLETION_RESPONSE_FORMAT", "json_object")

	client := &http.Client{Timeout: 60 * time.Second}
	extractor := NewChatCompletionExtractor(apiKey, model, baseURL, responseFormat, client)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := extractor.Extract(ctx, validOpenAIRequest())
	if err != nil {
		t.Fatalf("real API call failed: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
	if resp.Proposal == nil {
		t.Fatal("expected non-nil proposal")
	}
	if resp.Proposal.Title == "" {
		t.Error("expected non-empty Title")
	}
	if resp.Proposal.Manufacturer == "" {
		t.Error("expected non-empty Manufacturer")
	}
	if resp.Proposal.ModelNumber == "" {
		t.Error("expected non-empty ModelNumber")
	}
	if resp.Meta.DurationMS <= 0 {
		t.Errorf("expected positive DurationMS, got %d", resp.Meta.DurationMS)
	}
	if resp.Meta.Provider != "chatcompletion" {
		t.Errorf("expected provider chatcompletion, got %q", resp.Meta.Provider)
	}
	if resp.Proposal.RequiredAddOns == nil {
		t.Error("expected non-nil RequiredAddOns (coalesced from null)")
	}
	if resp.Proposal.AvailableFinishes == nil {
		t.Error("expected non-nil AvailableFinishes (coalesced from null)")
	}

	t.Logf("title=%q manufacturer=%q modelNumber=%q durationMS=%d",
		resp.Proposal.Title, resp.Proposal.Manufacturer, resp.Proposal.ModelNumber, resp.Meta.DurationMS)
}

// loadDotEnv reads a .env file and sets any variables not already present in the environment.
func loadDotEnv(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return // missing .env is fine; env vars may already be set
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		os.Setenv(key, value)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

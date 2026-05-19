package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultPort = "8080"
const defaultOpenAIBaseURL = "https://api.openai.com/v1"
const defaultOpenAITimeout = 15 * time.Second
const defaultOllamaTimeout = 60 * time.Second

// Contractor view freshness thresholds (days). A price/lead-time snapshot
// older than the amber threshold shows in amber; older than the red threshold
// in red. Tunable via env so we can adjust after seeing real contractor use.
const defaultContractorStaleAmberDays = 30
const defaultContractorStaleRedDays = 90

type Config struct {
	Port                        string
	LLMProvider                 string
	OpenAIAPIKey                string
	OpenAIModel                 string
	OpenAIBaseURL               string
	OpenAITimeout               time.Duration
	ChatCompletionResponseFormat string
	OllamaBaseURL               string
	OllamaModel                 string
	OllamaTimeout               time.Duration
	AnthropicAPIKey             string
	AnthropicModel              string
	AllowMockFallback           bool
	DatabaseURL                 string
	GoogleClientID              string
	GoogleClientSecret          string
	GoogleRedirectURL           string
	SessionSecret               string
	UploadsDir                  string
	AdminEmail                  string

	// ContractorViewStaleAmberDays: a price snapshot older than this shows
	// in amber in the contractor view. Default 30. Env: SALLY_CONTRACTOR_STALE_AMBER_DAYS.
	ContractorViewStaleAmberDays int
	// ContractorViewStaleRedDays: same, in red. Default 90.
	// Env: SALLY_CONTRACTOR_STALE_RED_DAYS.
	ContractorViewStaleRedDays int
}

func Load() Config {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	return Config{
		Port:              port,
		LLMProvider:       strings.TrimSpace(strings.ToLower(os.Getenv("LLM_PROVIDER"))),
		OpenAIAPIKey:      strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIModel:       strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
		OpenAIBaseURL:     firstNonEmpty(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), defaultOpenAIBaseURL),
		OpenAITimeout:               parseDurationMillisEnv("OPENAI_TIMEOUT_MS", defaultOpenAITimeout),
		ChatCompletionResponseFormat: firstNonEmpty(strings.TrimSpace(os.Getenv("CHATCOMPLETION_RESPONSE_FORMAT")), "json_schema"),
		OllamaBaseURL:               strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")),
		OllamaModel:                 strings.TrimSpace(os.Getenv("OLLAMA_MODEL")),
		OllamaTimeout:               parseDurationMillisEnv("OLLAMA_TIMEOUT_MS", defaultOllamaTimeout),
		AnthropicAPIKey:             strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		AnthropicModel:              strings.TrimSpace(os.Getenv("ANTHROPIC_MODEL")),
		AllowMockFallback:           parseBoolEnv("SALLY_ALLOW_MOCK_FALLBACK"),
		DatabaseURL:                 strings.TrimSpace(os.Getenv("DATABASE_URL")),
		GoogleClientID:              strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret:          strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleRedirectURL:           strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL")),
		SessionSecret:               strings.TrimSpace(os.Getenv("SESSION_SECRET")),
		UploadsDir:                  firstNonEmpty(strings.TrimSpace(os.Getenv("UPLOADS_DIR")), "./uploads"),
		AdminEmail:                  strings.TrimSpace(os.Getenv("ADMIN_EMAIL")),
		ContractorViewStaleAmberDays: parseIntEnv("SALLY_CONTRACTOR_STALE_AMBER_DAYS", defaultContractorStaleAmberDays),
		ContractorViewStaleRedDays:   parseIntEnv("SALLY_CONTRACTOR_STALE_RED_DAYS", defaultContractorStaleRedDays),
	}
}

func parseIntEnv(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func parseBoolEnv(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseDurationMillisEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	millis, err := strconv.Atoi(value)
	if err != nil || millis <= 0 {
		return fallback
	}
	return time.Duration(millis) * time.Millisecond
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

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

type Config struct {
	Port              string
	LLMProvider       string
	OpenAIAPIKey      string
	OpenAIModel       string
	OpenAIBaseURL     string
	OpenAITimeout     time.Duration
	OllamaBaseURL     string
	OllamaModel       string
	AllowMockFallback bool
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
		OpenAITimeout:     parseDurationMillisEnv("OPENAI_TIMEOUT_MS", defaultOpenAITimeout),
		OllamaBaseURL:     strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")),
		OllamaModel:       strings.TrimSpace(os.Getenv("OLLAMA_MODEL")),
		AllowMockFallback: parseBoolEnv("SALLY_ALLOW_MOCK_FALLBACK"),
	}
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

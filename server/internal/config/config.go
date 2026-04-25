package config

import (
	"os"
	"strings"
)

const defaultPort = "8080"

type Config struct {
	Port              string
	OpenAIAPIKey      string
	AllowMockFallback bool
}

func Load() Config {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	return Config{
		Port:              port,
		OpenAIAPIKey:      strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
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

package main

import (
	"log"
	"net/http"

	"sally/server/internal/config"
	"sally/server/internal/httpapi"
	"sally/server/internal/provider"
)

func main() {
	cfg := config.Load()
	addr := ":" + cfg.Port
	extractor := newExtractor(cfg)

	server := &http.Server{
		Addr:    addr,
		Handler: httpapi.NewRouterWithExtractor(cfg, extractor),
	}

	log.Printf("sally server listening on %s provider=%s timeout=%s", addr, cfg.LLMProvider, cfg.OpenAITimeout)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func newExtractor(cfg config.Config) provider.Extractor {
	client := &http.Client{Timeout: cfg.OpenAITimeout}

	switch cfg.LLMProvider {
	case "", "stub":
		return provider.NewStubExtractor()
	case "openai":
		validateOpenAIConfig(cfg)
		return provider.NewOpenAIExtractor(
			cfg.OpenAIAPIKey,
			cfg.OpenAIModel,
			cfg.OpenAIBaseURL,
			client,
		)
	case "ollama":
		validateOllamaConfig(cfg)
		return provider.NewOllamaExtractor(
			cfg.OllamaModel,
			cfg.OllamaBaseURL,
			client,
		)
	default:
		log.Fatalf("unsupported LLM_PROVIDER %q", cfg.LLMProvider)
		return nil
	}
}

func validateOpenAIConfig(cfg config.Config) {
	if cfg.OpenAIAPIKey == "" || cfg.OpenAIModel == "" {
		log.Fatal("LLM_PROVIDER=openai requires OPENAI_API_KEY and OPENAI_MODEL")
	}
}

func validateOllamaConfig(cfg config.Config) {
	if cfg.OllamaBaseURL == "" || cfg.OllamaModel == "" {
		log.Fatal("LLM_PROVIDER=ollama requires OLLAMA_BASE_URL and OLLAMA_MODEL")
	}
}

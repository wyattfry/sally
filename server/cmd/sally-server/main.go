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
	validateOpenAIConfig(cfg)
	addr := ":" + cfg.Port
	extractor := provider.Extractor(provider.NewStubExtractor())
	if cfg.OpenAIAPIKey != "" && cfg.OpenAIModel != "" {
		extractor = provider.NewOpenAIExtractor(
			cfg.OpenAIAPIKey,
			cfg.OpenAIModel,
			cfg.OpenAIBaseURL,
			&http.Client{Timeout: cfg.OpenAITimeout},
		)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: httpapi.NewRouterWithExtractor(cfg, extractor),
	}

	log.Printf("sally server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func validateOpenAIConfig(cfg config.Config) {
	hasAPIKey := cfg.OpenAIAPIKey != ""
	hasModel := cfg.OpenAIModel != ""
	if hasAPIKey == hasModel {
		return
	}

	if hasAPIKey {
		log.Fatal("OPENAI_API_KEY is set but OPENAI_MODEL is missing")
	}
	log.Fatal("OPENAI_MODEL is set but OPENAI_API_KEY is missing")
}

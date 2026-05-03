package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"sally/server/internal/config"
	appdb "sally/server/internal/db"
	"sally/server/internal/httpapi"
	"sally/server/internal/provider"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load()
	addr := ":" + cfg.Port
	extractor := newExtractor(cfg)
	database := openDatabase(cfg)
	if database != nil {
		defer database.Close()
	}

	server := &http.Server{
		Addr:    addr,
		Handler: httpapi.NewRouterWithExtractor(cfg, extractor),
	}

	log.Printf("sally server listening on %s provider=%s timeout=%s", addr, cfg.LLMProvider, cfg.OpenAITimeout)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func openDatabase(cfg config.Config) *sql.DB {
	if cfg.DatabaseURL == "" {
		return nil
	}

	database, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := appdb.RunMigrations(context.Background(), database, "migrations"); err != nil {
		_ = database.Close()
		log.Fatalf("run database migrations: %v", err)
	}
	log.Printf("database connected and migrated")
	return database
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
	case "chatcompletion":
		validateChatCompletionConfig(cfg)
		return provider.NewChatCompletionExtractor(
			cfg.OpenAIAPIKey,
			cfg.OpenAIModel,
			cfg.OpenAIBaseURL,
			cfg.ChatCompletionResponseFormat,
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

func validateChatCompletionConfig(cfg config.Config) {
	if cfg.OpenAIAPIKey == "" || cfg.OpenAIModel == "" {
		log.Fatal("LLM_PROVIDER=chatcompletion requires OPENAI_API_KEY and OPENAI_MODEL")
	}
}

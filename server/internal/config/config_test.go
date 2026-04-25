package config

import "testing"

func TestLoadDefaultsPortWhenUnset(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("SALLY_ALLOW_MOCK_FALLBACK", "")

	cfg := Load()

	if cfg.Port != defaultPort {
		t.Fatalf("expected default port %q, got %q", defaultPort, cfg.Port)
	}
	if cfg.AllowMockFallback {
		t.Fatal("expected mock fallback to default to false")
	}
}

func TestLoadParsesAllowMockFallback(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("SALLY_ALLOW_MOCK_FALLBACK", "true")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Fatalf("expected port %q, got %q", "9090", cfg.Port)
	}
	if !cfg.AllowMockFallback {
		t.Fatal("expected mock fallback to parse as true")
	}
}

func TestLoadParsesLLMProviderAndOllamaSettings(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "ollama")
	t.Setenv("OLLAMA_BASE_URL", "http://10.0.0.200:11434")
	t.Setenv("OLLAMA_MODEL", "qwen2.5:7b")

	cfg := Load()

	if cfg.LLMProvider != "ollama" {
		t.Fatalf("expected llm provider %q, got %q", "ollama", cfg.LLMProvider)
	}
	if cfg.OllamaBaseURL != "http://10.0.0.200:11434" {
		t.Fatalf("expected ollama base url to parse, got %q", cfg.OllamaBaseURL)
	}
	if cfg.OllamaModel != "qwen2.5:7b" {
		t.Fatalf("expected ollama model to parse, got %q", cfg.OllamaModel)
	}
}

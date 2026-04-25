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

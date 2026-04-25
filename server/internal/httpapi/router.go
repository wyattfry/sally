package httpapi

import (
	"net/http"

	"sally/server/internal/config"
	"sally/server/internal/provider"
)

func NewRouter(cfg config.Config) http.Handler {
	return NewRouterWithExtractor(cfg, provider.NewStubExtractor())
}

func NewRouterWithExtractor(cfg config.Config, extractor provider.Extractor) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /v1/extract-spec", NewExtractHandler(extractor))

	_ = cfg

	return mux
}

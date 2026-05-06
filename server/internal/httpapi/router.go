package httpapi

import (
	"net/http"
	"strings"

	"sally/server/internal/config"
	"sally/server/internal/provider"
	"sally/server/internal/web"
)

func NewRouter(cfg config.Config) http.Handler {
	return NewRouterWithExtractor(cfg, provider.NewStubExtractor())
}

func NewRouterWithExtractor(cfg config.Config, extractor provider.Extractor) http.Handler {
	return NewRouterWithDeps(cfg, extractor, web.Deps{})
}

func NewRouterWithDeps(cfg config.Config, extractor provider.Extractor, webDeps web.Deps) http.Handler {
	mux := http.NewServeMux()

	if webDeps.Queries != nil {
		web.RegisterRoutes(mux, webDeps)
		registerMothershipAPI(mux, webDeps)
	}

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	extractHandler := NewExtractHandler(extractor, webDeps.Queries, webDeps.Queries)
	mux.HandleFunc("POST /api/v1/extract-spec", extractHandler)
	mux.HandleFunc("POST /v1/extract-spec", extractHandler)

	_ = cfg

	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			requestHeaders := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers"))
			if requestHeaders == "" {
				requestHeaders = "Content-Type, X-Session-Token"
			}
			w.Header().Set("Access-Control-Allow-Headers", requestHeaders)
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

package httpapi

import (
	"net/http"

	"sally/server/internal/config"
)

func NewRouter(cfg config.Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /v1/extract-spec", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not implemented", http.StatusNotImplemented)
	})

	_ = cfg

	return mux
}

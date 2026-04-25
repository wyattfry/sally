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
	extractor := provider.NewStubExtractor()

	server := &http.Server{
		Addr:    addr,
		Handler: httpapi.NewRouterWithExtractor(cfg, extractor),
	}

	log.Printf("sally server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

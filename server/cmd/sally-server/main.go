package main

import (
	"log"
	"net/http"

	"sally/server/internal/config"
	"sally/server/internal/httpapi"
)

func main() {
	cfg := config.Load()
	addr := ":" + cfg.Port

	server := &http.Server{
		Addr:    addr,
		Handler: httpapi.NewRouter(cfg),
	}

	log.Printf("sally server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

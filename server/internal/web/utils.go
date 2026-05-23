package web

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func render(w http.ResponseWriter, data any) {
	// Render to a buffer first so a template error mid-render doesn't leave
	// the response in a half-written state ("superfluous WriteHeader" if we
	// then try to call http.Error). Only flush to the wire on success.
	var buf bytes.Buffer
	if err := pageTemplate.ExecuteTemplate(&buf, "page.html", data); err != nil {
		log.Printf("render: template execute: %v", err)
		http.Error(w, "could not render page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

func renderNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_ = pageTemplate.ExecuteTemplate(w, "page.html", notFoundPage{Kind: "not-found", Title: "Page not found"})
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func firstPositive(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func requestBaseURL(r *http.Request) string {
	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

func splitLines(value string) []string {
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseInt32(value string, fallback int32) int32 {
	var parsed int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed); err != nil {
		return fallback
	}
	return int32(parsed)
}

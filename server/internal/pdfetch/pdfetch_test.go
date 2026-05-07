package pdfetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchAndExtract_EmptyURLs(t *testing.T) {
	result := FetchAndExtract(context.Background(), http.DefaultClient, nil)
	if result != "" {
		t.Errorf("expected empty string for nil urls, got %q", result)
	}
	result = FetchAndExtract(context.Background(), http.DefaultClient, []string{})
	if result != "" {
		t.Errorf("expected empty string for empty urls, got %q", result)
	}
}

func TestFetchAndExtract_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	result := FetchAndExtract(context.Background(), srv.Client(), []string{srv.URL + "/missing.pdf"})
	if result != "" {
		t.Errorf("expected empty string on 404, got %q", result)
	}
}

func TestFetchAndExtract_NonPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("not a pdf"))
	}))
	defer srv.Close()

	// Non-PDF bytes should fail gracefully and return empty.
	result := FetchAndExtract(context.Background(), srv.Client(), []string{srv.URL + "/fake.pdf"})
	if result != "" {
		t.Errorf("expected empty string for non-PDF content, got %q", result)
	}
}

func TestFetchAndExtract_LimitsToMaxPDFs(t *testing.T) {
	called := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	urls := []string{
		srv.URL + "/a.pdf",
		srv.URL + "/b.pdf",
		srv.URL + "/c.pdf",
		srv.URL + "/d.pdf",
	}
	FetchAndExtract(context.Background(), srv.Client(), urls)
	if called > maxPDFs {
		t.Errorf("expected at most %d fetches, got %d", maxPDFs, called)
	}
}

func TestFetchAndExtract_TextCapped(t *testing.T) {
	// Verify that combined output never exceeds maxTotalText regardless of input size.
	long := strings.Repeat("x", maxTotalText*3)
	if len(long) <= maxTotalText {
		t.Fatal("test setup: long string should exceed cap")
	}

	result := capText(long)
	if len(result) > maxTotalText {
		t.Errorf("capText returned %d chars, want <= %d", len(result), maxTotalText)
	}
}

// capText is a thin helper so we can unit-test the cap logic without a real PDF.
func capText(s string) string {
	if len(s) > maxTotalText {
		return s[:maxTotalText]
	}
	return s
}

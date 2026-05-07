// Package pdfetch fetches PDF URLs and extracts their plain text for use in
// LLM prompts. It is intentionally best-effort: any fetch or parse failure
// is silently skipped so that a bad PDF never blocks an extraction.
package pdfetch

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

const (
	maxPDFs         = 2
	maxDownloadSize = 5 << 20  // 5 MB per PDF
	maxTextPerPDF   = 4_000    // chars
	maxTotalText    = 8_000    // chars combined
	fetchTimeout    = 10 * time.Second
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// FetchAndExtract fetches up to maxPDFs from urls, extracts plain text from
// each, and returns the combined result capped at maxTotalText characters.
// Any individual failure is logged and skipped.
func FetchAndExtract(ctx context.Context, client HTTPClient, urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	var parts []string
	total := 0
	attempted := 0
	for _, u := range urls {
		if attempted >= maxPDFs {
			break
		}
		if total >= maxTotalText {
			break
		}
		attempted++
		text := fetchOne(ctx, client, u)
		if text == "" {
			continue
		}
		if len(text) > maxTextPerPDF {
			text = text[:maxTextPerPDF]
		}
		parts = append(parts, text)
		total += len(text)
	}
	combined := strings.Join(parts, "\n\n")
	if len(combined) > maxTotalText {
		combined = combined[:maxTotalText]
	}
	return combined
}

func fetchOne(ctx context.Context, client HTTPClient, url string) string {
	fCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fCtx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[pdfetch] build request %s: %v", url, err)
		return ""
	}
	req.Header.Set("User-Agent", "Sally/1.0 (spec-extraction)")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[pdfetch] fetch %s: %v", url, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[pdfetch] fetch %s: status %d", url, resp.StatusCode)
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize))
	if err != nil {
		log.Printf("[pdfetch] read %s: %v", url, err)
		return ""
	}

	text, err := extractText(body)
	if err != nil {
		log.Printf("[pdfetch] parse %s: %v", url, err)
		return ""
	}
	return text
}

// extractText parses PDF bytes and returns the plain text content.
func extractText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		content, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		b.WriteString(content)
		if b.Len() >= maxTextPerPDF {
			break
		}
	}
	return strings.TrimSpace(b.String()), nil
}

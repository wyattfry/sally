package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	queries "sally/server/internal/db/generated"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed templates/page.html templates/partials/*.html templates/pages/*.html
var templatesFS embed.FS

var pageTemplate = template.Must(template.New("page").Funcs(template.FuncMap{
	// Stub implementations replaced per-request in render() via Clone+Funcs.
	"currentUser":     func() *queries.User { return nil },
	"userInitials":    func() string { return "" },
	"userDisplayName": func() string { return "" },
	"add":         func(a, b int) int { return a + b },
	// sumItemCounts totals ItemCount across all schedules. Used by the
	// new-user idle-hint to decide whether to show the toast.
	"sumItemCounts": func(schedules []scheduleSummary) int {
		n := 0
		for _, s := range schedules {
			n += s.ItemCount
		}
		return n
	},
	"grandTotalDisplay": func(schedules []scheduleSummary) string {
		var cents int64
		var hasAny bool
		for _, s := range schedules {
			if s.ContractorTotals != nil && s.ContractorTotals.SubtotalCents > 0 {
				cents += s.ContractorTotals.SubtotalCents
				hasAny = true
			}
		}
		if !hasAny {
			return ""
		}
		return formatCents(cents)
	},
	"formatBytes": FormatBytes,
	"pct": func(num, den int64) string {
		if den == 0 {
			return "—"
		}
		return fmt.Sprintf("%.0f%%", float64(num)/float64(den)*100)
	},
	"nl2br": func(s string) template.HTML {
		escaped := template.HTMLEscapeString(s)
		escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		return template.HTML(escaped)
	},
	"prettyJSON": func(s string) template.HTML {
		var v any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			return template.HTML(template.HTMLEscapeString(s))
		}
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return template.HTML(template.HTMLEscapeString(s))
		}
		return template.HTML("<pre class=\"admin-json\">" + template.HTMLEscapeString(string(b)) + "</pre>")
	},
	// extractPromptMessages unwraps a saved provider request JSON to show
	// just the human-readable prompt content: the `system` string and each
	// message's content[].text. Returns the original string if it doesn't
	// parse as an object so callers still see something useful.
	"extractPromptMessages": func(s string) template.HTML {
		var req struct {
			System   any `json:"system"`
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal([]byte(s), &req); err != nil {
			return template.HTML(template.HTMLEscapeString(s))
		}
		var b strings.Builder
		writeBlock := func(label, body string) {
			body = strings.TrimSpace(body)
			if body == "" {
				return
			}
			b.WriteString("<div class=\"admin-prompt-block\"><h4>")
			b.WriteString(template.HTMLEscapeString(label))
			b.WriteString("</h4><pre class=\"admin-prompt-text\">")
			b.WriteString(template.HTMLEscapeString(body))
			b.WriteString("</pre></div>")
		}
		// System can be a string or array of blocks.
		switch sys := req.System.(type) {
		case string:
			writeBlock("System", sys)
		case []any:
			var parts []string
			for _, blk := range sys {
				if m, ok := blk.(map[string]any); ok {
					if t, _ := m["text"].(string); t != "" {
						parts = append(parts, t)
					}
				}
			}
			writeBlock("System", strings.Join(parts, "\n"))
		}
		for _, msg := range req.Messages {
			label := cases.Title(language.AmericanEnglish).String(msg.Role)
			if label == "" {
				label = "Message"
			}
			var parts []string
			switch c := msg.Content.(type) {
			case string:
				parts = append(parts, c)
			case []any:
				for _, blk := range c {
					if m, ok := blk.(map[string]any); ok {
						if t, _ := m["text"].(string); t != "" {
							parts = append(parts, t)
						}
					}
				}
			}
			writeBlock(label, strings.Join(parts, "\n"))
		}
		return template.HTML(b.String())
	},
	"isoTime": func(t time.Time) string { return t.UTC().Format(time.RFC3339) },
	// freshnessClass: green / amber / red CSS modifier based on how long ago
	// a price snapshot was captured. Drives the small chip under the price.
	"freshnessClass": func(pricedAt string, amberDays, redDays int) string {
		t, err := time.Parse(time.RFC3339, pricedAt)
		if err != nil {
			return "freshness--unknown"
		}
		age := time.Since(t)
		if redDays > 0 && age > time.Duration(redDays)*24*time.Hour {
			return "freshness--red"
		}
		if amberDays > 0 && age > time.Duration(amberDays)*24*time.Hour {
			return "freshness--amber"
		}
		return "freshness--fresh"
	},
	"freshnessLabel": func(pricedAt string) string {
		t, err := time.Parse(time.RFC3339, pricedAt)
		if err != nil {
			return ""
		}
		days := int(time.Since(t).Hours() / 24)
		switch {
		case days <= 0:
			return "priced today"
		case days == 1:
			return "priced 1d ago"
		default:
			return fmt.Sprintf("priced %dd ago", days)
		}
	},
	"stockLabel": func(status string) string {
		switch status {
		case "in_stock":
			return "In stock"
		case "low_stock":
			return "Low stock"
		case "backordered":
			return "Backordered"
		case "out_of_stock":
			return "Out of stock"
		default:
			return "Unknown"
		}
	},
	"humanTime": func(t time.Time) string {
		now := time.Now()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		tStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		days := int(todayStart.Sub(tStart).Hours() / 24)
		switch {
		case days == 0:
			return "Today at " + t.Format("3:04 PM")
		case days == 1:
			return "Yesterday at " + t.Format("3:04 PM")
		case days < 7:
			return fmt.Sprintf("%d days ago", days)
		case t.Year() == now.Year():
			return t.Format("Jan 2")
		default:
			return t.Format("Jan 2, 2006")
		}
	},
}).ParseFS(templatesFS, "templates/page.html", "templates/partials/*.html", "templates/pages/*.html"))

package web

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed templates/page.html
var pageTemplateHTML string

var pageTemplate = template.Must(template.New("page").Funcs(template.FuncMap{
	"add":         func(a, b int) int { return a + b },
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
			label := strings.Title(msg.Role)
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
}).Parse(pageTemplateHTML))

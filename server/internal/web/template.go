package web

import (
	_ "embed"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed templates/page.html
var pageTemplateHTML string

var pageTemplate = template.Must(template.New("page").Funcs(template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"nl2br": func(s string) template.HTML {
		escaped := template.HTMLEscapeString(s)
		escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		return template.HTML(escaped)
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

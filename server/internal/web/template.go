package web

import (
	_ "embed"
	"html/template"
)

//go:embed templates/page.html
var pageTemplateHTML string

var pageTemplate = template.Must(template.New("page").Funcs(template.FuncMap{
	"add": func(a, b int) int { return a + b },
}).Parse(pageTemplateHTML))

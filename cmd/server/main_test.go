package main

import (
	"html/template"
	"path/filepath"
	"testing"
)

func TestTemplatesParse(t *testing.T) {
	funcMap := template.FuncMap{
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"lt": func(a, b float64) bool { return a < b },
	}

	pattern := filepath.Join("..", "..", "web", "templates", "*.html")
	if _, err := template.New("").Funcs(funcMap).ParseGlob(pattern); err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}
}

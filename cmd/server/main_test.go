package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
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
		"asset": func(path string) string {
			trimmed := strings.TrimPrefix(path, "/")
			diskPath := trimmed
			if strings.HasPrefix(trimmed, "static/") {
				diskPath = filepath.Join("..", "..", "web", trimmed)
			}
			info, err := os.Stat(diskPath)
			if err != nil {
				return path
			}
			return fmt.Sprintf("%s?v=%d", path, info.ModTime().Unix())
		},
	}

	pattern := filepath.Join("..", "..", "web", "templates", "*.html")
	if _, err := template.New("").Funcs(funcMap).ParseGlob(pattern); err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}
}

package main

import (
	"path/filepath"
	"testing"
)

func TestTemplatesParse(t *testing.T) {
	pattern := filepath.Join("..", "..", "web", "templates", "*.html")
	webRoot := filepath.Join("..", "..", "web")
	if _, err := loadTemplates(pattern, webRoot); err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}
}

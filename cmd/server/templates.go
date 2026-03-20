package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

func templateFuncMap(webRoot string) template.FuncMap {
	return template.FuncMap{
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
				diskPath = filepath.Join(webRoot, trimmed)
			}
			info, err := os.Stat(diskPath)
			if err != nil {
				return path
			}
			return fmt.Sprintf("%s?v=%d", path, info.ModTime().Unix())
		},
	}
}

func loadTemplates(pattern, webRoot string) (*template.Template, error) {
	return template.New("").Funcs(templateFuncMap(webRoot)).ParseGlob(pattern)
}

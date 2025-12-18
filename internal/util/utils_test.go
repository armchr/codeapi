package util

import (
	"github.com/armchr/codeapi/internal/config"
	"testing"
)

func TestIsLanguageMatch(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		language string
		expected bool
	}{
		// Go
		{"Go file", "/path/to/file.go", "go", true},
		{"Go file uppercase", "/path/to/file.GO", "go", true},

		// Python
		{"Python file", "/path/to/file.py", "python", true},
		{"Python interface file", "/path/to/file.pyi", "python", true},
		{"Python extension", "/path/to/file.pyx", "python", true},

		// JavaScript variants
		{"JavaScript file", "/path/to/file.js", "javascript", true},
		{"JSX file", "/path/to/component.jsx", "javascript", true},
		{"MJS file", "/path/to/module.mjs", "javascript", true},
		{"CJS file", "/path/to/common.cjs", "javascript", true},

		// TypeScript variants
		{"TypeScript file", "/path/to/file.ts", "typescript", true},
		{"TSX file", "/path/to/component.tsx", "typescript", true},
		{"MTS file", "/path/to/module.mts", "typescript", true},
		{"CTS file", "/path/to/common.cts", "typescript", true},

		// CSS variants
		{"CSS file", "/path/to/style.css", "css", true},
		{"SCSS file", "/path/to/style.scss", "css", true},
		{"SASS file", "/path/to/style.sass", "css", true},
		{"LESS file", "/path/to/style.less", "css", true},

		// Negative cases
		{"Wrong language", "/path/to/file.py", "go", false},
		{"No extension", "/path/to/README", "go", false},
		{"Different extension", "/path/to/file.txt", "go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLanguageMatch(tt.filePath, tt.language)
			if result != tt.expected {
				t.Errorf("isLanguageMatch(%q, %q) = %v, want %v", tt.filePath, tt.language, result, tt.expected)
			}
		})
	}
}

func TestShouldSkipFile_WithLanguageFilter(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		repo         *config.Repository
		shouldSkip   bool
		description  string
	}{
		{
			name:     "Go repo, Go file, skip enabled",
			filePath: "/repo/main.go",
			repo: &config.Repository{
				Language:           "go",
				SkipOtherLanguages: true,
			},
			shouldSkip:  false,
			description: "Should process Go files in Go repo",
		},
		{
			name:     "Go repo, Python file, skip enabled",
			filePath: "/repo/script.py",
			repo: &config.Repository{
				Language:           "go",
				SkipOtherLanguages: true,
			},
			shouldSkip:  true,
			description: "Should skip Python files in Go repo when SkipOtherLanguages is true",
		},
		{
			name:     "Go repo, Python file, skip disabled",
			filePath: "/repo/script.py",
			repo: &config.Repository{
				Language:           "go",
				SkipOtherLanguages: false,
			},
			shouldSkip:  false,
			description: "Should process Python files in Go repo when SkipOtherLanguages is false",
		},
		{
			name:     "JavaScript repo, JSX file, skip enabled",
			filePath: "/repo/Component.jsx",
			repo: &config.Repository{
				Language:           "javascript",
				SkipOtherLanguages: true,
			},
			shouldSkip:  false,
			description: "Should process JSX files in JavaScript repo (variant)",
		},
		{
			name:     "TypeScript repo, TSX file, skip enabled",
			filePath: "/repo/Component.tsx",
			repo: &config.Repository{
				Language:           "typescript",
				SkipOtherLanguages: true,
			},
			shouldSkip:  false,
			description: "Should process TSX files in TypeScript repo (variant)",
		},
		{
			name:        "No repo config",
			filePath:    "/repo/main.go",
			repo:        nil,
			shouldSkip:  false,
			description: "Should process all files when repo is nil",
		},
		{
			name:     "Dockerfile always skipped",
			filePath: "/repo/Dockerfile",
			repo: &config.Repository{
				Language:           "go",
				SkipOtherLanguages: true,
			},
			shouldSkip:  true,
			description: "Should skip Dockerfile regardless of language settings",
		},
		{
			name:     "bin directory file skipped",
			filePath: "/repo/bin/bot-go",
			repo: &config.Repository{
				Language:           "go",
				SkipOtherLanguages: true,
			},
			shouldSkip:  true,
			description: "Should skip files in bin directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipFile(tt.filePath, tt.repo)
			if result != tt.shouldSkip {
				t.Errorf("ShouldSkipFile(%q, repo) = %v, want %v - %s", tt.filePath, result, tt.shouldSkip, tt.description)
			}
		})
	}
}

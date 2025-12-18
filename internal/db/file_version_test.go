package db

import (
	"testing"
)

func TestSanitizeTableName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "hyphen replacement",
			input:    "bot-go",
			expected: "bot_go",
		},
		{
			name:     "multiple hyphens",
			input:    "my-repo-name",
			expected: "my_repo_name",
		},
		{
			name:     "spaces and special chars",
			input:    "my repo!@#$name",
			expected: "my_repo_name",
		},
		{
			name:     "leading and trailing special chars",
			input:    "-bot-go-",
			expected: "bot_go",
		},
		{
			name:     "multiple consecutive special chars",
			input:    "bot---go___test",
			expected: "bot_go_test",
		},
		{
			name:     "already valid",
			input:    "bot_go",
			expected: "bot_go",
		},
		{
			name:     "alphanumeric only",
			input:    "botgo123",
			expected: "botgo123",
		},
		{
			name:     "mixed case preserved",
			input:    "BotGo-Test",
			expected: "BotGo_Test",
		},
		{
			name:     "dots and slashes",
			input:    "my.repo/name",
			expected: "my_repo_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeTableName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeTableName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

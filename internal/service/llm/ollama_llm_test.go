package llm

import "testing"

func TestExtractThinkingContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty response",
			input:    "",
			expected: "",
		},
		{
			name:     "no thinking tags",
			input:    "This is a simple response",
			expected: "This is a simple response",
		},
		{
			name:     "content after think tags",
			input:    "<think>Some reasoning here</think>The actual answer",
			expected: "The actual answer",
		},
		{
			name:     "content after think tags with whitespace",
			input:    "<think>Some reasoning here</think>\n\nThe actual answer\n",
			expected: "The actual answer",
		},
		{
			name:     "only thinking content - fallback to think content",
			input:    "<think>This is all the reasoning with no answer after</think>",
			expected: "This is all the reasoning with no answer after",
		},
		{
			name:     "unclosed think tag",
			input:    "<think>Still thinking about this...",
			expected: "Still thinking about this...",
		},
		{
			name:     "multiline thinking with answer",
			input:    "<think>\nFirst I'll analyze the code.\nThen I'll summarize it.\n</think>\nONE_LINE: This function does X.\nDESCRIPTION: More details.",
			expected: "ONE_LINE: This function does X.\nDESCRIPTION: More details.",
		},
		{
			name:     "whitespace only response",
			input:    "   \n\t  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractThinkingContent(tt.input)
			if result != tt.expected {
				t.Errorf("extractThinkingContent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanThinkingTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no tags",
			input:    "Clean content",
			expected: "Clean content",
		},
		{
			name:     "remove opening tag",
			input:    "<think>Some content",
			expected: "Some content",
		},
		{
			name:     "remove closing tag",
			input:    "Some content</think>",
			expected: "Some content",
		},
		{
			name:     "remove both tags",
			input:    "<think>Some content</think>",
			expected: "Some content",
		},
		{
			name:     "multiple tags",
			input:    "<think>First</think>Middle<think>Second</think>",
			expected: "FirstMiddleSecond",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanThinkingTags(tt.input)
			if result != tt.expected {
				t.Errorf("cleanThinkingTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

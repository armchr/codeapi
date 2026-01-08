package base

import "testing"

func TestExtractJavaMethodName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "method with params and return type",
			input:    "findByOwnerId(Long) : List<PetDto>",
			expected: "findByOwnerId",
		},
		{
			name:     "method with primitive return type",
			input:    "countByType(String) : long",
			expected: "countByType",
		},
		{
			name:     "method with generic return type",
			input:    "findAll() : List<PetDto>",
			expected: "findAll",
		},
		{
			name:     "method with no params",
			input:    "toString()",
			expected: "toString",
		},
		{
			name:     "simple method name without signature",
			input:    "simpleMethod",
			expected: "simpleMethod",
		},
		{
			name:     "method with multiple params",
			input:    "update(Long, PetDto) : PetDto",
			expected: "update",
		},
		{
			name:     "method with nested generics",
			input:    "searchByNameAsync(String) : CompletableFuture<List<PetDto>>",
			expected: "searchByNameAsync",
		},
		{
			name:     "constructor-like name",
			input:    "PetController(PetService)",
			expected: "PetController",
		},
		{
			name:     "method name with return type prefix (edge case)",
			input:    "Optional<User> findById(Long)",
			expected: "findById",
		},
		{
			name:     "void return with space",
			input:    "delete(Long) : void",
			expected: "delete",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractJavaMethodName(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractJavaMethodName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

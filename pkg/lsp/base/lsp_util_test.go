package base

import "testing"

func TestLastSegment(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"dotted name", "com.example.MyClass.Method", "Method"},
		{"single segment", "Method", "Method"},
		{"two segments", "Class.Method", "Method"},
		{"empty string", "", ""},
		{"trailing dot", "pkg.", ""},
		{"leading dot", ".Method", "Method"},
		{"multiple dots", "a.b.c.d.e", "e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LastSegment(tt.input)
			if got != tt.want {
				t.Errorf("LastSegment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchLastSegment(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		nameInFile string
		delim      string
		want       bool
	}{
		{"exact match", "Method", "com.example.MyClass.Method", ".", true},
		{"no match", "Method", "OtherMethod", ".", false},
		{"empty symbol", "", "Something", ".", false},
		{"empty nameInFile", "Method", "", ".", false},
		{"both empty", "", "", ".", true},
		{"single segment match", "Method", "Method", ".", true},
		{"case sensitive no match", "method", "Method", ".", false},
		{"qualified vs qualified", "pkg.Class.Method", "other.Class.Method", ".", true},
		{"different last segments", "pkg.MethodA", "pkg.MethodB", ".", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchLastSegment(tt.symbol, tt.nameInFile, tt.delim)
			if got != tt.want {
				t.Errorf("MatchLastSegment(%q, %q, %q) = %v, want %v",
					tt.symbol, tt.nameInFile, tt.delim, got, tt.want)
			}
		})
	}
}

func TestMatchExact(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		nameInFile string
		want       bool
	}{
		{"exact match", "Method", "Method", true},
		{"no match", "Method", "OtherMethod", false},
		{"case sensitive", "method", "Method", false},
		{"empty strings", "", "", true},
		{"one empty", "Method", "", false},
		{"with dots", "pkg.Class.Method", "pkg.Class.Method", true},
		{"unicode", "méthodé", "méthodé", true},
		{"unicode mismatch", "méthodé", "methode", false},
		{"whitespace matters", "Method ", "Method", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchExact(tt.symbol, tt.nameInFile)
			if got != tt.want {
				t.Errorf("MatchExact(%q, %q) = %v, want %v",
					tt.symbol, tt.nameInFile, got, tt.want)
			}
		})
	}
}

func TestMatchIgnoreCase(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		nameInFile string
		want       bool
	}{
		{"same case", "Method", "Method", true},
		{"different case", "method", "METHOD", true},
		{"mixed case", "mEtHoD", "MeThOd", true},
		{"no match", "Method", "Other", false},
		{"empty strings", "", "", true},
		{"one empty", "Method", "", false},
		{"with dots case insensitive", "Pkg.Class.Method", "pkg.class.method", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchIgnoreCase(tt.symbol, tt.nameInFile)
			if got != tt.want {
				t.Errorf("MatchIgnoreCase(%q, %q) = %v, want %v",
					tt.symbol, tt.nameInFile, got, tt.want)
			}
		})
	}
}

func TestMatchIgnoreCaseLastSegment(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		nameInFile string
		delim      string
		want       bool
	}{
		{"same case", "pkg.Method", "other.Method", ".", true},
		{"different case", "pkg.method", "other.METHOD", ".", true},
		{"no match", "pkg.MethodA", "other.MethodB", ".", false},
		{"single segment", "Method", "method", ".", true},
		{"empty strings", "", "", ".", true},
		{"mixed case last segment", "pkg.MyMethod", "other.mymethod", ".", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchIgnoreCaseLastSegment(tt.symbol, tt.nameInFile, tt.delim)
			if got != tt.want {
				t.Errorf("MatchIgnoreCaseLastSegment(%q, %q, %q) = %v, want %v",
					tt.symbol, tt.nameInFile, tt.delim, got, tt.want)
			}
		})
	}
}

func TestRangeInRange(t *testing.T) {
	tests := []struct {
		name  string
		outer Range
		inner Range
		want  bool
	}{
		{
			name:  "inner fully contained",
			outer: Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 10, Character: 0}},
			inner: Range{Start: Position{Line: 2, Character: 0}, End: Position{Line: 5, Character: 0}},
			want:  true,
		},
		{
			name:  "inner equals outer",
			outer: Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 10, Character: 0}},
			inner: Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 10, Character: 0}},
			want:  true,
		},
		{
			name:  "inner starts before outer",
			outer: Range{Start: Position{Line: 5, Character: 0}, End: Position{Line: 10, Character: 0}},
			inner: Range{Start: Position{Line: 2, Character: 0}, End: Position{Line: 8, Character: 0}},
			want:  false,
		},
		{
			name:  "inner ends after outer",
			outer: Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 10, Character: 0}},
			inner: Range{Start: Position{Line: 5, Character: 0}, End: Position{Line: 15, Character: 0}},
			want:  false,
		},
		{
			name:  "inner completely outside",
			outer: Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 5, Character: 0}},
			inner: Range{Start: Position{Line: 10, Character: 0}, End: Position{Line: 15, Character: 0}},
			want:  false,
		},
		{
			name:  "single line ranges",
			outer: Range{Start: Position{Line: 5, Character: 0}, End: Position{Line: 5, Character: 50}},
			inner: Range{Start: Position{Line: 5, Character: 10}, End: Position{Line: 5, Character: 20}},
			want:  true,
		},
		{
			name:  "character position matters",
			outer: Range{Start: Position{Line: 5, Character: 10}, End: Position{Line: 5, Character: 20}},
			inner: Range{Start: Position{Line: 5, Character: 5}, End: Position{Line: 5, Character: 15}},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RangeInRange(tt.outer, tt.inner)
			if got != tt.want {
				t.Errorf("RangeInRange(%+v, %+v) = %v, want %v",
					tt.outer, tt.inner, got, tt.want)
			}
		})
	}
}

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
			name:     "method name with return type prefix",
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
		{
			name:     "just parens",
			input:    "()",
			expected: "()", // no method name to extract, returns as-is
		},
		{
			name:     "no params with colon return",
			input:    "getName : String",
			expected: "getName",
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

package util

import "testing"

func TestSplitCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"findByEmail", "find By Email"},
		{"UserService", "User Service"},
		{"getHTTPClient", "get HTTPClient"}, // consecutive caps stay together
		{"ID", "ID"},                         // all caps stay together
		{"id", "id"},
		{"", ""},
		{"lowercase", "lowercase"},
		{"XMLParser", "XMLParser"}, // leading caps stay together
		{"getUserById", "get User By Id"},
		{"saveOrder", "save Order"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("splitCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeTypeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"String", "String"},
		{"List<User>", "List User"},
		{"Map<String, User>", "Map String User"},
		{"com.example.User", "User"},
		{"String[]", "String"},  // array suffix removed but array word not added (simplified)
		{"void", "void"},
		{"ResponseEntity<List<UserDto>>", "Response Entity List User Dto"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTypeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeSignatureForEmbedding(t *testing.T) {
	tests := []struct {
		name     string
		info     SignatureInfo
		expected string
	}{
		{
			name: "simple method",
			info: SignatureInfo{
				ClassName:  "UserService",
				MethodName: "findByEmail",
				Parameters: []ParameterInfo{{Name: "email", Type: "String"}},
				ReturnType: "User",
			},
			expected: "User Service find By Email String email returns User",
		},
		{
			name: "void method",
			info: SignatureInfo{
				ClassName:  "OrderService",
				MethodName: "saveOrder",
				Parameters: []ParameterInfo{{Name: "order", Type: "Order"}},
				ReturnType: "void",
			},
			expected: "Order Service save Order Order order returns void",
		},
		{
			name: "method with generics",
			info: SignatureInfo{
				ClassName:  "ProductRepository",
				MethodName: "getProductsByCategory",
				Parameters: []ParameterInfo{{Name: "category", Type: "String"}},
				ReturnType: "List<Product>",
			},
			expected: "Product Repository get Products By Category String category returns List Product",
		},
		{
			name: "authentication method",
			info: SignatureInfo{
				ClassName:  "AuthService",
				MethodName: "authenticate",
				Parameters: []ParameterInfo{
					{Name: "username", Type: "String"},
					{Name: "password", Type: "String"},
				},
				ReturnType: "AuthToken",
			},
			expected: "Auth Service authenticate String username String password returns Auth Token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSignatureForEmbedding(tt.info)
			if result != tt.expected {
				t.Errorf("NormalizeSignatureForEmbedding() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildSignatureInfo(t *testing.T) {
	info := BuildSignatureInfo(
		"UserService",
		"findByEmail",
		"User",
		[]string{"email"},
		[]string{"String"},
	)

	if info.ClassName != "UserService" {
		t.Errorf("ClassName = %q, want %q", info.ClassName, "UserService")
	}
	if len(info.Parameters) != 1 {
		t.Errorf("len(Parameters) = %d, want 1", len(info.Parameters))
	}
	if info.Parameters[0].Name != "email" {
		t.Errorf("Parameters[0].Name = %q, want %q", info.Parameters[0].Name, "email")
	}
}

func TestFormatSignatureString(t *testing.T) {
	info := SignatureInfo{
		MethodName: "findByEmail",
		Parameters: []ParameterInfo{{Name: "email", Type: "String"}},
		ReturnType: "User",
	}

	result := FormatSignatureString(info)
	expected := "User findByEmail(String email)"

	if result != expected {
		t.Errorf("FormatSignatureString() = %q, want %q", result, expected)
	}
}

package util

import (
	"regexp"
	"strings"
	"unicode"
)

// SignatureInfo holds parsed method signature information
type SignatureInfo struct {
	ClassName      string
	MethodName     string
	Parameters     []ParameterInfo
	ReturnType     string
	ParameterTypes []string
	ParameterNames []string
}

// ParameterInfo holds parameter details
type ParameterInfo struct {
	Name string
	Type string
}

// NormalizeSignatureForEmbedding converts a method signature into a normalized
// text representation optimized for embedding-based semantic search.
//
// Example:
//
//	Input: class=UserService, method=findByEmail, params=[(email, String)], returns=User
//	Output: "UserService find By Email String email returns User"
func NormalizeSignatureForEmbedding(info SignatureInfo) string {
	var parts []string

	// Add class name (split camelCase)
	if info.ClassName != "" {
		parts = append(parts, splitCamelCase(info.ClassName))
	}

	// Add method name (split camelCase)
	if info.MethodName != "" {
		parts = append(parts, splitCamelCase(info.MethodName))
	}

	// Add parameter types and names
	for _, param := range info.Parameters {
		normalizedType := normalizeTypeName(param.Type)
		parts = append(parts, normalizedType)
		if param.Name != "" {
			parts = append(parts, splitCamelCase(param.Name))
		}
	}

	// Add return type
	if info.ReturnType != "" && info.ReturnType != "void" {
		parts = append(parts, "returns", normalizeTypeName(info.ReturnType))
	} else {
		parts = append(parts, "returns void")
	}

	return strings.Join(parts, " ")
}

// splitCamelCase splits a camelCase or PascalCase string into separate words.
// Example: "findByEmail" -> "find By Email"
func splitCamelCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	var prevLower bool

	for i, r := range s {
		isUpper := unicode.IsUpper(r)

		if i > 0 && isUpper && prevLower {
			result.WriteRune(' ')
		}

		result.WriteRune(r)
		prevLower = unicode.IsLower(r)
	}

	return result.String()
}

// normalizeTypeName normalizes a type name for embedding.
// - Removes generic brackets: List<User> -> List User
// - Splits camelCase
// - Removes array brackets: String[] -> String
// - Handles common Java types
func normalizeTypeName(typeName string) string {
	if typeName == "" {
		return ""
	}

	// Handle arrays - just remove the brackets
	typeName = strings.ReplaceAll(typeName, "[]", "")

	// Handle generics: List<User> -> List User
	// Map<String, User> -> Map String User
	genericRegex := regexp.MustCompile(`[<>]`)
	typeName = genericRegex.ReplaceAllString(typeName, " ")
	typeName = strings.ReplaceAll(typeName, ",", " ")

	// Handle fully qualified names: take last part
	// com.example.User -> User
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		typeName = typeName[idx+1:]
	}

	// Split camelCase
	result := splitCamelCase(strings.TrimSpace(typeName))

	// Clean up multiple spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	result = spaceRegex.ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

// BuildSignatureInfo creates a SignatureInfo from raw components
func BuildSignatureInfo(className, methodName, returnType string, paramNames, paramTypes []string) SignatureInfo {
	info := SignatureInfo{
		ClassName:      className,
		MethodName:     methodName,
		ReturnType:     returnType,
		ParameterTypes: paramTypes,
		ParameterNames: paramNames,
	}

	// Build Parameters slice
	maxLen := len(paramNames)
	if len(paramTypes) > maxLen {
		maxLen = len(paramTypes)
	}

	for i := 0; i < maxLen; i++ {
		param := ParameterInfo{}
		if i < len(paramNames) {
			param.Name = paramNames[i]
		}
		if i < len(paramTypes) {
			param.Type = paramTypes[i]
		}
		info.Parameters = append(info.Parameters, param)
	}

	return info
}

// FormatSignatureString creates a human-readable signature string
// Example: "User findByEmail(String email)"
func FormatSignatureString(info SignatureInfo) string {
	var params []string
	for _, p := range info.Parameters {
		if p.Type != "" && p.Name != "" {
			params = append(params, p.Type+" "+p.Name)
		} else if p.Type != "" {
			params = append(params, p.Type)
		}
	}

	returnPart := ""
	if info.ReturnType != "" {
		returnPart = info.ReturnType + " "
	}

	return returnPart + info.MethodName + "(" + strings.Join(params, ", ") + ")"
}

// ParseJavaSignature parses a Java method signature string to extract components
// Example: "public User findByEmail(String email)" -> SignatureInfo
func ParseJavaSignature(signature, methodName, className string) SignatureInfo {
	info := SignatureInfo{
		ClassName:  className,
		MethodName: methodName,
	}

	if signature == "" {
		return info
	}

	// Java signature format: [modifiers] returnType methodName(paramType paramName, ...)
	// Examples:
	// "public User findByEmail(String email)"
	// "public static void main(String[] args)"
	// "List<User> findAll()"

	// Find parameters part within parentheses
	parenStart := strings.Index(signature, "(")
	parenEnd := strings.LastIndex(signature, ")")

	if parenStart > 0 && parenEnd > parenStart {
		// Extract and parse parameters
		paramsStr := strings.TrimSpace(signature[parenStart+1 : parenEnd])
		if paramsStr != "" {
			params := splitParameters(paramsStr)
			for _, param := range params {
				param = strings.TrimSpace(param)
				if param == "" {
					continue
				}
				// Split by last space to get type and name
				parts := strings.Fields(param)
				if len(parts) >= 2 {
					// Last part is name, rest is type (handles "final String email", "List<User> users")
					paramName := parts[len(parts)-1]
					paramType := strings.Join(parts[:len(parts)-1], " ")
					info.Parameters = append(info.Parameters, ParameterInfo{Name: paramName, Type: paramType})
					info.ParameterTypes = append(info.ParameterTypes, paramType)
					info.ParameterNames = append(info.ParameterNames, paramName)
				} else if len(parts) == 1 {
					// Just type, no name (rare but possible)
					info.Parameters = append(info.Parameters, ParameterInfo{Type: parts[0]})
					info.ParameterTypes = append(info.ParameterTypes, parts[0])
				}
			}
		}

		// Extract return type from before method name
		beforeParen := strings.TrimSpace(signature[:parenStart])
		// Find method name in the string
		methodIdx := strings.LastIndex(beforeParen, methodName)
		if methodIdx > 0 {
			// Everything before method name is modifiers + return type
			beforeMethod := strings.TrimSpace(beforeParen[:methodIdx])
			// Split by space and take last non-modifier part as return type
			parts := strings.Fields(beforeMethod)
			if len(parts) > 0 {
				// Skip common modifiers and take last part as return type
				for i := len(parts) - 1; i >= 0; i-- {
					if !isJavaModifier(parts[i]) {
						info.ReturnType = parts[i]
						break
					}
				}
			}
		}
	}

	return info
}

// ParseGoSignature parses a Go function signature string to extract components
// Example: "FindByEmail(email string) *User" -> SignatureInfo
func ParseGoSignature(signature, methodName, className string) SignatureInfo {
	info := SignatureInfo{
		ClassName:  className,
		MethodName: methodName,
	}

	if signature == "" {
		return info
	}

	// Go signature format: funcName(params) returnType
	// or: funcName(params) (returnType, error)
	// Examples:
	// "FindByEmail(email string) *User"
	// "Save(ctx context.Context, user *User) error"

	parenStart := strings.Index(signature, "(")
	parenEnd := strings.Index(signature, ")")

	if parenStart >= 0 && parenEnd > parenStart {
		// Extract parameters
		paramsStr := strings.TrimSpace(signature[parenStart+1 : parenEnd])
		if paramsStr != "" {
			params := splitParameters(paramsStr)
			for _, param := range params {
				param = strings.TrimSpace(param)
				if param == "" {
					continue
				}
				// Go format: name type (e.g., "email string", "ctx context.Context")
				parts := strings.Fields(param)
				if len(parts) >= 2 {
					paramName := parts[0]
					paramType := strings.Join(parts[1:], " ")
					info.Parameters = append(info.Parameters, ParameterInfo{Name: paramName, Type: paramType})
					info.ParameterTypes = append(info.ParameterTypes, paramType)
					info.ParameterNames = append(info.ParameterNames, paramName)
				} else if len(parts) == 1 {
					// Could be just type (for grouped params like "a, b int")
					info.Parameters = append(info.Parameters, ParameterInfo{Type: parts[0]})
					info.ParameterTypes = append(info.ParameterTypes, parts[0])
				}
			}
		}

		// Extract return type after the closing paren
		afterParen := strings.TrimSpace(signature[parenEnd+1:])
		if afterParen != "" {
			// Handle multiple return values: (User, error) -> just take first
			if strings.HasPrefix(afterParen, "(") {
				end := strings.Index(afterParen, ")")
				if end > 0 {
					returns := strings.Split(afterParen[1:end], ",")
					if len(returns) > 0 {
						info.ReturnType = strings.TrimSpace(returns[0])
					}
				}
			} else {
				info.ReturnType = afterParen
			}
		}
	}

	return info
}

// ParsePythonSignature parses a Python function signature string to extract components
// Example: "find_by_email(email: str) -> User" -> SignatureInfo
func ParsePythonSignature(signature, methodName, className string) SignatureInfo {
	info := SignatureInfo{
		ClassName:  className,
		MethodName: methodName,
	}

	if signature == "" {
		return info
	}

	// Python signature format: func_name(params) -> return_type
	// Examples:
	// "find_by_email(email: str) -> User"
	// "save(self, user: User) -> None"

	// Find return type after ->
	arrowIdx := strings.Index(signature, "->")
	signaturePart := signature
	if arrowIdx > 0 {
		info.ReturnType = strings.TrimSpace(signature[arrowIdx+2:])
		signaturePart = signature[:arrowIdx]
	}

	parenStart := strings.Index(signaturePart, "(")
	parenEnd := strings.LastIndex(signaturePart, ")")

	if parenStart >= 0 && parenEnd > parenStart {
		paramsStr := strings.TrimSpace(signaturePart[parenStart+1 : parenEnd])
		if paramsStr != "" {
			params := splitParameters(paramsStr)
			for _, param := range params {
				param = strings.TrimSpace(param)
				if param == "" || param == "self" || param == "cls" {
					continue
				}
				// Python format: name: type or just name
				colonIdx := strings.Index(param, ":")
				if colonIdx > 0 {
					paramName := strings.TrimSpace(param[:colonIdx])
					paramType := strings.TrimSpace(param[colonIdx+1:])
					// Remove default value if present
					if eqIdx := strings.Index(paramType, "="); eqIdx > 0 {
						paramType = strings.TrimSpace(paramType[:eqIdx])
					}
					info.Parameters = append(info.Parameters, ParameterInfo{Name: paramName, Type: paramType})
					info.ParameterTypes = append(info.ParameterTypes, paramType)
					info.ParameterNames = append(info.ParameterNames, paramName)
				} else {
					// Just parameter name, no type annotation
					paramName := param
					// Remove default value if present
					if eqIdx := strings.Index(paramName, "="); eqIdx > 0 {
						paramName = strings.TrimSpace(paramName[:eqIdx])
					}
					info.Parameters = append(info.Parameters, ParameterInfo{Name: paramName})
					info.ParameterNames = append(info.ParameterNames, paramName)
				}
			}
		}
	}

	return info
}

// ParseJavaScriptSignature parses a JavaScript/TypeScript function signature
// Example: "findByEmail(email: string): User" -> SignatureInfo
func ParseJavaScriptSignature(signature, methodName, className string) SignatureInfo {
	info := SignatureInfo{
		ClassName:  className,
		MethodName: methodName,
	}

	if signature == "" {
		return info
	}

	// TypeScript format: funcName(param: type): returnType
	// JavaScript format: funcName(param)

	// Find return type after ): for TypeScript
	parenEnd := strings.LastIndex(signature, ")")
	if parenEnd > 0 && parenEnd < len(signature)-1 {
		afterParen := strings.TrimSpace(signature[parenEnd+1:])
		if strings.HasPrefix(afterParen, ":") {
			info.ReturnType = strings.TrimSpace(afterParen[1:])
		}
	}

	parenStart := strings.Index(signature, "(")
	if parenStart >= 0 && parenEnd > parenStart {
		paramsStr := strings.TrimSpace(signature[parenStart+1 : parenEnd])
		if paramsStr != "" {
			params := splitParameters(paramsStr)
			for _, param := range params {
				param = strings.TrimSpace(param)
				if param == "" {
					continue
				}
				// TypeScript format: name: type or name?: type
				colonIdx := strings.Index(param, ":")
				if colonIdx > 0 {
					paramName := strings.TrimSpace(param[:colonIdx])
					paramName = strings.TrimSuffix(paramName, "?") // Remove optional marker
					paramType := strings.TrimSpace(param[colonIdx+1:])
					info.Parameters = append(info.Parameters, ParameterInfo{Name: paramName, Type: paramType})
					info.ParameterTypes = append(info.ParameterTypes, paramType)
					info.ParameterNames = append(info.ParameterNames, paramName)
				} else {
					// Just parameter name (JavaScript)
					info.Parameters = append(info.Parameters, ParameterInfo{Name: param})
					info.ParameterNames = append(info.ParameterNames, param)
				}
			}
		}
	}

	return info
}

// ParseSignatureByLanguage parses a signature string based on the language
func ParseSignatureByLanguage(signature, methodName, className, language string) SignatureInfo {
	switch language {
	case "java":
		return ParseJavaSignature(signature, methodName, className)
	case "go":
		return ParseGoSignature(signature, methodName, className)
	case "python":
		return ParsePythonSignature(signature, methodName, className)
	case "javascript", "typescript":
		return ParseJavaScriptSignature(signature, methodName, className)
	default:
		// Fallback: just use method name and class name
		return SignatureInfo{
			ClassName:  className,
			MethodName: methodName,
		}
	}
}

// splitParameters splits a parameter string by commas, respecting nested brackets
func splitParameters(paramsStr string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, ch := range paramsStr {
		switch ch {
		case '<', '[', '(':
			depth++
			current.WriteRune(ch)
		case '>', ']', ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// isJavaModifier checks if a string is a Java modifier keyword
func isJavaModifier(s string) bool {
	modifiers := map[string]bool{
		"public": true, "private": true, "protected": true,
		"static": true, "final": true, "abstract": true,
		"synchronized": true, "native": true, "strictfp": true,
		"transient": true, "volatile": true, "default": true,
	}
	return modifiers[s]
}

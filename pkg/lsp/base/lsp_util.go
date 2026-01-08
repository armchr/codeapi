package base

import "strings"

func MatchLastSegment(name, nameInFile string, delim string) bool {
	nameLastSegment := LastSegment(name)
	nameInFileLastSegment := LastSegment(nameInFile)

	return nameLastSegment == nameInFileLastSegment
}

func LastSegment(name string) string {
	// Split by delimiter and return last segment
	nameParts := strings.Split(name, ".")
	return nameParts[len(nameParts)-1]
}

func MatchExact(name, nameInFile string) bool {
	return name == nameInFile
}

func MatchIgnoreCase(name, nameInFile string) bool {
	return strings.EqualFold(name, nameInFile)
}

func MatchIgnoreCaseLastSegment(name, nameInFile string, delim string) bool {
	// Split by delimiter and match last segment ignoring case
	nameParts := strings.Split(name, delim)
	nameInFileParts := strings.Split(nameInFile, delim)

	return strings.EqualFold(nameParts[len(nameParts)-1], nameInFileParts[len(nameInFileParts)-1])
}

func RangeInRange(outer, inner Range) bool {
	return outer.ContainsRange(&inner)
}

// ExtractJavaMethodName extracts just the method name from a Java LSP signature.
// Java LSP (Eclipse JDT.LS) returns method names in formats like:
//   - "findByOwnerId(Long) : List<PetDto>" -> "findByOwnerId"
//   - "countByType(String) : long" -> "countByType"
//   - "toString()" -> "toString"
//   - "simpleMethod" -> "simpleMethod" (no change if no signature)
//   - "Optional<User> methodName(Long)" -> "methodName" (return type prefix)
func ExtractJavaMethodName(fullSignature string) string {
	name := fullSignature

	// Handle case where return type is prefixed (rare but possible)
	// e.g., "Optional<User> findById(Long)" - find the method name before "("
	if parenIdx := strings.Index(name, "("); parenIdx > 0 {
		// Everything before "(" contains the method name (possibly with return type)
		beforeParen := name[:parenIdx]

		// If there's a space, the method name is the last word before "("
		if spaceIdx := strings.LastIndex(beforeParen, " "); spaceIdx >= 0 {
			name = beforeParen[spaceIdx+1:]
		} else {
			name = beforeParen
		}
	} else if colonIdx := strings.Index(name, " : "); colonIdx > 0 {
		// Handle case like "methodName : ReturnType" (no params)
		name = strings.TrimSpace(name[:colonIdx])
	}

	return name
}

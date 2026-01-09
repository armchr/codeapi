package parse

import (
	"context"
	"encoding/json"
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	"go.uber.org/zap"
)

// Helper to create a JavaVisitor for testing
func newTestJavaVisitor(sourceCode []byte) *JavaVisitor {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(1, 1, nil, sourceCode, logger)
	return NewJavaVisitor(logger, translator)
}

// Helper to parse Java code and return the root node
func parseJava(t *testing.T, code string) (*tree_sitter.Tree, *tree_sitter.Node) {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	
	if err := parser.SetLanguage(tree_sitter.NewLanguage(java.Language())); err != nil {
		t.Fatalf("Failed to set Java language: %v", err)
	}
	
	tree := parser.Parse([]byte(code), nil)
	if tree == nil {
		t.Fatal("Failed to parse Java code")
	}
	
	return tree, tree.RootNode()
}

// Helper to find a node by kind
func findNodeByKind(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	if node.Kind() == kind {
		return node
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		if found := findNodeByKind(node.Child(i), kind); found != nil {
			return found
		}
	}
	return nil
}

// Helper to find all nodes by kind
func findAllNodesByKind(node *tree_sitter.Node, kind string) []*tree_sitter.Node {
	var result []*tree_sitter.Node
	if node.Kind() == kind {
		result = append(result, node)
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		result = append(result, findAllNodesByKind(node.Child(i), kind)...)
	}
	return result
}

func TestExtractAnnotations_MarkerAnnotation(t *testing.T) {
	code := `
public class MyClass {
    @Override
    public void method() {}
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	// Find the method_declaration node
	methodNode := findNodeByKind(root, "method_declaration")
	if methodNode == nil {
		t.Fatal("Could not find method_declaration node")
	}
	
	annotations := jv.extractAnnotations(methodNode)
	
	if len(annotations) != 1 {
		t.Fatalf("Expected 1 annotation, got %d", len(annotations))
	}
	
	// Parse the JSON annotation
	var ann map[string]interface{}
	if err := json.Unmarshal([]byte(annotations[0]), &ann); err != nil {
		t.Fatalf("Failed to parse annotation JSON: %v", err)
	}
	
	if ann["name"] != "Override" {
		t.Errorf("Expected annotation name 'Override', got %v", ann["name"])
	}
	
	if ann["arguments"] != nil {
		t.Error("Marker annotation should not have arguments")
	}
}

func TestExtractAnnotations_WithStringValue(t *testing.T) {
	code := `
public class MyController {
    @GetMapping("/api/users")
    public void getUsers() {}
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	methodNode := findNodeByKind(root, "method_declaration")
	if methodNode == nil {
		t.Fatal("Could not find method_declaration node")
	}
	
	annotations := jv.extractAnnotations(methodNode)
	
	if len(annotations) != 1 {
		t.Fatalf("Expected 1 annotation, got %d", len(annotations))
	}
	
	var ann map[string]interface{}
	if err := json.Unmarshal([]byte(annotations[0]), &ann); err != nil {
		t.Fatalf("Failed to parse annotation JSON: %v", err)
	}
	
	if ann["name"] != "GetMapping" {
		t.Errorf("Expected annotation name 'GetMapping', got %v", ann["name"])
	}
	
	args, ok := ann["arguments"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected arguments map")
	}
	
	if args["value"] != "/api/users" {
		t.Errorf("Expected value '/api/users', got %v", args["value"])
	}
}

func TestExtractAnnotations_WithNamedArgs(t *testing.T) {
	code := `
public class MyClass {
    @Size(min = 1, max = 50)
    private String name;
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	// Find field_declaration
	fieldNode := findNodeByKind(root, "field_declaration")
	if fieldNode == nil {
		t.Fatal("Could not find field_declaration node")
	}
	
	annotations := jv.extractAnnotations(fieldNode)
	
	if len(annotations) != 1 {
		t.Fatalf("Expected 1 annotation, got %d", len(annotations))
	}
	
	var ann map[string]interface{}
	if err := json.Unmarshal([]byte(annotations[0]), &ann); err != nil {
		t.Fatalf("Failed to parse annotation JSON: %v", err)
	}
	
	if ann["name"] != "Size" {
		t.Errorf("Expected annotation name 'Size', got %v", ann["name"])
	}
	
	args, ok := ann["arguments"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected arguments map")
	}
	
	if args["min"] != "1" {
		t.Errorf("Expected min='1', got %v", args["min"])
	}
	if args["max"] != "50" {
		t.Errorf("Expected max='50', got %v", args["max"])
	}
}

func TestExtractAnnotations_Multiple(t *testing.T) {
	code := `
public class MyController {
    @Deprecated
    @GetMapping("/test")
    public void testMethod() {}
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	methodNode := findNodeByKind(root, "method_declaration")
	if methodNode == nil {
		t.Fatal("Could not find method_declaration node")
	}
	
	annotations := jv.extractAnnotations(methodNode)
	
	if len(annotations) != 2 {
		t.Fatalf("Expected 2 annotations, got %d", len(annotations))
	}
	
	// Check that both annotations are present
	var foundDeprecated, foundGetMapping bool
	for _, annJson := range annotations {
		var ann map[string]interface{}
		json.Unmarshal([]byte(annJson), &ann)
		switch ann["name"] {
		case "Deprecated":
			foundDeprecated = true
		case "GetMapping":
			foundGetMapping = true
		}
	}
	
	if !foundDeprecated {
		t.Error("Expected to find @Deprecated annotation")
	}
	if !foundGetMapping {
		t.Error("Expected to find @GetMapping annotation")
	}
}

func TestExtractAnnotations_NoAnnotations(t *testing.T) {
	code := `
public class MyClass {
    public void method() {}
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	methodNode := findNodeByKind(root, "method_declaration")
	if methodNode == nil {
		t.Fatal("Could not find method_declaration node")
	}
	
	annotations := jv.extractAnnotations(methodNode)
	
	if annotations != nil && len(annotations) != 0 {
		t.Errorf("Expected no annotations, got %d", len(annotations))
	}
}

func TestExtractAnnotations_ClassLevel(t *testing.T) {
	code := `
@RestController
@RequestMapping("/api")
public class MyController {
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	classNode := findNodeByKind(root, "class_declaration")
	if classNode == nil {
		t.Fatal("Could not find class_declaration node")
	}
	
	annotations := jv.extractAnnotations(classNode)
	
	if len(annotations) != 2 {
		t.Fatalf("Expected 2 annotations, got %d", len(annotations))
	}
}

func TestExtractAnnotationArguments_SingleString(t *testing.T) {
	code := `
public class MyController {
    @GetMapping("/users")
    public void getUsers() {}
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	// Find the annotation_argument_list node
	argList := findNodeByKind(root, "annotation_argument_list")
	if argList == nil {
		t.Fatal("Could not find annotation_argument_list node")
	}
	
	args := jv.extractAnnotationArguments(argList)
	
	if args["value"] != "/users" {
		t.Errorf("Expected value='/users', got %v", args["value"])
	}
}

func TestExtractAnnotationArguments_NamedPairs(t *testing.T) {
	code := `
public class MyClass {
    @Column(name = "user_name", nullable = false)
    private String userName;
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	argList := findNodeByKind(root, "annotation_argument_list")
	if argList == nil {
		t.Fatal("Could not find annotation_argument_list node")
	}
	
	args := jv.extractAnnotationArguments(argList)
	
	if args["name"] != "user_name" {
		t.Errorf("Expected name='user_name', got %v", args["name"])
	}
	if args["nullable"] != "false" {
		t.Errorf("Expected nullable='false', got %v", args["nullable"])
	}
}

func TestExtractAnnotationArguments_IntegerValue(t *testing.T) {
	code := `
public class MyClass {
    @Size(min = 5, max = 100)
    private String field;
}
`
	tree, root := parseJava(t, code)
	defer tree.Close()
	
	jv := newTestJavaVisitor([]byte(code))
	
	argList := findNodeByKind(root, "annotation_argument_list")
	if argList == nil {
		t.Fatal("Could not find annotation_argument_list node")
	}
	
	args := jv.extractAnnotationArguments(argList)
	
	if args["min"] != "5" {
		t.Errorf("Expected min='5', got %v", args["min"])
	}
	if args["max"] != "100" {
		t.Errorf("Expected max='100', got %v", args["max"])
	}
}

// Note: Full TraverseNode tests require a mock CodeGraph and are not included here.
// The annotation extraction tests above provide coverage for the core parsing logic.

func TestJavaVisitor_NilNode(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(1, 1, nil, []byte(""), logger)
	jv := NewJavaVisitor(logger, translator)
	
	ctx := context.Background()
	nodeID := jv.TraverseNode(ctx, nil, 0)
	
	if nodeID != 0 {
		t.Error("Expected InvalidNodeID (0) for nil input")
	}
}

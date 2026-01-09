package parse

import (
	"context"
	"testing"

	"github.com/armchr/codeapi/internal/model/ast"
	"go.uber.org/zap"
)

// --- Symbol Tests ---

func TestNewSymbol(t *testing.T) {
	node := &ast.Node{ID: 1, Name: "testVar"}
	sym := NewSymbol(node)

	if sym == nil {
		t.Fatal("NewSymbol returned nil")
	}
	if sym.Node != node {
		t.Error("Symbol.Node does not match input node")
	}
	if sym.Fields == nil {
		t.Error("Symbol.Fields is nil, expected initialized map")
	}
}

func TestSymbol_AddField(t *testing.T) {
	parentNode := &ast.Node{ID: 1, Name: "parent"}
	parentSym := NewSymbol(parentNode)

	fieldNode := &ast.Node{ID: 2, Name: "field1"}
	fieldSym := NewSymbol(fieldNode)

	// Add field successfully
	err := parentSym.AddField(fieldSym)
	if err != nil {
		t.Errorf("AddField failed: %v", err)
	}

	// Verify field was added
	if got := parentSym.GetField("field1"); got != fieldSym {
		t.Error("GetField did not return the added field")
	}

	// Try to add duplicate field
	duplicateField := NewSymbol(&ast.Node{ID: 3, Name: "field1"})
	err = parentSym.AddField(duplicateField)
	if err == nil {
		t.Error("AddField should return error for duplicate field name")
	}
}

func TestSymbol_GetField(t *testing.T) {
	parentNode := &ast.Node{ID: 1, Name: "parent"}
	parentSym := NewSymbol(parentNode)

	// Get non-existent field
	if got := parentSym.GetField("nonexistent"); got != nil {
		t.Error("GetField should return nil for non-existent field")
	}

	// Add and get field
	fieldNode := &ast.Node{ID: 2, Name: "myField"}
	fieldSym := NewSymbol(fieldNode)
	parentSym.AddField(fieldSym)

	if got := parentSym.GetField("myField"); got != fieldSym {
		t.Error("GetField did not return the correct field")
	}
}

// --- Scope Tests ---

func TestNewScope(t *testing.T) {
	tests := []struct {
		name     string
		parent   *Scope
		isRhs    bool
		wantRhs  bool
	}{
		{"root scope", nil, false, false},
		{"child scope", NewScope(nil, false), false, false},
		{"rhs scope", nil, true, true},
		{"child rhs scope", NewScope(nil, false), true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := NewScope(tt.parent, tt.isRhs)
			if scope == nil {
				t.Fatal("NewScope returned nil")
			}
			if scope.Parent != tt.parent {
				t.Error("Parent scope mismatch")
			}
			if scope.IsRhs() != tt.wantRhs {
				t.Errorf("IsRhs() = %v, want %v", scope.IsRhs(), tt.wantRhs)
			}
			if scope.symbols == nil {
				t.Error("symbols map not initialized")
			}
			if scope.notContainedNodes == nil {
				t.Error("notContainedNodes map not initialized")
			}
		})
	}
}

func TestScope_AddSymbol(t *testing.T) {
	scope := NewScope(nil, false)

	sym1 := NewSymbol(&ast.Node{ID: 1, Name: "var1"})
	err := scope.AddSymbol(sym1)
	if err != nil {
		t.Errorf("AddSymbol failed: %v", err)
	}

	// Verify symbol was added
	if got := scope.GetSymbol("var1"); got != sym1 {
		t.Error("GetSymbol did not return added symbol")
	}

	// Try to add duplicate symbol
	sym2 := NewSymbol(&ast.Node{ID: 2, Name: "var1"})
	err = scope.AddSymbol(sym2)
	if err == nil {
		t.Error("AddSymbol should return error for duplicate symbol name")
	}
}

func TestScope_GetSymbol(t *testing.T) {
	scope := NewScope(nil, false)

	// Non-existent symbol
	if got := scope.GetSymbol("nonexistent"); got != nil {
		t.Error("GetSymbol should return nil for non-existent symbol")
	}

	// Add and get symbol
	sym := NewSymbol(&ast.Node{ID: 1, Name: "myVar"})
	scope.AddSymbol(sym)

	if got := scope.GetSymbol("myVar"); got != sym {
		t.Error("GetSymbol did not return correct symbol")
	}
}

func TestScope_Resolve(t *testing.T) {
	// Create scope hierarchy: global -> outer -> inner
	globalScope := NewScope(nil, false)
	outerScope := NewScope(globalScope, false)
	innerScope := NewScope(outerScope, false)

	// Add symbols at different levels
	globalSym := NewSymbol(&ast.Node{ID: 1, Name: "globalVar"})
	globalScope.AddSymbol(globalSym)

	outerSym := NewSymbol(&ast.Node{ID: 2, Name: "outerVar"})
	outerScope.AddSymbol(outerSym)

	innerSym := NewSymbol(&ast.Node{ID: 3, Name: "innerVar"})
	innerScope.AddSymbol(innerSym)

	// Shadowing: add same-named symbol in inner scope
	shadowedSym := NewSymbol(&ast.Node{ID: 4, Name: "outerVar"})
	innerScope.AddSymbol(shadowedSym)

	tests := []struct {
		name       string
		scope      *Scope
		symbolName string
		wantSym    *Symbol
	}{
		{"resolve in current scope", innerScope, "innerVar", innerSym},
		{"resolve in parent scope", innerScope, "globalVar", globalSym},
		{"resolve shadowed symbol", innerScope, "outerVar", shadowedSym}, // Should get inner, not outer
		{"resolve from outer scope", outerScope, "outerVar", outerSym},
		{"resolve from outer scope - global", outerScope, "globalVar", globalSym},
		{"resolve non-existent", innerScope, "nonexistent", nil},
		{"resolve from global scope", globalScope, "globalVar", globalSym},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scope.Resolve(tt.symbolName)
			if got != tt.wantSym {
				t.Errorf("Resolve(%q) got %v, want %v", tt.symbolName, got, tt.wantSym)
			}
		})
	}
}

func TestScope_NotContainedNodes(t *testing.T) {
	scope := NewScope(nil, false)

	// Add nodes
	scope.AddNotContainedNode(ast.NodeID(1))
	scope.AddNotContainedNode(ast.NodeID(2))
	scope.AddNotContainedNode(ast.NodeID(3))

	// Check if nodes are not contained
	if !scope.IsNotContainedNode(ast.NodeID(1)) {
		t.Error("Node 1 should be not contained")
	}
	if !scope.IsNotContainedNode(ast.NodeID(2)) {
		t.Error("Node 2 should be not contained")
	}
	if scope.IsNotContainedNode(ast.NodeID(999)) {
		t.Error("Node 999 should not be in not contained nodes")
	}

	// Remove a node
	scope.RemoveNotContainedNode(ast.NodeID(2))
	if scope.IsNotContainedNode(ast.NodeID(2)) {
		t.Error("Node 2 should have been removed")
	}

	// Get all not contained nodes
	notContained := scope.GetAllNotContainedNodes()
	if len(notContained) != 2 {
		t.Errorf("Expected 2 not contained nodes, got %d", len(notContained))
	}
}

func TestScope_RhsVars(t *testing.T) {
	// Non-RHS scope
	normalScope := NewScope(nil, false)
	normalScope.AddRhsVar(ast.NodeID(1)) // Should be no-op
	if vars := normalScope.GetRhsVars(); vars != nil {
		t.Error("Non-RHS scope should return nil for GetRhsVars")
	}

	// RHS scope
	rhsScope := NewScope(nil, true)
	rhsScope.AddRhsVar(ast.NodeID(1))
	rhsScope.AddRhsVar(ast.NodeID(2))

	vars := rhsScope.GetRhsVars()
	if len(vars) != 2 {
		t.Errorf("Expected 2 RHS vars, got %d", len(vars))
	}
}

// --- TranslateFromSyntaxTree Tests ---

func TestNewTranslateFromSyntaxTree(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(1, 1, nil, []byte("test content"), logger)

	if translator == nil {
		t.Fatal("NewTranslateFromSyntaxTree returned nil")
	}
	if translator.FileID != 1 {
		t.Errorf("FileID = %d, want 1", translator.FileID)
	}
	if translator.Version != 1 {
		t.Errorf("Version = %d, want 1", translator.Version)
	}
	if translator.CurrentScope == nil {
		t.Error("CurrentScope should not be nil")
	}
	if len(translator.ScopeStack) != 1 {
		t.Errorf("ScopeStack length = %d, want 1", len(translator.ScopeStack))
	}
	if translator.NodeIDSeq != 1 {
		t.Errorf("NodeIDSeq = %d, want 1", translator.NodeIDSeq)
	}
}

func TestTranslateFromSyntaxTree_PushScope(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(1, 1, nil, []byte(""), logger)

	initialStackLen := len(translator.ScopeStack)
	initialScope := translator.CurrentScope

	// Push a non-RHS scope
	translator.PushScope(false)

	if len(translator.ScopeStack) != initialStackLen+1 {
		t.Errorf("ScopeStack length = %d, want %d", len(translator.ScopeStack), initialStackLen+1)
	}
	if translator.CurrentScope == initialScope {
		t.Error("CurrentScope should have changed after PushScope")
	}
	if translator.CurrentScope.Parent != initialScope {
		t.Error("New scope's parent should be the previous current scope")
	}
	if translator.CurrentScope.IsRhs() {
		t.Error("Non-RHS scope should not be RHS")
	}

	// Push an RHS scope
	translator.PushScope(true)
	if !translator.CurrentScope.IsRhs() {
		t.Error("RHS scope should be RHS")
	}
}

func TestTranslateFromSyntaxTree_PopScope(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(1, 1, nil, []byte(""), logger)

	// Push some scopes
	translator.PushScope(false)
	translator.PushScope(false)
	middleScope := translator.CurrentScope
	translator.PushScope(false)

	stackLen := len(translator.ScopeStack)

	// Pop with InvalidNodeID (doesn't create relations, just moves not-contained nodes to parent)
	ctx := context.Background()
	translator.PopScope(ctx, ast.InvalidNodeID)

	if len(translator.ScopeStack) != stackLen-1 {
		t.Errorf("ScopeStack length = %d, want %d", len(translator.ScopeStack), stackLen-1)
	}
	if translator.CurrentScope != middleScope {
		t.Error("CurrentScope should be the previous scope after PopScope")
	}
}

func TestTranslateFromSyntaxTree_PopScope_Underflow(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(1, 1, nil, []byte(""), logger)

	// Pop all scopes including the initial one
	ctx := context.Background()
	translator.PopScope(ctx, ast.InvalidNodeID)

	// Try to pop again - should handle gracefully (stack underflow)
	// This shouldn't panic
	translator.PopScope(ctx, ast.InvalidNodeID)
	// If we got here without panic, the test passes
}

func TestTranslateFromSyntaxTree_PopScope_NotContainedNodes(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(1, 1, nil, []byte(""), logger)

	parentScope := translator.CurrentScope

	// Push a child scope
	translator.PushScope(false)
	childScope := translator.CurrentScope

	// Add some not-contained nodes to the child scope
	childScope.AddNotContainedNode(ast.NodeID(100))
	childScope.AddNotContainedNode(ast.NodeID(200))

	// Pop with InvalidNodeID - should move not-contained nodes to parent
	ctx := context.Background()
	translator.PopScope(ctx, ast.InvalidNodeID)

	// Check that nodes were moved to parent scope
	if !parentScope.IsNotContainedNode(ast.NodeID(100)) {
		t.Error("Node 100 should have been moved to parent scope")
	}
	if !parentScope.IsNotContainedNode(ast.NodeID(200)) {
		t.Error("Node 200 should have been moved to parent scope")
	}
}

func TestTranslateFromSyntaxTree_NextNodeID(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(5, 1, nil, []byte(""), logger)

	id1 := translator.NextNodeID()
	id2 := translator.NextNodeID()
	id3 := translator.NextNodeID()

	// IDs should be unique and incrementing
	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Error("Node IDs should be unique")
	}

	// FileID should be encoded in the high bits
	// The ID format is: (fileID << 32) | sequenceNumber
	fileIDFromId1 := int32(id1 >> 32)
	if fileIDFromId1 != 5 {
		t.Errorf("FileID encoded in NodeID = %d, want 5", fileIDFromId1)
	}
}

func TestTranslateFromSyntaxTree_ScopeStackIntegration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	translator := NewTranslateFromSyntaxTree(1, 1, nil, []byte(""), logger)
	ctx := context.Background()

	// Simulate nested function scopes
	// Global scope (already exists)
	globalSym := NewSymbol(&ast.Node{ID: 1, Name: "globalFunc"})
	translator.CurrentScope.AddSymbol(globalSym)

	// Enter function scope
	translator.PushScope(false)
	funcSym := NewSymbol(&ast.Node{ID: 2, Name: "localVar"})
	translator.CurrentScope.AddSymbol(funcSym)

	// Enter nested block scope
	translator.PushScope(false)
	blockSym := NewSymbol(&ast.Node{ID: 3, Name: "blockVar"})
	translator.CurrentScope.AddSymbol(blockSym)

	// Should resolve all symbols from innermost scope
	if translator.CurrentScope.Resolve("blockVar") != blockSym {
		t.Error("Should resolve blockVar in current scope")
	}
	if translator.CurrentScope.Resolve("localVar") != funcSym {
		t.Error("Should resolve localVar from parent scope")
	}
	if translator.CurrentScope.Resolve("globalFunc") != globalSym {
		t.Error("Should resolve globalFunc from global scope")
	}

	// Pop block scope
	translator.PopScope(ctx, ast.InvalidNodeID)

	// blockVar should no longer be resolvable
	if translator.CurrentScope.Resolve("blockVar") != nil {
		t.Error("blockVar should not be resolvable after leaving its scope")
	}
	if translator.CurrentScope.Resolve("localVar") != funcSym {
		t.Error("localVar should still be resolvable")
	}

	// Pop function scope
	translator.PopScope(ctx, ast.InvalidNodeID)

	// Only global symbols should be resolvable
	if translator.CurrentScope.Resolve("localVar") != nil {
		t.Error("localVar should not be resolvable after leaving function scope")
	}
	if translator.CurrentScope.Resolve("globalFunc") != globalSym {
		t.Error("globalFunc should still be resolvable in global scope")
	}
}

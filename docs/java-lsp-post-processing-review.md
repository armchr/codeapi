# Java LSP Post-Processing Code Review

**Date:** January 2025
**Scope:** Analysis of function call resolution and inheritance tracking for Java
**Files Analyzed:**
- `internal/controller/post_process.go`
- `internal/model/response.go`
- `internal/parse/java_visitor.go`
- `pkg/lsp/java_client.go`
- `pkg/lsp/lsp_service.go`

---

## Executive Summary

The post-processing code has significant issues when matching Java function calls to their definitions. The primary problems are:

1. **Name format mismatch** between tree-sitter (simple names) and LSP (full signatures with generics)
2. **Missing static method resolution** (e.g., `ResponseEntity.ok()`)
3. **No inheritance/interface tracking** during class declaration parsing
4. **Unused LSP type hierarchy capabilities** for resolving Java inheritance

---

## Issue 1: Java Function Call Name Matching Failure

### Location
`internal/controller/post_process.go:258-265`

### Current Code
```go
func (pp *PostProcessor) matchesFunctionCall(callNode *ast.Node, dependency *model.FunctionDependency) bool {
    if !dependency.IsIn(&callNode.Range) {
        return false
    }
    // dependency name ends with call node name
    return strings.HasSuffix(callNode.Name, dependency.Definition.Name)
}
```

### Problem

The matching logic compares:
- `callNode.Name` (from tree-sitter) = `"findByOwnerId"` (simple method name)
- `dependency.Definition.Name` (from LSP) = `"findByOwnerId(Long) : List<PetDto>"` (full signature)

The check `strings.HasSuffix("findByOwnerId", "findByOwnerId(Long) : List<PetDto>")` **always returns false** because a shorter string cannot have a longer string as its suffix.

### Evidence from Logs

**LSP Response (line 517):**
```json
{
  "to": {
    "name": "findByOwnerId(Long) : List<PetDto>",
    "detail": "com.example.petclinic.service.PetService",
    "kind": 6,
    "uri": "file:///...PetService.java",
    "range": {...}
  },
  "fromRanges": [{"start":{"line":45,"character":33},"end":{"line":45,"character":66}}]
}
```

**Warning Log (line 527):**
```
No matching dependency found for function call  callNodeId:25769803805  callName:"findByOwnerId"
```

### Root Cause

`internal/model/response.go:233-237` stores the raw LSP name without parsing:

```go
return FunctionDependency{
    Name:          call.To.Name,  // Full signature from LSP
    Definition: FunctionDefinition{
        Name: call.To.Name,       // Full signature again
    },
}
```

The Java LSP (Eclipse JDT.LS) returns method names in the format:
```
methodName(ParamType1, ParamType2) : ReturnType
```

With generics:
```
methodName(Long) : List<PetDto>
methodName(String) : Optional<User>
```

### Impact

**All Java function call relationships fail to be created.** The code graph will have:
- Function call nodes (from tree-sitter parsing)
- Function definition nodes (from tree-sitter parsing)
- **Missing:** `CALLS_FUNCTION` relationships between them

---

## Issue 2: Static Method Calls Not Resolved

### Location
`internal/controller/post_process.go:131`

### Problem

For Java code like:
```java
return ResponseEntity.ok(petService.findByOwnerId(ownerId));
```

Tree-sitter correctly identifies two function calls:
1. `ok` (static method on `ResponseEntity`)
2. `findByOwnerId` (instance method on `petService`)

However, the LSP call hierarchy request is made at the **container function level** (`getPetsByOwner`), which returns calls to methods **within the project**.

The LSP returns:
- `PetService.findByOwnerId()` ✓

But does **not** return:
- `ResponseEntity.ok()` ✗ (external Spring Framework class)

### Evidence from Logs

**Warning Log (line 526):**
```
No matching dependency found for function call  callNodeId:25769803803  callName:"ok"
```

### Root Cause

1. `ResponseEntity` is from Spring Framework (external dependency)
2. Eclipse JDT.LS may not include external static method calls in call hierarchy
3. The post-processor doesn't distinguish between "not found in project" vs "external call"

### Impact

- Static method calls to JDK classes (`String.valueOf()`, `Integer.parseInt()`) are unresolved
- Static method calls to framework classes (`ResponseEntity.ok()`, `Collections.emptyList()`) are unresolved
- These appear as warnings in logs, cluttering output

---

## Issue 3: Java Inheritance Not Captured

### Location
`internal/parse/java_visitor.go:218-221`

### Current Code
```go
case "type_arguments", "type_list", "extends_interfaces", "implements_interfaces", "superclass":
    // Type-related nodes - traverse children for type references
    jv.translate.TraverseChildren(ctx, tsNode, scopeID)
    return ast.InvalidNodeID
```

### Problem

The visitor encounters `superclass`, `extends_interfaces`, and `implements_interfaces` nodes during parsing but **discards the inheritance information**. It only traverses children without storing the relationship.

### What Tree-sitter Provides

For Java code:
```java
public class PetService extends BaseService implements CrudOperations, Auditable {
    // ...
}
```

Tree-sitter produces:
```
(class_declaration
  name: (identifier) @class.name           ; "PetService"
  superclass: (superclass
    (type_identifier) @extends)            ; "BaseService"
  interfaces: (super_interfaces
    (type_list
      (type_identifier) @implements        ; "CrudOperations"
      (type_identifier) @implements)))     ; "Auditable"
```

### What Currently Happens

1. `handleClassDeclaration` is called
2. Class name is extracted: `"PetService"`
3. Methods and fields are extracted
4. `superclass` node is encountered → `TraverseChildren()` → **information lost**
5. `super_interfaces` node is encountered → `TraverseChildren()` → **information lost**
6. Class node is created **without inheritance metadata**

### Impact

- No `EXTENDS` relationships in code graph
- No `IMPLEMENTS` relationships in code graph
- Cannot query "what classes extend BaseService?"
- Cannot query "what classes implement CrudOperations?"

---

## Issue 4: LSP Type Hierarchy Not Utilized

### Evidence

From LSP initialize response (log line 98):
```json
{
  "capabilities": {
    "typeHierarchyProvider": true,
    "callHierarchyProvider": true,
    ...
  }
}
```

### Available LSP Methods (Unused)

The Java LSP supports but codeapi doesn't use:

| Method | Purpose |
|--------|---------|
| `textDocument/prepareTypeHierarchy` | Get type hierarchy item for a position |
| `typeHierarchy/supertypes` | Get parent types (what this class extends/implements) |
| `typeHierarchy/subtypes` | Get child types (what extends/implements this class) |

### Current LSP Model

`pkg/lsp/base/lsp_model.go:61,71` has commented-out type hierarchy fields:
```go
// TypeHierarchyProvider  interface{} `json:"typeHierarchyProvider,omitempty"`
// TypeHierarchyOptions   interface{} `json:"typeHierarchyOptions,omitempty"`
```

### Opportunity

LSP type hierarchy can:
1. Resolve inheritance when tree-sitter only has the type name (not fully qualified)
2. Discover external superclasses/interfaces from dependencies
3. Build complete inheritance trees including framework classes

---

## Proposed Fixes

### Fix 1: Extract Method Name from Java LSP Signatures (P0)

**File:** `internal/model/response.go` or new helper in `pkg/lsp/java_client.go`

```go
// ExtractJavaMethodName extracts just the method name from a Java LSP signature.
// Examples:
//   "findByOwnerId(Long) : List<PetDto>" -> "findByOwnerId"
//   "countByType(String) : long" -> "countByType"
//   "toString()" -> "toString"
//   "simpleMethod" -> "simpleMethod"
func ExtractJavaMethodName(fullSignature string) string {
    // Find the opening parenthesis which marks end of method name
    if idx := strings.Index(fullSignature, "("); idx > 0 {
        return fullSignature[:idx]
    }
    // No parenthesis found, return as-is
    return fullSignature
}
```

**Update `MapToFunctionDependency`:**
```go
func MapToFunctionDependency(call base.CallHierarchyOutgoingCall, lspClient base.LSPClient) FunctionDependency {
    // Extract clean method name for matching
    methodName := call.To.Name
    if javaClient, ok := lspClient.(*lsp.JavaLanguageServerClient); ok {
        methodName = ExtractJavaMethodName(call.To.Name)
    }

    return FunctionDependency{
        Name:          methodName,
        CallLocations: callLocations,
        Definition: FunctionDefinition{
            Name:     methodName,
            FullName: call.To.Name,  // Keep original for display
            // ... rest unchanged
        },
    }
}
```

### Fix 2: Fix matchesFunctionCall Logic (P0)

**File:** `internal/controller/post_process.go`

The current logic is backwards. Change:
```go
func (pp *PostProcessor) matchesFunctionCall(callNode *ast.Node, dependency *model.FunctionDependency) bool {
    if !dependency.IsIn(&callNode.Range) {
        return false
    }

    // Compare method names directly (both should now be simple names)
    return callNode.Name == dependency.Definition.Name
}
```

Or if keeping the HasSuffix approach for qualified names:
```go
func (pp *PostProcessor) matchesFunctionCall(callNode *ast.Node, dependency *model.FunctionDependency) bool {
    if !dependency.IsIn(&callNode.Range) {
        return false
    }

    // Check if call name matches dependency name
    // Handle both simple names and qualified names (e.g., "this.method" vs "method")
    depName := dependency.Definition.Name
    callName := callNode.Name

    return callName == depName ||
           strings.HasSuffix(callName, "."+depName) ||
           strings.HasSuffix(depName, "."+callName)
}
```

### Fix 3: Capture Java Inheritance During Parsing (P1)

**File:** `internal/parse/java_visitor.go`

Update `handleClassDeclaration`:
```go
func (jv *JavaVisitor) handleClassDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
    nameNode := jv.translate.TreeChildByFieldName(tsNode, "name")
    className := ""
    if nameNode != nil {
        className = jv.translate.String(nameNode)
    }

    // Extract superclass
    var superclass string
    superclassNode := jv.translate.TreeChildByKind(tsNode, "superclass")
    if superclassNode != nil {
        // superclass node contains: (superclass "extends" (type_identifier))
        typeNode := jv.translate.TreeChildByKind(superclassNode, "type_identifier")
        if typeNode == nil {
            // Could be generic_type: (generic_type (type_identifier) (type_arguments))
            genericType := jv.translate.TreeChildByKind(superclassNode, "generic_type")
            if genericType != nil {
                typeNode = jv.translate.TreeChildByKind(genericType, "type_identifier")
            }
        }
        if typeNode != nil {
            superclass = jv.translate.String(typeNode)
        }
    }

    // Extract implemented interfaces
    var interfaces []string
    interfacesNode := jv.translate.TreeChildByKind(tsNode, "super_interfaces")
    if interfacesNode != nil {
        typeList := jv.translate.TreeChildByKind(interfacesNode, "type_list")
        if typeList != nil {
            for i := uint32(0); i < typeList.NamedChildCount(); i++ {
                child := typeList.NamedChild(i)
                typeName := jv.extractTypeName(child)
                if typeName != "" {
                    interfaces = append(interfaces, typeName)
                }
            }
        }
    }

    // Build metadata with inheritance info
    metadata := make(map[string]any)
    annotations := jv.extractAnnotations(tsNode)
    if len(annotations) > 0 {
        metadata["annotations"] = annotations
    }
    if superclass != "" {
        metadata["extends"] = superclass
    }
    if len(interfaces) > 0 {
        metadata["implements"] = interfaces
    }

    // ... rest of method unchanged, use metadata
    classNodeID := jv.translate.HandleClassWithMetadata(ctx, scopeID, tsNode, className, methods, nil, metadata)
    // ...
}

// Helper to extract type name from type_identifier or generic_type
func (jv *JavaVisitor) extractTypeName(node *tree_sitter.Node) string {
    if node == nil {
        return ""
    }
    switch node.Kind() {
    case "type_identifier":
        return jv.translate.String(node)
    case "generic_type":
        typeId := jv.translate.TreeChildByKind(node, "type_identifier")
        if typeId != nil {
            return jv.translate.String(typeId)
        }
    case "scoped_type_identifier":
        // e.g., java.util.List
        return jv.translate.String(node)
    }
    return ""
}
```

### Fix 4: Handle Unresolved External Calls Gracefully (P1)

**File:** `internal/controller/post_process.go`

```go
func (pp *PostProcessor) createCallsRelations(ctx context.Context, repo *config.Repository, calls []*ast.Node, dependencies []model.FunctionDependency) error {
    for _, call := range calls {
        dep := pp.findCallInDependency(call, dependencies)
        if dep == nil {
            // Check if this looks like a static method call or common framework method
            if pp.isLikelyExternalCall(call.Name) {
                // Mark as external call without warning
                if call.MetaData == nil {
                    call.MetaData = make(map[string]any)
                }
                call.MetaData["external"] = true
                call.MetaData["unresolved"] = true
                pp.codeGraph.CreateFunctionCall(ctx, call)
                pp.logger.Debug("Marked unresolved call as external",
                    zap.Int64("callNodeId", int64(call.ID)),
                    zap.String("callName", call.Name))
            } else {
                pp.logger.Warn("No matching dependency found for function call",
                    zap.Int64("callNodeId", int64(call.ID)),
                    zap.String("callName", call.Name))
            }
            continue
        }
        // ... rest unchanged
    }
    return nil
}

// isLikelyExternalCall checks if a method name suggests an external/framework call
func (pp *PostProcessor) isLikelyExternalCall(name string) bool {
    // Common JDK static methods
    jdkMethods := map[string]bool{
        "valueOf": true, "parseInt": true, "parseDouble": true,
        "toString": true, "equals": true, "hashCode": true,
        "format": true, "join": true, "of": true,
    }
    if jdkMethods[name] {
        return true
    }

    // Common framework methods
    frameworkMethods := map[string]bool{
        "ok": true, "created": true, "noContent": true, "badRequest": true,  // ResponseEntity
        "emptyList": true, "singletonList": true, "unmodifiableList": true,  // Collections
        "empty": true, "ofNullable": true, "orElse": true,                   // Optional
    }
    return frameworkMethods[name]
}
```

### Fix 5: Add LSP Type Hierarchy Support (P2)

**File:** `pkg/lsp/base/lsp_model.go`

Add type hierarchy models:
```go
// TypeHierarchyItem represents an item in the type hierarchy
type TypeHierarchyItem struct {
    Name           string   `json:"name"`
    Kind           int      `json:"kind"`
    Tags           []int    `json:"tags,omitempty"`
    Detail         string   `json:"detail,omitempty"`
    URI            string   `json:"uri"`
    Range          Range    `json:"range"`
    SelectionRange Range    `json:"selectionRange"`
    Data           any      `json:"data,omitempty"`
}

// TypeHierarchyPrepareParams for textDocument/prepareTypeHierarchy
type TypeHierarchyPrepareParams struct {
    TextDocument TextDocumentIdentifier `json:"textDocument"`
    Position     Position               `json:"position"`
}

// TypeHierarchySupertypesParams for typeHierarchy/supertypes
type TypeHierarchySupertypesParams struct {
    Item TypeHierarchyItem `json:"item"`
}
```

**File:** `pkg/lsp/base_client_lsp.go`

Add type hierarchy methods:
```go
// GetTypeHierarchyItem prepares a type hierarchy item for a position
func (c *BaseClient) GetTypeHierarchyItem(ctx context.Context, uri string, position Position) ([]TypeHierarchyItem, error) {
    params := TypeHierarchyPrepareParams{
        TextDocument: TextDocumentIdentifier{URI: uri},
        Position:     position,
    }

    result, err := c.SendRequest(ctx, "textDocument/prepareTypeHierarchy", params)
    if err != nil {
        return nil, err
    }

    // Parse result into []TypeHierarchyItem
    // ...
}

// GetSupertypes returns the supertypes (extends/implements) for a type
func (c *BaseClient) GetSupertypes(ctx context.Context, item TypeHierarchyItem) ([]TypeHierarchyItem, error) {
    params := TypeHierarchySupertypesParams{Item: item}

    result, err := c.SendRequest(ctx, "typeHierarchy/supertypes", params)
    if err != nil {
        return nil, err
    }

    // Parse result into []TypeHierarchyItem
    // ...
}

// GetSubtypes returns the subtypes (classes that extend/implement) for a type
func (c *BaseClient) GetSubtypes(ctx context.Context, item TypeHierarchyItem) ([]TypeHierarchyItem, error) {
    params := TypeHierarchySupertypesParams{Item: item}

    result, err := c.SendRequest(ctx, "typeHierarchy/subtypes", params)
    if err != nil {
        return nil, err
    }

    // Parse result into []TypeHierarchyItem
    // ...
}
```

### Fix 6: Resolve Inheritance in Post-Processing (P2)

**File:** `internal/controller/post_process.go`

Add new phase for inheritance resolution:
```go
func (pp *PostProcessor) processOneFile(ctx context.Context, repo *config.Repository, fileScope *ast.Node) error {
    language := fileScope.MetaData["language"].(string)
    langType := parse.NewLanguageTypeFromString(language)

    if langType == parse.Go {
        if err := pp.ProcessFakeClasses(ctx, fileScope); err != nil {
            pp.logger.Error("Failed to process fake classes", zap.Error(err))
        }
    }

    // Process function calls
    if err := pp.processFunctionCalls(ctx, repo, fileScope); err != nil {
        return fmt.Errorf("failed to process function calls: %w", err)
    }

    // NEW: Process inheritance for Java
    if langType == parse.Java {
        if err := pp.processInheritance(ctx, repo, fileScope); err != nil {
            pp.logger.Error("Failed to process inheritance", zap.Error(err))
        }
    }

    return nil
}

func (pp *PostProcessor) processInheritance(ctx context.Context, repo *config.Repository, fileScope *ast.Node) error {
    // Find all classes in this file
    classes, err := pp.codeGraph.FindClassesByFile(ctx, fileScope.FileID)
    if err != nil {
        return fmt.Errorf("failed to find classes: %w", err)
    }

    for _, class := range classes {
        // Check if class has extends/implements in metadata
        if class.MetaData == nil {
            continue
        }

        // Process extends
        if extends, ok := class.MetaData["extends"].(string); ok && extends != "" {
            if err := pp.resolveExtends(ctx, repo, class, extends); err != nil {
                pp.logger.Warn("Failed to resolve extends",
                    zap.String("class", class.Name),
                    zap.String("extends", extends),
                    zap.Error(err))
            }
        }

        // Process implements
        if implements, ok := class.MetaData["implements"].([]string); ok {
            for _, iface := range implements {
                if err := pp.resolveImplements(ctx, repo, class, iface); err != nil {
                    pp.logger.Warn("Failed to resolve implements",
                        zap.String("class", class.Name),
                        zap.String("implements", iface),
                        zap.Error(err))
                }
            }
        }
    }

    return nil
}

func (pp *PostProcessor) resolveExtends(ctx context.Context, repo *config.Repository, class *ast.Node, superclassName string) error {
    // First try to find the superclass in the code graph (same repo)
    superclasses, err := pp.codeGraph.FindClassesByName(ctx, repo.Name, superclassName)
    if err == nil && len(superclasses) > 0 {
        // Found in project - create EXTENDS relationship
        return pp.codeGraph.CreateExtendsRelation(ctx, class.ID, superclasses[0].ID)
    }

    // Not found in project - try LSP type hierarchy to get full info
    // This would resolve to external classes and provide full qualified names
    // Mark as external extends
    if class.MetaData == nil {
        class.MetaData = make(map[string]any)
    }
    class.MetaData["extends_external"] = superclassName
    return pp.codeGraph.UpdateClassMetadata(ctx, class)
}
```

---

## Summary of Required Changes

| Priority | Issue | File | Change |
|----------|-------|------|--------|
| **P0** | Name matching | `model/response.go` | Extract method name from Java LSP signatures |
| **P0** | Name matching | `post_process.go` | Fix `matchesFunctionCall` logic |
| **P1** | Inheritance capture | `java_visitor.go` | Extract extends/implements during parsing |
| **P1** | External calls | `post_process.go` | Handle unresolved external calls gracefully |
| **P2** | Type hierarchy | `base/lsp_model.go` | Add TypeHierarchyItem models |
| **P2** | Type hierarchy | `base_client_lsp.go` | Add type hierarchy LSP methods |
| **P2** | Inheritance resolution | `post_process.go` | Use LSP to resolve inheritance |

---

## Testing Recommendations

1. **Unit test for `ExtractJavaMethodName`:**
   - `"findByOwnerId(Long) : List<PetDto>"` → `"findByOwnerId"`
   - `"countByType(String) : long"` → `"countByType"`
   - `"Optional<User> findById(Long)"` → `"Optional<User> findById"` (edge case)
   - `"toString()"` → `"toString"`
   - `"simpleMethod"` → `"simpleMethod"`

2. **Integration test for function call resolution:**
   - Parse Java file with method calls
   - Run post-processing
   - Verify `CALLS_FUNCTION` relationships are created

3. **Integration test for inheritance:**
   - Parse Java file with class extending another class in same project
   - Verify `EXTENDS` relationship is created
   - Parse Java file with class extending external class
   - Verify metadata captures external extends

---

## Appendix: Log Analysis

### Successful LSP Call Hierarchy Request
```
Line 496: Request: textDocument/prepareCallHierarchy at line 43, character 40
Line 502: Response: name="getPetsByOwner(Long) : ResponseEntity<List<PetDto>>"
Line 510: Request: callHierarchy/outgoingCalls
Line 517: Response: to.name="findByOwnerId(Long) : List<PetDto>"
```

### Failed Matching
```
Line 526: WARN - No matching dependency found for "ok"
Line 527: WARN - No matching dependency found for "findByOwnerId"
```

The `findByOwnerId` failure is due to name format mismatch (tree-sitter has `"findByOwnerId"`, LSP returns `"findByOwnerId(Long) : List<PetDto>"`).

The `ok` failure is due to `ResponseEntity.ok()` being an external static method not included in LSP call hierarchy results.

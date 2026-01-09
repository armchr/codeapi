# Unit Test Plan for CodeAPI

This document outlines the unit testing strategy for CodeAPI, focusing on logic-heavy components that would benefit most from test coverage.

## Current State

| Component | Lines | Test Coverage |
|-----------|-------|---------------|
| `internal/parse/*` | ~5,000 | 0% |
| `internal/service/codegraph/*` | ~1,000 | 0% |
| `pkg/lsp/*` | ~3,500 | 0% |
| `internal/config/*` | ~240 | 40% |
| `internal/util/*` | ~380 | ~50% |

**Total estimated coverage: ~3%**

---

## Tier 1: Critical (Highest ROI)

These components have the highest complexity and are most likely to contain bugs. Testing them provides the greatest return on investment.

### 1. Scope & Symbol Resolution (`internal/parse/translate.go`)

**Why Critical:** Most parsing bugs originate here. Core logic for AST translation, scope management, and symbol resolution.

| Function | Description | Test Cases |
|----------|-------------|------------|
| `PushScope()` / `PopScope()` | Scope lifecycle management | Nested scopes, stack underflow, scope cleanup on error |
| `Resolve()` | Multi-level symbol resolution | Symbol in current scope, parent scope, not found, shadowing |
| `ResolveNameChain()` | Field access resolution | `obj.field.method()`, unresolved intermediate, empty chain |
| `GetTreeNodeName()` | Identifier extraction | All 7+ identifier kinds, missing name with fallback |
| `HandleAssignment()` | Data flow tracking | Simple assign, chained assign, compound operators |
| `HandleCall()` | Function call creation | With/without args, method vs function, nested calls |
| `HandleConditional()` | Branch handling | If/else, switch, empty branches, nested conditionals |
| `HandleLoop()` | Loop node creation | For, while, do-while, break/continue handling |

**Edge Cases:**
- Undefined variables with no parent scope
- Empty identifier names requiring fallback extraction
- Nested scopes (functions within classes within modules)
- Field access chains (`obj.field1.field2.method()`)
- Anonymous functions and closures
- Multiple assignments in single statement
- Scope stack underflow/overflow

**Test Fixtures Needed:**
- Minimal AST node mocks
- Mock `CodeGraph` interface
- Sample tree-sitter nodes for each language

---

### 2. Symbol Matching (`pkg/lsp/base/lsp_util.go`)

**Why Critical:** LSP accuracy depends entirely on correct symbol matching. Wrong matches lead to incorrect call graphs.

| Function | Description | Test Cases |
|----------|-------------|------------|
| `MatchLastSegment()` | Qualified name matching | `pkg.Class.Method` matches `Method`, no match, empty input |
| `LastSegment()` | Extract rightmost component | Dotted name, no dots, trailing dot, empty string |
| `MatchExact()` | Exact string matching | Case sensitivity, unicode, whitespace |
| `MatchIgnoreCaseLastSegment()` | Case-insensitive matching | Mixed case, unicode normalization |

**Example Test Structure:**
```go
func TestMatchLastSegment(t *testing.T) {
    tests := []struct {
        name       string
        symbol     string
        nameInFile string
        separator  string
        want       bool
    }{
        {"exact match", "Method", "com.example.MyClass.Method", ".", true},
        {"no match", "Method", "OtherMethod", ".", false},
        {"empty symbol", "", "Something", ".", false},
        {"empty nameInFile", "Method", "", ".", false},
        {"single segment", "Method", "Method", ".", true},
        {"case sensitive", "method", "Method", ".", false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := MatchLastSegment(tt.symbol, tt.nameInFile, tt.separator)
            if got != tt.want {
                t.Errorf("MatchLastSegment(%q, %q, %q) = %v, want %v",
                    tt.symbol, tt.nameInFile, tt.separator, got, tt.want)
            }
        })
    }
}
```

---

### 3. Buffer & Flush Operations (`internal/service/codegraph/code_graph.go`)

**Why Critical:** Database consistency depends on correct buffering. Bugs here cause data loss or corruption.

| Function | Description | Test Cases |
|----------|-------------|------------|
| `InitializeFileBuffers()` | Per-file buffer setup | New file, duplicate file ID, concurrent init |
| `CleanupFileBuffers()` | Flush and memory cleanup | Normal cleanup, cleanup with pending data |
| `FlushNodes()` | Node batch writing | Empty buffer, partial flush, file-specific flush |
| `FlushRelations()` | Relation batch writing | Nodes flushed first (ordering), empty relations |
| `Flush()` | Orchestrated flush | Node-first ordering, partial failure handling |
| `dbRecordToNode()` | Record to Node conversion | All node types, missing fields, metadata parsing |

**Edge Cases:**
- Buffer overflow when exceeding batch size
- Empty buffers (no nodes/relations)
- Flushing non-existent file IDs
- Concurrent flushes from different files
- Metadata preservation during round-trip (Node -> DB -> Node)
- Type conversion failures (invalid JSON, wrong types)
- Partial buffer flushes when first flush fails
- Race conditions in buffer access (`bufferMutex`)

**Test Fixtures Needed:**
- Mock `GraphDB` interface
- Sample node/relation records
- Concurrent access test harness

---

## Tier 2: High Value

### 4. Java Annotation Extraction (`internal/parse/java_visitor.go`)

**Why Important:** Business-critical for Spring Boot and framework analysis.

| Function | Description | Test Cases |
|----------|-------------|------------|
| `extractAnnotations()` | Parse annotations from modifiers | Marker `@Override`, with args `@GetMapping("/path")`, multiple |
| `extractAnnotationArguments()` | Extract annotation parameters | String, integer, boolean, named args `min=1, max=50` |

**Example Test with Tree-Sitter:**
```go
func TestExtractAnnotations(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        wantAnns []map[string]any
    }{
        {
            name: "marker annotation",
            code: `@Override
public void method() {}`,
            wantAnns: []map[string]any{{"name": "Override"}},
        },
        {
            name: "annotation with value",
            code: `@GetMapping("/api/users")
public List<User> getUsers() {}`,
            wantAnns: []map[string]any{
                {"name": "GetMapping", "arguments": map[string]string{"value": "/api/users"}},
            },
        },
        {
            name: "annotation with named args",
            code: `@Size(min = 1, max = 50)
private String name;`,
            wantAnns: []map[string]any{
                {"name": "Size", "arguments": map[string]string{"min": "1", "max": "50"}},
            },
        },
    }
    // Parse with tree-sitter, call extractAnnotations, compare
}
```

---

### 5. File Filtering (`internal/parse/parse.go`)

**Why Important:** Controls what gets parsed. Wrong filtering = missing code or wasted processing.

| Function | Description | Test Cases |
|----------|-------------|------------|
| `DetectLanguage()` | Extension to language mapping | All extensions, unknown, uppercase, no extension |
| `ShouldSkipFile()` | File exclusion logic | node_modules, vendor, .git, binaries, lock files |
| `GetLanguageParser()` | Tree-sitter parser factory | All languages, invalid language |
| `GetLanguageVisitor()` | Visitor factory | All languages, nil logger handling |
| `isAllowedFileExtensionsInRepo()` | Repo language filtering | Match, mismatch, skip_other_languages flag |

**Test Cases for `DetectLanguage()`:**
```go
func TestDetectLanguage(t *testing.T) {
    tests := []struct {
        filename string
        want     string
    }{
        {"main.go", "go"},
        {"App.java", "java"},
        {"script.py", "python"},
        {"component.tsx", "typescript"},
        {"component.jsx", "javascript"},
        {"Program.cs", "csharp"},
        {"README.md", ""},
        {"Makefile", ""},
        {"MAIN.GO", "go"},  // uppercase
        {"noextension", ""},
    }
    // ...
}
```

**Test Cases for `ShouldSkipFile()`:**
```go
func TestShouldSkipFile(t *testing.T) {
    tests := []struct {
        path     string
        repoLang string
        want     bool
    }{
        {"node_modules/lodash/index.js", "javascript", true},
        {"vendor/github.com/pkg/errors/errors.go", "go", true},
        {".git/objects/pack/pack-123.idx", "go", true},
        {"src/main.go", "go", false},
        {"package-lock.json", "javascript", true},
        {"go.sum", "go", true},
        {"image.png", "go", true},
        {"binary.exe", "go", true},
    }
    // ...
}
```

---

### 6. LSP Client Implementations (`pkg/lsp/*_client.go`)

**Why Important:** Each language has specific external module detection and symbol matching rules.

| Client | Key Functions | Test Cases |
|--------|---------------|------------|
| `java_client.go` | `IsExternalModule()` | `.m2/repository/`, `.gradle/caches/`, `target/`, JDK paths |
| `go_client.go` | `IsExternalModule()` | `vendor/`, outside root path |
| `csharp_client.go` | `IsExternalModule()` | `.nuget/packages/`, `/dotnet/`, SDK paths |
| `python_client.go` | `IsExternalModule()` | `site-packages/`, `.venv/`, `dist-packages/` |
| All clients | `LanguageID()` | Correct language for file extensions |
| All clients | `MatchSymbolByName()` | Language-specific matching rules |

**Example Test:**
```go
func TestJavaClient_IsExternalModule(t *testing.T) {
    client := &JavaLanguageServerClient{rootPath: "/project"}

    tests := []struct {
        uri  string
        want bool
    }{
        {"file:///project/src/Main.java", false},
        {"file:///home/user/.m2/repository/org/springframework/spring-core/5.0/spring-core.jar", true},
        {"file:///home/user/.gradle/caches/modules-2/files-2.1/com.google/guava/guava.jar", true},
        {"file:///project/target/classes/Main.class", true},
        {"file:///usr/lib/jvm/java-17/lib/src.zip", true},
    }
    // ...
}
```

---

## Tier 3: Supporting

### 7. Safe Concurrent Map (`internal/util/safe_map.go`)

| Function | Test Cases |
|----------|------------|
| `Get()` | Key exists, key missing, nil value |
| `Set()` | New key, overwrite existing, nil value |
| Concurrent | Multiple goroutines read/write, no data races |

```go
func TestSafeMap_Concurrent(t *testing.T) {
    m := NewSafeMap[string, int]()
    var wg sync.WaitGroup

    // 100 goroutines writing
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            m.Set(fmt.Sprintf("key%d", n), n)
        }(i)
    }

    // 100 goroutines reading
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            m.Get(fmt.Sprintf("key%d", n))
        }(i)
    }

    wg.Wait()
}
```

---

### 8. Configuration Loading (`internal/config/config.go`)

Extend existing tests for `expandEnvVars()` to cover full config loading.

| Function | Test Cases |
|----------|------------|
| `LoadConfig()` | Valid config, missing file, invalid YAML, merge precedence |
| `GetRepository()` | Found, not found, disabled repo |
| `validateRepositories()` | Valid repos, missing required fields, duplicate names |

---

### 9. Path Utilities (`internal/util/utils.go`)

| Function | Test Cases |
|----------|------------|
| `ToUri()` | Absolute path, relative path, Windows path, spaces in path |
| `ExtractPathFromURI()` | Valid file:// URI, missing scheme, encoded characters |
| `ToRelativePath()` | Inside root, outside root, same as root |

---

## Test Infrastructure

### Mock Interfaces

Create mock implementations for external dependencies:

```go
// internal/service/codegraph/mock_graph_db.go
type MockGraphDB struct {
    nodes       map[ast.NodeID]*ast.Node
    relations   []Relation
    createCalls int
}

func (m *MockGraphDB) CreateNode(ctx context.Context, node *ast.Node) error {
    m.nodes[node.ID] = node
    m.createCalls++
    return nil
}

// pkg/lsp/mock_lsp_client.go
type MockLSPClient struct {
    symbols      []DocumentSymbol
    initialized  bool
    initError    error
}
```

### Test Fixtures Directory

```
tests/
├── fixtures/
│   ├── go/
│   │   ├── simple_function.go
│   │   ├── interface_impl.go
│   │   └── generics.go
│   ├── java/
│   │   ├── simple_class.java
│   │   ├── spring_controller.java
│   │   └── annotations.java
│   ├── python/
│   │   ├── simple_module.py
│   │   ├── decorators.py
│   │   └── async_functions.py
│   └── golden/
│       ├── simple_function.go.json    # Expected AST
│       └── spring_controller.java.json
```

### Table-Driven Test Pattern

All tests should follow the table-driven pattern already used in existing tests:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"empty input", emptyInput, zeroValue, true},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Function() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Function() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## Implementation Phases

| Phase | Components | Files Created | Tests | Status |
|-------|------------|---------------|-------|--------|
| **Phase 1** | LSP utilities, SafeMap | `pkg/lsp/base/lsp_util_test.go`, `internal/util/safe_map_test.go` | 15 | Done |
| **Phase 2** | Scope management | `internal/parse/translate_test.go` | 16 | Done |
| **Phase 3** | Java annotations | `internal/parse/java_visitor_test.go` | 10 | Done |
| **Phase 4** | Code graph buffering | `internal/service/codegraph/code_graph_test.go` | ~25 | Pending |
| **Phase 5** | File filtering | `internal/parse/parse_test.go` | ~20 | Pending |
| **Phase 6** | LSP clients | `pkg/lsp/java_client_test.go`, etc. | ~20 | Pending |

**Completed: 41 tests** | **Remaining: ~65 tests**

---

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/parse/...

# Run tests with verbose output
go test -v ./pkg/lsp/base/...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

## Success Criteria

- **Phase 1 Complete:** All Tier 3 tests passing, 10%+ coverage
- **Phase 2 Complete:** Scope management tests passing, 25%+ coverage
- **Phase 3 Complete:** Java annotation tests passing, 35%+ coverage
- **Phase 4 Complete:** Code graph tests passing, 50%+ coverage
- **Phase 5 Complete:** File filtering tests passing, 60%+ coverage
- **Phase 6 Complete:** LSP client tests passing, 70%+ coverage

Target: **70% code coverage** on logic-heavy components within 6 phases.

---

## Completed Work

### Phase 1: LSP Utilities & SafeMap (Done)

**`pkg/lsp/base/lsp_util_test.go`** - 7 tests:
- `TestLastSegment` - Extract rightmost segment from dotted names
- `TestMatchLastSegment` - Match symbols by last segment
- `TestMatchExact` - Exact string matching
- `TestMatchIgnoreCase` - Case-insensitive matching
- `TestMatchIgnoreCaseLastSegment` - Case-insensitive last segment matching
- `TestRangeInRange` - LSP range containment
- `TestExtractJavaMethodName` - Java method name extraction from signatures

**`internal/util/safe_map_test.go`** - 8 tests:
- `TestNewSafeMap` - Map creation
- `TestSafeMap_SetAndGet` - Basic set/get operations
- `TestSafeMap_Overwrite` - Value overwriting
- `TestSafeMap_StringValues` - String value handling
- `TestSafeMap_StructValues` - Struct value handling
- `TestSafeMap_Concurrent` - Concurrent read/write safety
- `TestSafeMap_ConcurrentReadWrite` - Mixed concurrent operations
- `TestSafeMap_EmptyKey` - Empty string key handling

### Phase 2: Scope Management (Done)

**`internal/parse/translate_test.go`** - 16 tests:
- `TestNewSymbol` - Symbol creation
- `TestSymbol_AddField` - Field addition to symbols
- `TestSymbol_GetField` - Field retrieval
- `TestNewScope` - Scope creation with/without RHS
- `TestScope_AddSymbol` - Symbol registration in scope
- `TestScope_GetSymbol` - Symbol lookup
- `TestScope_Resolve` - Multi-level symbol resolution with shadowing
- `TestScope_NotContainedNodes` - Not-contained node tracking
- `TestScope_RhsVars` - RHS variable tracking
- `TestNewTranslateFromSyntaxTree` - Translator initialization
- `TestTranslateFromSyntaxTree_PushScope` - Scope stack push
- `TestTranslateFromSyntaxTree_PopScope` - Scope stack pop
- `TestTranslateFromSyntaxTree_PopScope_Underflow` - Stack underflow handling
- `TestTranslateFromSyntaxTree_PopScope_NotContainedNodes` - Node transfer on pop
- `TestTranslateFromSyntaxTree_NextNodeID` - Node ID generation
- `TestTranslateFromSyntaxTree_ScopeStackIntegration` - Full scope lifecycle

### Phase 3: Java Annotations (Done)

**`internal/parse/java_visitor_test.go`** - 10 tests:
- `TestExtractAnnotations_MarkerAnnotation` - `@Override` style annotations
- `TestExtractAnnotations_WithStringValue` - `@GetMapping("/path")` annotations
- `TestExtractAnnotations_WithNamedArgs` - `@Size(min=1, max=50)` annotations
- `TestExtractAnnotations_Multiple` - Multiple annotations on one element
- `TestExtractAnnotations_NoAnnotations` - Elements without annotations
- `TestExtractAnnotations_ClassLevel` - Class-level annotations
- `TestExtractAnnotationArguments_SingleString` - Single string argument parsing
- `TestExtractAnnotationArguments_NamedPairs` - Named argument pairs
- `TestExtractAnnotationArguments_IntegerValue` - Integer argument values
- `TestJavaVisitor_NilNode` - Nil node handling

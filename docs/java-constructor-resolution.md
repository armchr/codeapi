# Java Constructor Resolution

This document describes the approaches considered for resolving `new ClassName(args)` constructor calls to their actual constructor definitions, and the approach we implemented.

## Problem Statement

When parsing Java code, we encounter two types of constructor-related nodes:

1. **Constructor declarations** - The actual constructor method in a class:
   ```java
   public class Pet {
       public Pet(String name) { ... }  // Constructor declaration
   }
   ```

2. **Constructor calls (object creation)** - `new` expressions that invoke constructors:
   ```java
   Pet pet = new Pet("Buddy");  // Constructor call
   ```

The goal is to create `CALLS_FUNCTION` relationships between constructor calls and their corresponding constructor declarations.

## Metadata Markers

During parsing, both are marked with `is_constructor: true` in their metadata:

- **Constructor declarations**: Created via `handleConstructorDeclaration` in `java_visitor.go`
- **Constructor calls**: Created via `handleObjectCreationExpression` in `java_visitor.go`

## Resolution Options Considered

### Option 1: LSP textDocument/definition

**Approach**: Use LSP's `textDocument/definition` request on the type name position in the `new` expression.

**Pros**:
- Standard LSP approach
- Works for external/library types

**Cons**:
- Returns the class definition location, not the specific constructor
- Requires additional logic to find the matching constructor by parameter count/types
- Extra LSP round-trip for each constructor call

**Example**:
```
Request: textDocument/definition at position of "Pet" in "new Pet(...)"
Response: Location of class Pet declaration (not constructor)
```

### Option 2: LSP textDocument/references (Reverse Lookup)

**Approach**: From each constructor declaration, use `textDocument/references` to find all call sites.

**Pros**:
- Accurate results from LSP
- Handles overloaded constructors correctly

**Cons**:
- Inefficient for large codebases (N queries for N constructors)
- Requires iterating all constructors first
- High LSP server load

### Option 3: LSP Call Hierarchy API

**Approach**: Use `callHierarchy/incomingCalls` from the constructor declaration to find callers.

**Pros**:
- Designed for exactly this purpose
- Part of LSP 3.16+ specification

**Cons**:
- Requires knowing the constructor location first
- Not all LSP servers implement call hierarchy fully for constructors
- Eclipse JDT.LS has inconsistent support for constructor call hierarchy

### Option 4: Code Graph Resolution (Chosen Approach)

**Approach**: Resolve constructor calls using the existing code graph without additional LSP calls.

**Algorithm**:
1. Find all function calls marked with `is_constructor: true`
2. For each constructor call:
   - Extract the class name from the call (e.g., "Pet" from `new Pet()`)
   - Find matching class nodes in the repository by name
   - Find constructors of that class (functions with `is_constructor: true` contained by the class)
   - Create `CALLS_FUNCTION` relationship

**Pros**:
- No additional LSP calls required
- Fast - uses existing indexed data
- Works for all constructors in the codebase
- Consistent with how we handle other relationships

**Cons**:
- Cannot resolve constructors for external/library classes (acceptable - we mark those as external)
- May have ambiguity with same-named classes in different packages (mitigated by package matching)

## Chosen Implementation

We chose **Option 4: Code Graph Resolution** for the following reasons:

1. **Performance**: No additional LSP round-trips during post-processing
2. **Consistency**: Follows the same pattern as inheritance resolution
3. **Simplicity**: Uses data already available in the code graph
4. **Reliability**: Not dependent on LSP server's constructor support

### Implementation Details

Location: `internal/controller/post_process.go`

The `processConstructorCalls` function:

1. Finds all function calls with `is_constructor: true` metadata in a file
2. For each constructor call:
   - Extracts the class name (the call's name field contains the type being constructed)
   - Searches for classes with that name in the repository
   - For each matching class, finds its constructors (functions with `is_constructor: true`)
   - Creates `CALLS_FUNCTION` relationship between the call and the constructor

### Handling Ambiguity

When multiple classes have the same name (different packages):
1. Prefer class in the same package/module as the caller
2. If no match, use the first found (with a warning log)

When multiple constructors exist (overloading):
- Currently matches by name only (all constructors of the class)
- Future enhancement: Match by parameter count

### External Constructors

Constructor calls for classes not in the repository (e.g., `new ArrayList<>()`) are:
1. Detected (no matching class found)
2. Marked as external in the call's metadata
3. Logged at debug level

## Future Enhancements

1. **Parameter matching**: Match constructor calls to specific overloaded constructors by parameter count
2. **Import resolution**: Use import statements to resolve ambiguous class names
3. **Generic handling**: Better support for generic constructor calls like `new ArrayList<String>()`

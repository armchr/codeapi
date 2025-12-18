# Parser Test Repositories

This directory contains test repositories for validating the bot-go parser across different programming languages. Each repository implements a calculator application using language-specific idioms and syntax patterns.

## Quick Start

```bash
# From the project root directory
./run_test.sh <repo-name> --build-index    # Build code graph
./run_test.sh <repo-name> --test-dump      # Dump graph for inspection
./run_test.sh <repo-name> --clean          # Clean up DB entries
./run_test.sh <repo-name> --all            # Run all operations
```

## Test Repositories

| Repository | Language | Files | Description |
|------------|----------|-------|-------------|
| [python-calculator](#python-calculator) | Python 3.10+ | 5 | Modern Python with type hints, dataclasses, async |
| [go-calculator](#go-calculator) | Go 1.21+ | 4 | Generics, interfaces, concurrency patterns |
| [typescript-calculator](#typescript-calculator) | TypeScript 5.x | 7 | Advanced types, generics, async patterns |
| [java-modern-calculator](#java-modern-calculator) | Java 17+ | 4 | Records, sealed classes, pattern matching |
| [java8-calculator](#java8-calculator) | Java 8 | 4 | Traditional Java without modern features |

---

## Python Calculator

**Path:** `repos/python-calculator/`
**Language:** Python 3.10+
**Files:** 5

### Structure
```
python-calculator/
├── main.py                    # Entry point, CLI, async processing
└── operations/
    ├── __init__.py            # Package exports, re-exports
    ├── basic.py               # Basic arithmetic, lambdas
    ├── advanced.py            # Classes, inheritance, generics
    └── utils.py               # Decorators, utilities
```

### Constructs Covered

| Category | Constructs |
|----------|------------|
| **Imports** | Absolute imports, relative imports (`from .module`), wildcard imports (`from X import *`), conditional imports (`try/except ImportError`), `__all__` exports |
| **Functions** | Regular functions, lambda expressions, async/await, generators, variadic args (`*args`, `**kwargs`), default parameters |
| **Classes** | ABC with `@abstractmethod`, dataclasses (`@dataclass`), generic classes (`Generic[T]`), `__enter__`/`__exit__` (context managers), `__iter__` (iterators), class/static methods, properties with setters |
| **Decorators** | `@property`, `@classmethod`, `@staticmethod`, `@abstractmethod`, `@dataclass`, custom decorators (`@log_call`, `@memoize`, `@retry`), decorator factories with parameters |
| **Type Hints** | `Union`, `Optional`, `TypeVar`, `Generic`, `Callable`, type aliases, `list[T]` (PEP 585) |
| **Control Flow** | `if/elif/else`, `match/case` (Python 3.10+), `for` loops, `while` loops, `try/except/finally`, `with` statements, list/dict/set comprehensions, generator expressions |
| **Async** | `async def`, `await`, `asyncio.gather`, async context managers |
| **Other** | f-strings, walrus operator (`:=`), `Enum`, closures, higher-order functions |

---

## Go Calculator

**Path:** `repos/go-calculator/`
**Language:** Go 1.21+
**Files:** 4 + go.mod

### Structure
```
go-calculator/
├── go.mod                     # Module definition, dependencies
├── main.go                    # Entry point, CLI, signal handling
└── operations/
    ├── basic.go               # Generic functions, type constraints
    ├── advanced.go            # Structs, interfaces, concurrency
    └── utils.go               # Utilities, Result type, Observer
```

### Constructs Covered

| Category | Constructs |
|----------|------------|
| **Imports** | Standard library, external packages (`github.com/...`), relative package imports, grouped imports, dot imports |
| **Functions** | Regular functions, methods (value/pointer receivers), variadic functions (`...T`), closures, function types as parameters/returns, deferred calls (`defer`) |
| **Generics** | Type parameters (`[T any]`), type constraints (`Number interface`), generic functions, generic structs |
| **Structs** | Struct definitions, embedded structs, struct tags, constructor functions (`NewX`), builder pattern |
| **Interfaces** | Interface definitions, interface composition, empty interface (`any`), type assertions, type switches |
| **Concurrency** | Goroutines (`go func()`), channels, `sync.Mutex`/`RWMutex`, `sync.WaitGroup`, `context.Context`, `select` statement |
| **Control Flow** | `if/else`, `switch` (expression and type), `for` loops (3-clause, range, infinite), `break`/`continue`, labeled statements |
| **Error Handling** | Multiple return values, error wrapping, custom error types, `errors.Is`/`errors.As` |
| **Patterns** | Functional options, Result type (like Rust), Observer pattern, factory functions |
| **Other** | Constants (`const`), `iota`, init functions, package-level variables |

---

## TypeScript Calculator

**Path:** `repos/typescript-calculator/`
**Language:** TypeScript 5.x
**Files:** 7 + package.json + tsconfig.json

### Structure
```
typescript-calculator/
├── package.json               # Dependencies, scripts
├── tsconfig.json              # TypeScript configuration
└── src/
    ├── main.ts                # Entry point, CLI, readline
    ├── operations/
    │   ├── index.ts           # Barrel exports
    │   ├── types.ts           # Type definitions, interfaces
    │   ├── basic.ts           # Basic operations, currying
    │   └── advanced.ts        # Classes, mixins, patterns
    └── utils/
        └── helpers.ts         # Utilities, event emitter
```

### Constructs Covered

| Category | Constructs |
|----------|------------|
| **Imports/Exports** | Named imports/exports, default imports/exports, type imports (`import type`), barrel exports (`index.ts`), re-exports, namespace imports (`import * as`) |
| **Types** | Type aliases, union types (`A \| B`), intersection types (`A & B`), literal types, template literal types, conditional types, mapped types, `keyof`, `typeof`, `infer` |
| **Generics** | Generic functions, generic classes, generic interfaces, type constraints (`extends`), default type parameters |
| **Interfaces** | Interface declarations, optional properties, readonly properties, index signatures, interface extension, interface merging |
| **Classes** | Class declarations, inheritance (`extends`), abstract classes, access modifiers (`public`/`private`/`protected`), static members, getters/setters, constructor overloading |
| **Functions** | Arrow functions, async/await, generators (`function*`), async generators (`async function*`), overload signatures, rest parameters, destructuring parameters |
| **Enums** | String enums, numeric enums, const enums, `enum` with `auto()` |
| **Control Flow** | `if/else`, `switch`, `for`/`for...of`/`for...in`, `while`, `try/catch/finally`, optional chaining (`?.`), nullish coalescing (`??`) |
| **Patterns** | Discriminated unions, Result type (`Ok`/`Err`), Observer pattern, Singleton, Mixin pattern, factory functions |
| **Other** | `as const`, `satisfies` operator (TS 4.9+), decorators (experimental), Promises, `Symbol` |

---

## Java Modern Calculator

**Path:** `repos/java-modern-calculator/`
**Language:** Java 17+
**Files:** 4 + pom.xml

### Structure
```
java-modern-calculator/
├── pom.xml                    # Maven configuration
└── src/main/java/com/example/calculator/
    ├── Main.java              # Entry point, CLI, CompletableFuture
    ├── operations/
    │   ├── BasicOperations.java    # Static methods, records
    │   └── AdvancedCalculator.java # Sealed classes, patterns
    └── util/
        └── ValidationUtils.java    # Utilities, Result type
```

### Constructs Covered

| Category | Constructs |
|----------|------------|
| **Imports** | Single imports, wildcard imports, static imports (`import static`), package organization |
| **Records** | Record declarations, compact constructors, record methods, records implementing interfaces |
| **Sealed Classes** | `sealed` classes/interfaces, `permits` clause, `non-sealed` subclasses, pattern matching with sealed types |
| **Classes** | Regular classes, abstract classes, inner classes, static nested classes, anonymous classes, builder pattern |
| **Interfaces** | Interface declarations, default methods, static methods, private methods (Java 9+), functional interfaces |
| **Generics** | Generic classes, generic methods, bounded type parameters (`<T extends X>`), wildcards (`?`, `? extends`, `? super`) |
| **Methods** | Instance methods, static methods, varargs (`T...`), method references (`Class::method`), factory methods |
| **Control Flow** | `if/else`, enhanced `switch` expressions (arrows, `yield`), pattern matching `switch` (Java 17+), `for`/enhanced `for`, `while`/`do-while`, `try-with-resources` |
| **Lambdas** | Lambda expressions, method references, functional interfaces (`Function`, `Consumer`, `Predicate`, `Supplier`) |
| **Streams** | `Stream.of`, `stream()`, `map`, `filter`, `reduce`, `collect`, `flatMap`, parallel streams |
| **Concurrency** | `CompletableFuture`, `ExecutorService`, `synchronized`, thread pools |
| **Other** | `var` (local variable type inference), text blocks (`"""`), `Optional`, `@Override`, `@FunctionalInterface` |

---

## Java 8 Calculator

**Path:** `repos/java8-calculator/`
**Language:** Java 8
**Files:** 4 + pom.xml

### Structure
```
java8-calculator/
├── pom.xml                    # Maven configuration (Java 8)
└── src/main/java/com/example/calculator/
    ├── Main.java              # Entry point, CLI, Future
    ├── operations/
    │   ├── BasicOperations.java    # Static methods, POJOs
    │   └── AdvancedCalculator.java # Traditional classes
    └── util/
        └── ValidationUtils.java    # Utilities, Result pattern
```

### Constructs Covered

This repository implements the same functionality as `java-modern-calculator` but uses Java 8 syntax only, demonstrating the contrast with modern Java features.

| Category | Constructs |
|----------|------------|
| **Imports** | Single imports, wildcard imports, static imports |
| **Classes** | Regular classes (no records), POJOs with getters/setters, abstract classes, inner classes, anonymous inner classes, builder pattern |
| **Interfaces** | Interface declarations, default methods (Java 8), static methods, no private methods |
| **Generics** | Generic classes, generic methods, bounded type parameters, wildcards |
| **Methods** | Instance methods, static methods, varargs, explicit type declarations (no `var`) |
| **Control Flow** | Traditional `if/else`, traditional `switch` (no arrows/yield), `for`/enhanced `for`, `while`/`do-while`, `try-catch-finally`, `try-with-resources` |
| **Lambdas** | Lambda expressions, method references, functional interfaces, but also anonymous inner classes for comparison |
| **Collections** | `ArrayList`, `HashMap`, `Arrays.asList`, manual iteration alongside streams |
| **Concurrency** | `Future`, `Callable`, `ExecutorService`, `synchronized`, no `CompletableFuture` chaining |
| **Differences from Modern Java** | No records (use POJOs), no sealed classes, no pattern matching switch, no text blocks, no `var`, explicit HashMap initialization (no `Map.of`) |

---

## Configuration

The test repositories are configured in `tests/source.yaml`:

```yaml
source:
  repositories:
    - name: python-calculator
      path: /Users/anindya/src/armchr/bot-go/tests/repos/python-calculator
      language: python
    - name: go-calculator
      path: /Users/anindya/src/armchr/bot-go/tests/repos/go-calculator
      language: go
    - name: typescript-calculator
      path: /Users/anindya/src/armchr/bot-go/tests/repos/typescript-calculator
      language: typescript
    - name: java-modern-calculator
      path: /Users/anindya/src/armchr/bot-go/tests/repos/java-modern-calculator
      language: java
    - name: java8-calculator
      path: /Users/anindya/src/armchr/bot-go/tests/repos/java8-calculator
      language: java
```

## Adding New Test Repositories

To add a new test repository:

1. Create the repository structure in `tests/repos/<name>/`
2. Add the repository to `tests/source.yaml`
3. Update the `AVAILABLE_REPOS` array in `run_test.sh`
4. Document the repository in this README

## Validating Parser Output

After running `--build-index`, you can inspect the generated code graph:

1. **Neo4j Browser**: Connect to `http://localhost:7474` and run Cypher queries
2. **Test Dump**: Use `--test-dump` to output the graph to a file
3. **API**: Query the REST API endpoints to verify relationships

Example Cypher queries:
```cypher
// List all files in a repository
MATCH (f:FileScope {repository: 'python-calculator'})
RETURN f.relativePath ORDER BY f.relativePath

// List all functions in a file
MATCH (f:FileScope)-[:CONTAINS*]->(fn:Function)
WHERE f.relativePath = 'operations/basic.py'
RETURN fn.name, fn.range

// Find function calls
MATCH (caller:Function)-[:CALLS]->(callee:FunctionCall)
RETURN caller.name, callee.name, callee.range
```

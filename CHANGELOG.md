# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Java LSP support** with Eclipse JDT Language Server
  - Full LSP integration for semantic analysis (call hierarchies, symbol resolution)
  - Java LSP client (`pkg/lsp/java_client.go`) with external module detection for Maven, Gradle, and JDK dependencies
  - Eclipse JDT.LS bundled in `assets/` folder as tar.gz archive
  - Launch script (`scripts/javalsp.sh`) for LSP server execution

- **Java annotation extraction**
  - Annotations automatically captured from classes, interfaces, records, enums, methods, and constructors
  - Stored as JSON strings in node metadata with `name` and `arguments` fields
  - Supports marker annotations (`@Override`), single-value (`@GetMapping("/path")`), and multi-value (`@Size(min=1, max=50)`) annotations
  - Enables Cypher queries to find code by framework annotations (e.g., Spring Boot controllers)

### Fixed

- **Java chained method call parsing** - Fixed `handleMethodInvocation` to recursively traverse nested method invocations
  - Chained calls like `repository.findById(id).stream().map(this::toDto).collect(...)` now correctly create FunctionCall nodes for all methods in the chain
  - Previously, only the outermost call (e.g., `collect`) was captured; inner calls (`findById`, `stream`, `map`) were missed
  - Enables proper CALLS_FUNCTION relationship tracking for repository pattern and fluent API usage

- **Java external module detection** - Fixed `IsExternalModule` check that incorrectly marked all Java files as external
  - The `/java/` path check was matching standard Maven project structure (`src/main/java/...`)
  - Changed to specific JDK location patterns (`/Library/Java/`, `/usr/lib/jvm/`) instead
  - CALLS_FUNCTION edges are now correctly created for calls to repository interfaces and other internal project classes

### Changed

- Updated `tests/run_tests.sh` to suppress codeapi log output (logs written to file only)
- Test script now prints full command with all arguments for easier debugging

### Documentation

- Added Java support section to README with setup instructions
- Added Java Annotations in Metadata section with examples and Cypher queries
- Updated Supported Languages table with LSP server information

## [1.0.0] - 2025-01-15

### Added

- **Multi-language code parsing** using tree-sitter
  - Go (including generics, interfaces, and concurrency primitives)
  - Python (async/await, decorators, type hints)
  - Java (modern Java features including records and sealed classes)
  - TypeScript (generics, advanced types, decorators)
  - JavaScript (ES6+ features)

- **Code Graph construction** with Neo4j
  - AST-based node creation (FileScope, Class, Function, Field, Variable, Block)
  - Relationship tracking (CONTAINS, CALLS, USES, DEFINES, INHERITS_FROM)
  - Batch writing support for improved performance
  - File-level buffering for parallel processing

- **Vector embeddings** with Qdrant and Ollama
  - Hierarchical code chunking (file, class, function, block levels)
  - Semantic similarity search
  - Support for multiple embedding models

- **REST API** with Gin framework
  - Index building endpoints (`/api/v1/buildIndex`, `/api/v1/indexFile`)
  - Semantic code search (`/api/v1/searchSimilarCode`)
  - Function dependencies (`/api/v1/functionDependencies`)
  - Code exploration APIs (`/codeapi/v1/`)
  - Graph analysis endpoints (call graphs, inheritance, impact analysis)
  - Raw Cypher query execution

- **CLI mode** for batch indexing
  - Multi-repository indexing support
  - Git HEAD mode for committed-only analysis
  - Code graph dump for debugging
  - Database cleanup option

- **File version tracking** with MySQL
  - Unique FileID generation
  - SHA256-based change detection
  - Commit ID association for git repositories

- **Docker support**
  - Dockerfile for containerized deployment
  - Docker Compose for full stack deployment

- **Test repositories**
  - Multi-language calculator examples
  - Comprehensive language feature coverage

### Infrastructure

- Makefile with common build targets
- Configuration via YAML files with environment variable support
- Zap-based structured logging
- Parallel file processing with configurable thread count

## [0.1.0] - Initial Development

### Added

- Initial project structure
- Basic tree-sitter integration
- Neo4j connection handling
- MySQL file tracking
- Core API endpoints

---

## Release Notes

### v1.0.0

This is the first public release of CodeAPI. It provides a complete solution for:

1. **Code Understanding**: Parse and analyze codebases in 5 major languages
2. **Knowledge Graph**: Build a queryable graph of code structure and relationships
3. **Semantic Search**: Find similar code using vector embeddings
4. **API Access**: RESTful APIs for integration with other tools

#### Known Limitations

- LSP integration requires external language servers to be installed
- Large repositories may require tuning of batch sizes and thread counts
- Vector embeddings require Ollama to be running locally

#### Migration Notes

This is the initial release. No migration required.

---

[1.0.0]: https://github.com/armchr/codeapi/releases/tag/v1.0.0
[0.1.0]: https://github.com/armchr/codeapi/releases/tag/v0.1.0

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

# CodeAPI

A multi-language code analysis and indexing platform that builds semantic code graphs and enables intelligent code search. CodeAPI parses source code using tree-sitter, constructs a knowledge graph in Neo4j, and optionally generates vector embeddings for semantic similarity search.

## Features

- **Multi-Language Support**: Go, Python, Java, TypeScript, and JavaScript
- **Code Graph Construction**: Builds a comprehensive knowledge graph capturing functions, classes, variables, call relationships, inheritance, and data flow
- **Semantic Code Search**: Vector embeddings enable similarity-based code search
- **Rich Query API**: REST endpoints for code exploration, call graph analysis, and impact assessment
- **Flexible Indexing**: Server mode for on-demand analysis or CLI mode for batch processing
- **Git Integration**: Optional git HEAD mode to index only committed versions

## Prerequisites

- **Go 1.23+**
- **Neo4j 4.x or 5.x** (for code graph storage)
- **MySQL 8.x** (for file version tracking)
- **Qdrant** (optional, for vector embeddings)
- **Ollama** (optional, for embedding generation)

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/armchr/codeapi.git
cd codeapi
make build
```

### 2. Configure

```bash
# Copy example configuration files
cp config/app.yaml.example config/app.yaml
cp config/source.yaml.example config/source.yaml

# Edit config/app.yaml with your database credentials
# Edit config/source.yaml with your repositories
```

### 3. Start Dependencies

```bash
# Using Docker Compose (recommended)
docker-compose up -d neo4j mysql

# Or start services manually
# Neo4j: http://localhost:7474 (bolt://localhost:7687)
# MySQL: localhost:3306
```

### 4. Run

```bash
# Server mode
make run

# Or directly
./bin/codeapi -app=config/app.yaml -source=config/source.yaml
```

The API will be available at `http://localhost:8181`.

## Building

```bash
# Build main binary
make build

# Build with all dependencies
make deps && make build

# Run tests
make test

# Clean build artifacts
make clean
```

## Configuration

### Application Configuration (config/app.yaml)

```yaml
app:
  port: 8181                    # HTTP server port
  codegraph: true               # Enable Neo4j code graph
  num_file_threads: 5           # Parallel file processing threads
  max_concurrent_file_processing: 5

neo4j:
  uri: "bolt://localhost:7687"
  username: "neo4j"
  password: "your-password"

mysql:
  host: "localhost"
  port: 3306
  username: "root"
  password: "your-password"
  database: "codeapi"

qdrant:                         # Optional: for vector embeddings
  host: "localhost"
  port: 6334

ollama:                         # Optional: for embedding generation
  url: "http://localhost:11434"
  model: "nomic-embed-text"
  dimension: 768

index_building:
  enable_code_graph: true       # Build code graph
  enable_embeddings: false      # Generate embeddings

code_graph:
  enable_batch_writes: false    # Batch writes (faster for large repos)
  batch_size: 10
```

### Repository Configuration (config/source.yaml)

```yaml
source:
  repositories:
    - name: my-project
      path: /path/to/your/project
      language: go              # go, python, java, typescript, javascript
      disabled: false
      skip_other_languages: false
```

## CLI Commands

### Server Mode (Default)

```bash
./bin/codeapi -app=config/app.yaml -source=config/source.yaml
```

### Build Index (CLI Mode)

```bash
# Index a single repository
./bin/codeapi -app=config/app.yaml -source=config/source.yaml -build-index=my-repo

# Index multiple repositories
./bin/codeapi -build-index=repo1 -build-index=repo2 -build-index=repo3

# Index using git HEAD (committed versions only)
./bin/codeapi -build-index=my-repo -head

# Dump code graph after indexing (for debugging)
./bin/codeapi -build-index=my-repo -test-dump=output.json

# Clean up database entries after indexing
./bin/codeapi -build-index=my-repo -clean
```

### Using Make

```bash
# Build index for a repository
make build-index REPO=my-repo

# Build index using git HEAD
make build-index-head REPO=my-repo

# Index multiple repos
make build-index REPO="repo1 repo2 repo3"
```

### Command-Line Flags

| Flag | Description |
|------|-------------|
| `-app` | Path to application config file (default: `app.yaml`) |
| `-source` | Path to source/repository config file (default: `source.yaml`) |
| `-workdir` | Working directory for temporary files |
| `-build-index` | Repository name to index (repeatable for multiple repos) |
| `-head` | Use git HEAD version instead of working directory |
| `-test-dump` | Output file path for dumping code graph (debugging) |
| `-clean` | Clean up all DB entries for the repository after processing |
| `-test` | Run in LSP test mode |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP REST API (Gin)                       │
│                    Port 8181 (default)                       │
├─────────────────────────────────────────────────────────────┤
│     RepoController      │      CodeAPIController            │
│  (Indexing & Search)    │   (Graph Queries & Analysis)      │
├─────────────────────────────────────────────────────────────┤
│                   IndexBuilder                               │
│           (Parallel File Processing Pipeline)                │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Tree-Sitter Parsers                     │   │
│  │    Go │ Python │ Java │ TypeScript │ JavaScript     │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────┐   │
│  │     Neo4j       │ │     Qdrant      │ │    MySQL    │   │
│  │   Code Graph    │ │   Embeddings    │ │  File IDs   │   │
│  └─────────────────┘ └─────────────────┘ └─────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Components

| Component | Responsibility |
|-----------|----------------|
| **RepoController** | Handles indexing requests and code search |
| **CodeAPIController** | Provides code graph query and analysis APIs |
| **IndexBuilder** | Orchestrates parallel file processing |
| **CodeGraphProcessor** | Parses files and builds Neo4j graph |
| **EmbeddingProcessor** | Generates vector embeddings for code chunks |
| **CodeGraph** | Neo4j interface for storing code structure |
| **FileVersionRepository** | MySQL-based file tracking with unique IDs |

### Code Graph Model

**Node Types:**
- `FileScope` - Source file
- `Class` - Class, struct, or interface
- `Function` - Function or method
- `Field` - Class field or property
- `Variable` - Local variable
- `Block` - Code block (conditional, loop)
- `Import` - Import statement

**Relationship Types:**
- `CONTAINS` - Hierarchical containment
- `CALLS` - Function invocation
- `USES` - Variable/field usage
- `DEFINES` - Variable definition
- `INHERITS_FROM` - Class inheritance
- `IMPLEMENTS` - Interface implementation

## API Reference

### Base URLs

- **Indexing & Search API**: `/api/v1/`
- **Code Analysis API**: `/codeapi/v1/`

### API Endpoint Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | [`/api/v1/health`](#health-check) | Health check |
| `POST` | [`/api/v1/buildIndex`](#build-index) | Build repository index |
| `POST` | [`/api/v1/indexFile`](#index-file) | Index specific files |
| `POST` | [`/api/v1/searchSimilarCode`](#search-similar-code) | Semantic code search |
| `POST` | [`/api/v1/functionDependencies`](#get-function-dependencies) | Get function call dependencies |
| `POST` | [`/api/v1/processDirectory`](#process-directory) | Process directory for embeddings |
| `GET` | [`/codeapi/v1/repos`](#list-repositories) | List indexed repositories |
| `POST` | [`/codeapi/v1/files`](#list-files) | List files in repository |
| `POST` | [`/codeapi/v1/classes`](#list-classes) | List classes |
| `POST` | [`/codeapi/v1/methods`](#list-methods) | List methods |
| `POST` | [`/codeapi/v1/functions`](#list-functions) | List functions |
| `POST` | [`/codeapi/v1/classes/find`](#find-classes-by-pattern) | Find classes by pattern |
| `POST` | [`/codeapi/v1/methods/find`](#find-methods-by-pattern) | Find methods by pattern |
| `POST` | [`/codeapi/v1/class`](#get-class) | Get class details |
| `POST` | [`/codeapi/v1/method`](#get-method) | Get method details |
| `POST` | [`/codeapi/v1/class/methods`](#get-class-methods) | Get methods of a class |
| `POST` | [`/codeapi/v1/class/fields`](#get-class-fields) | Get fields of a class |
| `POST` | [`/codeapi/v1/callgraph`](#get-call-graph) | Get call graph |
| `POST` | [`/codeapi/v1/callers`](#get-callers) | Get callers of a function |
| `POST` | [`/codeapi/v1/callees`](#get-callees) | Get callees of a function |
| `POST` | [`/codeapi/v1/data/dependents`](#get-data-dependents) | Get data dependents |
| `POST` | [`/codeapi/v1/data/sources`](#get-data-sources) | Get data sources |
| `POST` | [`/codeapi/v1/impact`](#get-impact-analysis) | Impact analysis |
| `POST` | [`/codeapi/v1/inheritance`](#get-inheritance-tree) | Get inheritance tree |
| `POST` | [`/codeapi/v1/field/accessors`](#get-field-accessors) | Get field accessors |
| `POST` | [`/codeapi/v1/cypher`](#execute-cypher-query-read) | Execute read Cypher query |
| `POST` | [`/codeapi/v1/cypher/write`](#execute-cypher-query-write) | Execute write Cypher query |
| `GET` | [`/codeapi/v1/health`](#codeapi-health-check) | CodeAPI health check |

---

### Indexing & Search Endpoints

#### Health Check

```
GET /api/v1/health
```

**Response:**
```json
{
  "status": "healthy"
}
```

---

#### Build Index

Build code graph index for a repository.

```
POST /api/v1/buildIndex
```

**Request:**
```json
{
  "repo_name": "my-project",
  "use_head": false
}
```

**Response:**
```json
{
  "repo_name": "my-project",
  "status": "completed",
  "message": "Repository indexed successfully"
}
```

---

#### Index File

Index specific files through all processors.

```
POST /api/v1/indexFile
```

**Request:**
```json
{
  "repo_name": "my-project",
  "relative_paths": [
    "src/main.go",
    "src/utils/helper.go"
  ]
}
```

**Response:**
```json
{
  "repo_name": "my-project",
  "files": [
    {
      "relative_path": "src/main.go",
      "file_id": 123,
      "file_sha": "abc123...",
      "processors_run": ["CodeGraph", "Embedding"],
      "success": true
    }
  ],
  "message": "Processed 2 file(s): 2 succeeded, 0 failed"
}
```

---

#### Search Similar Code

Find semantically similar code using vector embeddings.

```
POST /api/v1/searchSimilarCode
```

**Request:**
```json
{
  "repo_name": "my-project",
  "code_snippet": "func calculateSum(a, b int) int {\n  return a + b\n}",
  "language": "go",
  "limit": 10,
  "include_code": true
}
```

**Response:**
```json
{
  "repo_name": "my-project",
  "collection_name": "my-project",
  "query": {
    "code_snippet": "func calculateSum...",
    "language": "go",
    "chunks_found": 1
  },
  "results": [
    {
      "chunk": {
        "file_path": "/path/to/math.go",
        "start_line": 10,
        "end_line": 15,
        "chunk_type": "function",
        "name": "add"
      },
      "score": 0.95,
      "code": "func add(x, y int) int {\n  return x + y\n}"
    }
  ],
  "success": true
}
```

---

#### Get Function Dependencies

Get call graph for a function.

```
POST /api/v1/functionDependencies
```

**Request:**
```json
{
  "repo_name": "my-project",
  "relative_path": "src/main.go",
  "function_name": "main",
  "depth": 2
}
```

**Response:**
```json
{
  "repo_name": "my-project",
  "file_path": "src/main.go",
  "function_name": "main",
  "dependencies": [
    {
      "name": "initialize",
      "call_locations": [...],
      "definition": {
        "name": "initialize",
        "location": {...}
      }
    }
  ]
}
```

---

#### Process Directory

Process a directory and generate embeddings.

```
POST /api/v1/processDirectory
```

**Request:**
```json
{
  "repo_name": "my-project",
  "collection_name": "my-project-embeddings"
}
```

**Response:**
```json
{
  "repo_name": "my-project",
  "collection_name": "my-project-embeddings",
  "total_chunks": 150,
  "success": true,
  "message": "Directory processed successfully"
}
```

---

### Code Analysis API Endpoints

#### List Repositories

```
GET /codeapi/v1/repos
```

**Response:**
```json
{
  "repos": ["my-project", "another-project"]
}
```

---

#### List Files

```
POST /codeapi/v1/files
```

**Request:**
```json
{
  "repo_name": "my-project",
  "limit": 100,
  "offset": 0
}
```

**Response:**
```json
{
  "files": [
    {
      "id": "file-123",
      "path": "src/main.go",
      "language": "go"
    }
  ],
  "total": 50
}
```

---

#### List Classes

```
POST /codeapi/v1/classes
```

**Request:**
```json
{
  "repo_name": "my-project",
  "limit": 50
}
```

**Response:**
```json
{
  "classes": [
    {
      "id": "class-456",
      "name": "UserService",
      "file_path": "src/service/user.go",
      "start_line": 10,
      "end_line": 100
    }
  ]
}
```

---

#### List Methods

```
POST /codeapi/v1/methods
```

**Request:**
```json
{
  "repo_name": "my-project",
  "class_id": "class-456"
}
```

**Response:**
```json
{
  "methods": [
    {
      "id": "method-123",
      "name": "GetUser",
      "signature": "func (s *UserService) GetUser(id string) (*User, error)",
      "start_line": 15,
      "end_line": 25
    }
  ]
}
```

---

#### List Functions

```
POST /codeapi/v1/functions
```

**Request:**
```json
{
  "repo_name": "my-project",
  "file_path": "src/main.go"
}
```

**Response:**
```json
{
  "functions": [
    {
      "id": "func-789",
      "name": "main",
      "signature": "func main()",
      "start_line": 5,
      "end_line": 20
    }
  ]
}
```

---

#### Find Classes by Pattern

```
POST /codeapi/v1/classes/find
```

**Request:**
```json
{
  "repo_name": "my-project",
  "pattern": ".*Service$"
}
```

**Response:**
```json
{
  "classes": [
    {
      "id": "class-456",
      "name": "UserService",
      "file_path": "src/service/user.go"
    },
    {
      "id": "class-457",
      "name": "OrderService",
      "file_path": "src/service/order.go"
    }
  ]
}
```

---

#### Find Methods by Pattern

```
POST /codeapi/v1/methods/find
```

**Request:**
```json
{
  "repo_name": "my-project",
  "pattern": "Get.*"
}
```

**Response:**
```json
{
  "methods": [
    {
      "id": "method-123",
      "name": "GetUser",
      "class_name": "UserService"
    },
    {
      "id": "method-124",
      "name": "GetOrder",
      "class_name": "OrderService"
    }
  ]
}
```

---

#### Get Class

```
POST /codeapi/v1/class
```

**Request:**
```json
{
  "repo_name": "my-project",
  "class_id": "class-456"
}
```

**Response:**
```json
{
  "class": {
    "id": "class-456",
    "name": "UserService",
    "file_path": "src/service/user.go",
    "start_line": 10,
    "end_line": 100,
    "methods": [...],
    "fields": [...]
  }
}
```

---

#### Get Method

```
POST /codeapi/v1/method
```

**Request:**
```json
{
  "repo_name": "my-project",
  "method_id": "method-123"
}
```

**Response:**
```json
{
  "method": {
    "id": "method-123",
    "name": "GetUser",
    "signature": "func (s *UserService) GetUser(id string) (*User, error)",
    "start_line": 15,
    "end_line": 25,
    "class_id": "class-456"
  }
}
```

---

#### Get Class Methods

```
POST /codeapi/v1/class/methods
```

**Request:**
```json
{
  "repo_name": "my-project",
  "class_id": "class-456"
}
```

**Response:**
```json
{
  "methods": [
    {
      "id": "method-123",
      "name": "GetUser",
      "signature": "func (s *UserService) GetUser(id string) (*User, error)"
    },
    {
      "id": "method-124",
      "name": "CreateUser",
      "signature": "func (s *UserService) CreateUser(user *User) error"
    }
  ]
}
```

---

#### Get Class Fields

```
POST /codeapi/v1/class/fields
```

**Request:**
```json
{
  "repo_name": "my-project",
  "class_id": "class-456"
}
```

**Response:**
```json
{
  "fields": [
    {
      "id": "field-001",
      "name": "db",
      "type": "*sql.DB"
    },
    {
      "id": "field-002",
      "name": "logger",
      "type": "*zap.Logger"
    }
  ]
}
```

---

#### Get Call Graph

```
POST /codeapi/v1/callgraph
```

**Request:**
```json
{
  "repo_name": "my-project",
  "function_id": "func-789",
  "direction": "both",
  "max_depth": 3
}
```

**Response:**
```json
{
  "nodes": [
    {
      "id": "func-789",
      "name": "processRequest",
      "type": "function"
    }
  ],
  "edges": [
    {
      "from": "func-789",
      "to": "func-790",
      "type": "CALLS"
    }
  ]
}
```

---

#### Get Callers

Get functions that call a specific function.

```
POST /codeapi/v1/callers
```

**Request:**
```json
{
  "repo_name": "my-project",
  "function_id": "func-789"
}
```

**Response:**
```json
{
  "callers": [
    {
      "id": "func-100",
      "name": "main",
      "file_path": "src/main.go"
    },
    {
      "id": "func-101",
      "name": "handleRequest",
      "file_path": "src/handler.go"
    }
  ]
}
```

---

#### Get Callees

Get functions called by a specific function.

```
POST /codeapi/v1/callees
```

**Request:**
```json
{
  "repo_name": "my-project",
  "function_id": "func-789"
}
```

**Response:**
```json
{
  "callees": [
    {
      "id": "func-200",
      "name": "validateInput",
      "file_path": "src/validation.go"
    },
    {
      "id": "func-201",
      "name": "saveToDatabase",
      "file_path": "src/db.go"
    }
  ]
}
```

---

#### Get Data Dependents

Get entities that depend on a variable or field's data.

```
POST /codeapi/v1/data/dependents
```

**Request:**
```json
{
  "repo_name": "my-project",
  "variable_id": "var-123"
}
```

**Response:**
```json
{
  "dependents": [
    {
      "id": "func-300",
      "name": "processData",
      "type": "function"
    }
  ]
}
```

---

#### Get Data Sources

Get the sources of data for a variable or field.

```
POST /codeapi/v1/data/sources
```

**Request:**
```json
{
  "repo_name": "my-project",
  "variable_id": "var-123"
}
```

**Response:**
```json
{
  "sources": [
    {
      "id": "func-400",
      "name": "fetchData",
      "type": "function"
    }
  ]
}
```

---

#### Get Impact Analysis

Analyze the impact of changes to a function.

```
POST /codeapi/v1/impact
```

**Request:**
```json
{
  "repo_name": "my-project",
  "function_id": "func-789",
  "max_depth": 5
}
```

**Response:**
```json
{
  "impacted": [
    {
      "id": "func-100",
      "name": "main",
      "depth": 1,
      "impact_type": "direct_caller"
    },
    {
      "id": "func-101",
      "name": "runServer",
      "depth": 2,
      "impact_type": "indirect_caller"
    }
  ],
  "total_impacted": 15
}
```

---

#### Get Inheritance Tree

```
POST /codeapi/v1/inheritance
```

**Request:**
```json
{
  "repo_name": "my-project",
  "class_id": "class-456"
}
```

**Response:**
```json
{
  "class": {
    "id": "class-456",
    "name": "UserService"
  },
  "parents": [
    {
      "id": "class-100",
      "name": "BaseService"
    }
  ],
  "children": [
    {
      "id": "class-500",
      "name": "AdminUserService"
    }
  ],
  "interfaces": [
    {
      "id": "interface-001",
      "name": "Service"
    }
  ]
}
```

---

#### Get Field Accessors

Get functions that read or write a specific field.

```
POST /codeapi/v1/field/accessors
```

**Request:**
```json
{
  "repo_name": "my-project",
  "field_id": "field-001"
}
```

**Response:**
```json
{
  "readers": [
    {
      "id": "func-500",
      "name": "GetConnection",
      "access_type": "read"
    }
  ],
  "writers": [
    {
      "id": "func-501",
      "name": "SetConnection",
      "access_type": "write"
    }
  ]
}
```

---

#### Execute Cypher Query (Read)

Execute custom read-only Cypher queries against the code graph.

```
POST /codeapi/v1/cypher
```

**Request:**
```json
{
  "repo_name": "my-project",
  "query": "MATCH (f:Function)-[:CALLS]->(g:Function) WHERE f.name = 'main' RETURN g.name"
}
```

**Response:**
```json
{
  "results": [
    {"g.name": "initialize"},
    {"g.name": "run"}
  ]
}
```

---

#### Execute Cypher Query (Write)

Execute write Cypher queries (use with caution).

```
POST /codeapi/v1/cypher/write
```

**Request:**
```json
{
  "repo_name": "my-project",
  "query": "MATCH (f:Function {name: 'deprecated'}) SET f.deprecated = true"
}
```

**Response:**
```json
{
  "success": true,
  "nodes_affected": 3
}
```

---

#### CodeAPI Health Check

```
GET /codeapi/v1/health
```

**Response:**
```json
{
  "status": "healthy"
}
```

---

## Docker

### Build Image

```bash
make docker-build
```

### Run Container

```bash
# Interactive mode
make docker-run

# Detached mode
make docker-run-detached

# With custom work directory
make docker-run-with-workdir WORKDIR=/path/to/code
```

### Docker Compose (Full Stack)

```bash
# Start all services (Neo4j, MySQL, Qdrant, CodeAPI)
make docker-compose-up

# View logs
make docker-compose-logs

# Stop all services
make docker-compose-down
```

## Supported Languages

| Language | Parser | LSP Server | Key Features |
|----------|--------|------------|--------------|
| Go | tree-sitter-go | gopls | Generics, interfaces, goroutines, channels |
| Python | tree-sitter-python | pyright | Async/await, decorators, type hints, comprehensions |
| Java | tree-sitter-java | Eclipse JDT.LS | Records, sealed classes, pattern matching, lambdas, annotations |
| TypeScript | tree-sitter-typescript | typescript-language-server | Generics, advanced types, decorators |
| JavaScript | tree-sitter-javascript | typescript-language-server | ES6+, async/await, classes, arrow functions |
| C# | tree-sitter-c-sharp | csharp-ls | Classes, interfaces, LINQ, async/await |

### Java Support

Java support includes full LSP integration via [Eclipse JDT Language Server](https://github.com/eclipse-jdtls/eclipse.jdt.ls) for semantic analysis (call hierarchies, symbol resolution) combined with tree-sitter for fast syntax parsing.

**Chained Method Calls:**

CodeAPI correctly parses and tracks chained method calls common in Java (fluent APIs, Stream API, repository pattern):

```java
// All method calls in the chain are captured:
return repository.findByVetId(id)   // FunctionCall → findByVetId
    .stream()                        // FunctionCall → stream
    .map(this::toDto)                // FunctionCall → map
    .collect(Collectors.toList());   // FunctionCall → collect
```

This enables accurate CALLS_FUNCTION relationship tracking for Spring Data repositories, Stream operations, and builder patterns.

**Setup:**

Eclipse JDT.LS is bundled in the `assets/` folder as a tar.gz archive. Extract it before first use:

```bash
cd assets
tar -xzf jdt-language-server-*.tar.gz
```

This creates the `bin/jdtls` launcher and `plugins/` directory needed by the language server.

**Configuration:**
```yaml
language_servers:
  java: "${CODEAPI_ROOT}/scripts/javalsp.sh"
```

**External Module Detection:**
Java LSP client automatically detects external dependencies from:
- Maven local repository (`.m2/repository/`)
- Gradle cache (`.gradle/caches/`)
- Build output directories (`target/`, `build/`, `out/`)
- JDK/JRE locations

### Java Annotations in Metadata

Java annotations are automatically extracted and stored in the `metadata` field of Class and Function nodes. This enables querying code by framework-specific annotations (e.g., Spring Boot `@RestController`, `@GetMapping`).

**API Access:**

Metadata is exposed in all Code Graph API responses (`/codeapi/v1/classes`, `/codeapi/v1/methods`, etc.). See [API.md](API.md) for complete documentation.

**How Annotations Are Captured:**

Annotations are extracted from classes, interfaces, records, enums, methods, and constructors. Each annotation is serialized as a JSON string containing:
- `name`: The annotation name (e.g., `"RestController"`, `"GetMapping"`)
- `arguments`: Optional key-value pairs for annotation arguments

**Example: Spring Boot Controller**

For a class like:
```java
@RestController
@RequestMapping("/api/owners")
public class OwnerController {

    @GetMapping("/{id}")
    public Owner getOwner(@PathVariable Long id) { ... }
}
```

The metadata stored in Neo4j:
```json
// Class node metadata
{
  "annotations": [
    "{\"name\":\"RestController\"}",
    "{\"name\":\"RequestMapping\",\"arguments\":{\"value\":\"/api/owners\"}}"
  ]
}

// Method node metadata
{
  "annotations": [
    "{\"name\":\"GetMapping\",\"arguments\":{\"value\":\"/{id}\"}}"
  ]
}
```

**Querying Annotations via Cypher:**

Find all REST controllers:
```cypher
MATCH (c:Class)
WHERE ANY(a IN c.annotations WHERE a CONTAINS '"name":"RestController"')
RETURN c.name, c.file_path
```

Find all methods with `@GetMapping`:
```cypher
MATCH (f:Function)
WHERE ANY(a IN f.annotations WHERE a CONTAINS '"name":"GetMapping"')
RETURN f.name, f.annotations
```

**Supported Annotation Types:**
- Marker annotations: `@Override`, `@Repository`, `@Service`
- Single-value annotations: `@GetMapping("/path")`, `@Query("SELECT ...")`
- Multi-value annotations: `@Size(min = 1, max = 50)`, `@Column(name = "id", nullable = false)`

## Project Structure

```
codeapi/
├── cmd/                    # Entry points
│   └── main.go            # Main server and CLI
├── internal/
│   ├── handler/           # HTTP route handlers
│   ├── controller/        # Business logic
│   ├── service/           # Domain services
│   │   ├── codegraph/    # Neo4j code graph
│   │   └── vector/       # Vector DB & embeddings
│   ├── codeapi/          # CodeAPI facade
│   ├── parse/            # Language parsers
│   ├── model/            # Data models
│   ├── config/           # Configuration
│   ├── db/               # Database layer
│   └── util/             # Utilities
├── pkg/
│   └── lsp/              # LSP integration
├── config/               # Configuration files
├── tests/                # Test repositories
├── Makefile
├── Dockerfile
└── docker-compose.yml
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

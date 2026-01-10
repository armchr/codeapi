# CodeAPI - REST API Documentation

This document describes all available REST API endpoints for CodeAPI.

## Base URLs

- **Repository Operations**: `/api/v1`
- **Code Graph API**: `/codeapi/v1`

---

## Health Check Endpoints

### GET /api/v1/health

Check the health of the main API service.

**Response:**
```json
{
  "status": "healthy"
}
```

### GET /codeapi/v1/health

Check the health of the CodeAPI service.

**Response:**
```json
{
  "status": "healthy"
}
```

---

## Repository Operations (`/api/v1`)

### POST /api/v1/buildIndex

Build the code graph index for a repository.

**Request:**
```json
{
  "repo_name": "my-repo",
  "use_head": false
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository (must exist in source.yaml) |
| `use_head` | boolean | No | Use git HEAD version instead of working directory |

**Response:**
```json
{
  "repo_name": "my-repo",
  "status": "completed",
  "message": "Repository indexed successfully"
}
```

---

### POST /api/v1/indexFile

Index specific files through all registered processors.

**Request:**
```json
{
  "repo_name": "my-repo",
  "relative_paths": [
    "src/main/java/com/example/Service.java",
    "src/main/java/com/example/Controller.java"
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `relative_paths` | string[] | Yes | List of file paths relative to repository root |

**Response:**
```json
{
  "repo_name": "my-repo",
  "files": [
    {
      "relative_path": "src/main/java/com/example/Service.java",
      "file_id": 123,
      "file_sha": "abc123...",
      "processors_run": ["codegraph", "embeddings"],
      "success": true
    },
    {
      "relative_path": "src/main/java/com/example/Controller.java",
      "file_id": 124,
      "file_sha": "def456...",
      "processors_run": ["codegraph", "embeddings"],
      "success": true
    }
  ],
  "message": "Processed 2 file(s): 2 succeeded, 0 failed"
}
```

---

### POST /api/v1/functionDependencies

Get dependencies for a specific function.

**Request:**
```json
{
  "repo_name": "my-repo",
  "relative_path": "src/main/java/com/example/Service.java",
  "function_name": "processOrder",
  "depth": 2
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `relative_path` | string | Yes | File path relative to repository root |
| `function_name` | string | Yes | Name of the function |
| `depth` | int | No | Depth of dependency traversal (default: 2) |

---

### POST /api/v1/processDirectory

Process a directory for code chunking and embeddings.

**Request:**
```json
{
  "repo_name": "my-repo",
  "collection_name": "my-collection"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `collection_name` | string | No | Qdrant collection name (defaults to repo_name) |

**Response:**
```json
{
  "repo_name": "my-repo",
  "collection_name": "my-collection",
  "total_chunks": 150,
  "success": true,
  "message": "Directory processed successfully"
}
```

---

### POST /api/v1/searchSimilarCode

Search for similar code using a code snippet.

**Request:**
```json
{
  "repo_name": "my-repo",
  "code_snippet": "public void processOrder(Order order) {\n  // ...\n}",
  "language": "java",
  "collection_name": "my-collection",
  "limit": 10,
  "include_code": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `code_snippet` | string | Yes | Code snippet to search for |
| `language` | string | Yes | Language: `go`, `python`, `java`, `javascript`, `typescript` |
| `collection_name` | string | No | Qdrant collection name (defaults to repo_name) |
| `limit` | int | No | Maximum results to return (default: 10) |
| `include_code` | boolean | No | Include source code in results |

**Response:**
```json
{
  "repo_name": "my-repo",
  "collection_name": "my-collection",
  "query": {
    "code_snippet": "...",
    "language": "java",
    "chunks_found": 3,
    "chunks": [...]
  },
  "results": [
    {
      "chunk": {
        "file_path": "/path/to/file.java",
        "start_line": 10,
        "end_line": 25,
        "chunk_type": "method"
      },
      "score": 0.95,
      "query_chunk_index": 0,
      "code": "public void processOrder(Order order) { ... }"
    }
  ],
  "success": true,
  "message": "Search completed successfully"
}
```

---

### POST /api/v1/searchMethodsBySignature

Search for methods using natural language queries on their signatures. This endpoint enables semantic search on method signatures, allowing you to find methods by describing what they do (e.g., "find user by email") rather than requiring exact name matches.

**How it works:**
1. During indexing, method signatures are normalized into embedding-friendly text (e.g., `findByEmail(String email): User` becomes `"User Service find By Email String email returns User"`)
2. The query is converted to an embedding and compared against stored signature embeddings
3. Results are ranked by semantic similarity

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "query": "find user by email address",
  "limit": 10
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `query` | string | Yes | Natural language query describing the method signature |
| `limit` | int | No | Maximum results to return (default: 10) |

**Response:**
```json
{
  "repo_name": "spring-petclinic",
  "query": "find user by email address",
  "results": [
    {
      "method_name": "findByEmail",
      "class_name": "UserService",
      "signature": "User findByEmail(String email)",
      "return_type": "User",
      "parameter_types": ["String"],
      "parameter_names": ["email"],
      "file_path": "src/main/java/org/example/UserService.java",
      "start_line": 45,
      "end_line": 55,
      "score": 0.92,
      "normalized_text": "User Service find By Email String email returns User"
    },
    {
      "method_name": "findUserByEmailAddress",
      "class_name": "UserRepository",
      "signature": "Optional<User> findUserByEmailAddress(String emailAddress)",
      "return_type": "Optional<User>",
      "parameter_types": ["String"],
      "parameter_names": ["emailAddress"],
      "file_path": "src/main/java/org/example/UserRepository.java",
      "start_line": 22,
      "end_line": 30,
      "score": 0.87,
      "normalized_text": "User Repository find User By Email Address String email Address returns Optional User"
    }
  ],
  "success": true,
  "message": "Found 2 matching methods"
}
```

**Example Queries:**
- `"authenticate user with password"` - finds authentication methods
- `"save order to database"` - finds order persistence methods
- `"calculate total price"` - finds pricing calculation methods
- `"get products by category"` - finds product filtering methods

**Note:** Method signatures are indexed automatically during the normal indexing process (`/api/v1/buildIndex` or `--build-index` CLI). No additional configuration is required.

---

## Code Graph API (`/codeapi/v1`)

### Reader Endpoints

These endpoints query the code graph for entities (files, classes, methods, fields).

---

#### GET /codeapi/v1/repos

List all indexed repositories.

**Response:**
```json
{
  "repos": ["spring-petclinic", "my-service", "utils-lib"]
}
```

---

#### POST /codeapi/v1/files

List files in a repository.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "limit": 20,
  "offset": 0
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `limit` | int | No | Maximum results to return |
| `offset` | int | No | Number of results to skip |

**Response:**
```json
{
  "files": [
    {
      "id": 12345,
      "path": "src/main/java/org/example/Service.java",
      "language": "java",
      "file_id": 1,
      "repo_name": "spring-petclinic"
    }
  ]
}
```

---

#### POST /codeapi/v1/classes

List all classes in a repository.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "limit": 20,
  "offset": 0
}
```

**Response:**
```json
{
  "classes": [
    {
      "id": 67890,
      "name": "OwnerController",
      "file_path": "src/main/java/org/example/OwnerController.java",
      "file_id": 1,
      "range": {
        "start": {"line": 15, "character": 0},
        "end": {"line": 120, "character": 1}
      },
      "metadata": {
        "annotations": [
          "{\"name\":\"Controller\"}",
          "{\"name\":\"RequestMapping\",\"arguments\":[\"/owners\"]}"
        ],
        "is_interface": false
      }
    }
  ]
}
```

---

#### POST /codeapi/v1/classes/find

Find classes matching specific criteria.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "name": "OwnerController",
  "name_like": "Controller",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "file_id": 1,
  "limit": 10,
  "offset": 0
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `name` | string | No | Exact class name match |
| `name_like` | string | No | Pattern match (contains) |
| `file_path` | string | No | Exact file path |
| `file_id` | int | No | File ID |
| `limit` | int | No | Maximum results |
| `offset` | int | No | Results to skip |

**Response:**
```json
{
  "classes": [...]
}
```

---

#### POST /codeapi/v1/methods

List all methods (including class methods) in a repository.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "limit": 20,
  "offset": 0
}
```

**Response:**
```json
{
  "methods": [
    {
      "id": 11111,
      "name": "findOwner",
      "file_path": "src/main/java/org/example/OwnerController.java",
      "file_id": 1,
      "range": {
        "start": {"line": 45, "character": 4},
        "end": {"line": 55, "character": 5}
      },
      "metadata": {
        "annotations": [
          "{\"name\":\"GetMapping\",\"arguments\":[\"/owners/{ownerId}\"]}"
        ]
      }
    }
  ]
}
```

---

#### POST /codeapi/v1/functions

List top-level functions (not class methods) in a repository.

**Request:**
```json
{
  "repo_name": "my-go-project",
  "limit": 20,
  "offset": 0
}
```

**Note:** For languages like Java where all functions are class methods, this endpoint returns an empty array.

**Response:**
```json
{
  "functions": [
    {
      "id": 22222,
      "name": "main",
      "file_path": "cmd/main.go",
      "file_id": 5,
      "range": {...}
    }
  ]
}
```

---

#### POST /codeapi/v1/methods/find

Find methods matching specific criteria.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "name": "findOwner",
  "name_like": "find",
  "class_name": "OwnerController",
  "class_id": 67890,
  "file_path": "src/main/java/org/example/OwnerController.java",
  "file_id": 1,
  "limit": 10,
  "offset": 0
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `name` | string | No | Exact method name match |
| `name_like` | string | No | Pattern match (contains) |
| `class_name` | string | No | Filter by class name |
| `class_id` | int | No | Filter by class ID |
| `file_path` | string | No | Filter by file path |
| `file_id` | int | No | Filter by file ID |
| `limit` | int | No | Maximum results |
| `offset` | int | No | Results to skip |

**Response:**
```json
{
  "methods": [...]
}
```

---

#### POST /codeapi/v1/class

Get a specific class by ID.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "class_id": 67890,
  "include_methods": true,
  "include_fields": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `class_id` | int | Yes | Class node ID |
| `include_methods` | boolean | No | Include class methods in response |
| `include_fields` | boolean | No | Include class fields in response |

**Response:**
```json
{
  "class": {
    "id": 67890,
    "name": "OwnerController",
    "file_path": "src/main/java/org/example/OwnerController.java",
    "file_id": 1,
    "range": {...},
    "methods": [...],
    "fields": [...]
  }
}
```

---

#### POST /codeapi/v1/method

Get a specific method by ID.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "method_id": 11111
}
```

**Response:**
```json
{
  "method": {
    "id": 11111,
    "name": "findOwner",
    "file_path": "src/main/java/org/example/OwnerController.java",
    "file_id": 1,
    "range": {...}
  }
}
```

---

#### POST /codeapi/v1/class/methods

Get all methods belonging to a class.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "class_id": 67890
}
```

**Response:**
```json
{
  "methods": [
    {
      "id": 11111,
      "name": "findOwner",
      ...
    },
    {
      "id": 11112,
      "name": "createOwner",
      ...
    }
  ]
}
```

---

#### POST /codeapi/v1/class/fields

Get all fields belonging to a class.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "class_id": 67890
}
```

**Response:**
```json
{
  "fields": [
    {
      "id": 33333,
      "name": "ownerRepository",
      "type": "OwnerRepository",
      "range": {...}
    }
  ]
}
```

---

### Analyzer Endpoints

These endpoints perform graph traversals and analysis.

---

#### POST /codeapi/v1/callgraph

Get the call graph for a function.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "function_id": 11111,
  "function_name": "findOwner",
  "class_name": "OwnerController",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "direction": "outgoing",
  "max_depth": 3,
  "include_external": false
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `function_id` | int | No* | Function node ID |
| `function_name` | string | No* | Function name (requires file_path or class_name) |
| `class_name` | string | No | Class containing the function |
| `file_path` | string | No | File containing the function |
| `direction` | string | No | `outgoing` (callees), `incoming` (callers), or `both` (default: `outgoing`) |
| `max_depth` | int | No | Maximum traversal depth (default: 3) |
| `include_external` | boolean | No | Include external package calls |

*Either `function_id` or `function_name` is required.

**Response:**
```json
{
  "call_graph": {
    "root": {
      "id": 11111,
      "name": "findOwner",
      "class_name": "OwnerController",
      "file_path": "...",
      "depth": 0
    },
    "nodes": {
      "11111": {...},
      "44444": {...}
    },
    "edges": [
      {
        "caller_id": 11111,
        "callee_id": 44444,
        "call_site": {
          "file_path": "...",
          "range": {...}
        }
      }
    ],
    "direction": "outgoing",
    "max_depth": 3,
    "truncated": false
  }
}
```

---

#### POST /codeapi/v1/callers

Get functions that call a specific function (incoming call graph). The `function_id` field is flexible and accepts:
- A numeric ID: `11111`
- A qualified name: `"ClassName.methodName"`
- A simple function name: `"main"`

**Request (by numeric ID):**
```json
{
  "repo_name": "spring-petclinic",
  "function_id": 11111,
  "max_depth": 3
}
```

**Request (by qualified name):**
```json
{
  "repo_name": "spring-petclinic",
  "function_id": "PetRepository.findByOwnerId",
  "max_depth": 3
}
```

**Request (using separate fields):**
```json
{
  "repo_name": "spring-petclinic",
  "function_name": "findByOwnerId",
  "class_name": "PetRepository",
  "max_depth": 3
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `function_id` | int64/string | No* | Numeric ID or qualified name (e.g., `"Class.method"`) |
| `function_name` | string | No* | Function/method name |
| `class_name` | string | No | Class name (for methods) |
| `file_path` | string | No | File path to narrow search |
| `max_depth` | int | No | Maximum traversal depth (default: 3) |
| `include_external` | bool | No | Include external function calls |

*Either `function_id` or `function_name` is required.

**Response:**
```json
{
  "call_graph": {...}
}
```

---

#### POST /codeapi/v1/callees

Get functions called by a specific function (outgoing call graph). The `function_id` field is flexible and accepts:
- A numeric ID: `11111`
- A qualified name: `"ClassName.methodName"`
- A simple function name: `"main"`

**Request (by numeric ID):**
```json
{
  "repo_name": "spring-petclinic",
  "function_id": 11111,
  "max_depth": 3
}
```

**Request (by qualified name):**
```json
{
  "repo_name": "spring-petclinic",
  "function_id": "OwnerController.getOwnerPets",
  "max_depth": 3
}
```

**Request (using separate fields):**
```json
{
  "repo_name": "spring-petclinic",
  "function_name": "getOwnerPets",
  "class_name": "OwnerController",
  "max_depth": 3
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `function_id` | int64/string | No* | Numeric ID or qualified name (e.g., `"Class.method"`) |
| `function_name` | string | No* | Function/method name |
| `class_name` | string | No | Class name (for methods) |
| `file_path` | string | No | File path to narrow search |
| `max_depth` | int | No | Maximum traversal depth (default: 3) |
| `include_external` | bool | No | Include external function calls |

*Either `function_id` or `function_name` is required.

**Response:**
```json
{
  "call_graph": {...}
}
```

---

#### POST /codeapi/v1/data/dependents

Get nodes that depend on a value (data flow analysis).

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "node_id": 55555,
  "variable_name": "owner",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "max_depth": 5,
  "include_indirect": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `node_id` | int | No* | Node ID |
| `variable_name` | string | No* | Variable name (requires file_path) |
| `file_path` | string | No | File containing the variable |
| `max_depth` | int | No | Maximum traversal depth |
| `include_indirect` | boolean | No | Include transitive dependencies |

*Either `node_id` or `variable_name` is required.

**Response:**
```json
{
  "dependency_graph": {
    "root": {...},
    "nodes": {...},
    "edges": [
      {
        "source_id": 55555,
        "target_id": 66666,
        "flow_type": "assignment"
      }
    ],
    "direction": "outgoing",
    "truncated": false
  }
}
```

---

#### POST /codeapi/v1/data/sources

Get nodes that contribute to a value (backward data flow).

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "node_id": 55555,
  "max_depth": 5,
  "include_indirect": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `node_id` | int | Yes | Node ID |
| `max_depth` | int | No | Maximum traversal depth |
| `include_indirect` | boolean | No | Include transitive sources |

**Response:**
```json
{
  "dependency_graph": {...}
}
```

---

#### POST /codeapi/v1/impact

Perform impact analysis for a code element.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "node_id": 11111,
  "name": "findOwner",
  "node_type": "function",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "max_depth": 3,
  "include_call_graph": true,
  "include_data_flow": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `node_id` | int | No* | Node ID |
| `name` | string | No* | Element name (requires file_path and node_type) |
| `node_type` | string | No | Type: `function`, `class`, `field`, `variable` |
| `file_path` | string | No | File containing the element |
| `max_depth` | int | No | Maximum traversal depth (default: 3) |
| `include_call_graph` | boolean | No | Include call graph in analysis |
| `include_data_flow` | boolean | No | Include data flow in analysis |

*Either `node_id` or `name` is required.

**Response:**
```json
{
  "impact": {...}
}
```

---

#### POST /codeapi/v1/inheritance

Get the inheritance hierarchy for a class.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "class_id": 67890
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `class_id` | int | Yes | Class node ID |

**Response:**
```json
{
  "inheritance_tree": {
    "root": {
      "id": 67890,
      "name": "OwnerController",
      "file_path": "...",
      "parents": [...],
      "children": [...],
      "depth": 0
    },
    "nodes": {...},
    "max_depth": 3
  }
}
```

---

#### POST /codeapi/v1/field/accessors

Get methods that access a specific field.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "field_id": 33333,
  "class_name": "OwnerController",
  "field_name": "ownerRepository"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `field_id` | int | No* | Field node ID |
| `class_name` | string | No* | Class containing the field |
| `field_name` | string | No* | Field name |

*Either `field_id` or (`class_name` and `field_name`) is required.

**Response:**
```json
{
  "field_accessors": {...}
}
```

---

### Code Summary Endpoints

These endpoints query LLM-generated summaries for code entities.

**On-Demand Generation:** If a summary doesn't exist when requested via `/summaries/entity`, `/summaries/file`, or `/summaries/file/summary`, the system will automatically generate it using the configured LLM (if summary processor is enabled). This may take a few seconds for the first request.

---

#### POST /codeapi/v1/summaries/file

Get all summaries for entities in a file, optionally filtered by type.

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "entity_type": "function"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `file_path` | string | Yes | Path relative to repository root |
| `entity_type` | string | No | Filter by type: `function`, `class`, `file`, `folder`, `project` |

**Response:**
```json
{
  "file_path": "src/main/java/org/example/OwnerController.java",
  "summaries": [
    {
      "id": 1,
      "entity_id": "12884901895",
      "entity_type": "function",
      "entity_name": "findOwner",
      "file_path": "src/main/java/org/example/OwnerController.java",
      "summary": "ONE_LINE: Retrieves an owner by ID...",
      "context_hash": "abc123...",
      "llm_provider": "ollama",
      "llm_model": "qwen3:4b",
      "prompt_tokens": 150,
      "output_tokens": 200,
      "created_at": "2026-01-08T10:00:00Z",
      "updated_at": "2026-01-08T10:00:00Z"
    }
  ],
  "count": 1
}
```

---

#### POST /codeapi/v1/summaries/file/summary

Get the file-level summary for a specific file. **Supports on-demand generation.**

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "file_path": "src/main/java/org/example/OwnerController.java"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `file_path` | string | Yes | Path relative to repository root |

**Response:**
```json
{
  "id": 5,
  "entity_id": "src/main/java/org/example/OwnerController.java",
  "entity_type": "file",
  "entity_name": "OwnerController.java",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "summary": "This file defines the OwnerController class...",
  "context_hash": "def456...",
  "llm_provider": "ollama",
  "llm_model": "qwen3:4b",
  "prompt_tokens": 200,
  "output_tokens": 150,
  "created_at": "2026-01-08T10:00:00Z",
  "updated_at": "2026-01-08T10:00:00Z"
}
```

---

#### POST /codeapi/v1/summaries/entity

Get a specific function or class summary by name. **Supports on-demand generation.**

**Request:**
```json
{
  "repo_name": "spring-petclinic",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "entity_type": "function",
  "entity_name": "findOwner"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |
| `file_path` | string | Yes | Path relative to repository root |
| `entity_type` | string | Yes | Must be `function` or `class` |
| `entity_name` | string | Yes | Name of the function or class |

**Response:**
```json
{
  "id": 1,
  "entity_id": "12884901895",
  "entity_type": "function",
  "entity_name": "findOwner",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "summary": "ONE_LINE: Retrieves an owner by ID...",
  "context_hash": "abc123...",
  "llm_provider": "ollama",
  "llm_model": "qwen3:4b",
  "prompt_tokens": 150,
  "output_tokens": 200,
  "created_at": "2026-01-08T10:00:00Z",
  "updated_at": "2026-01-08T10:00:00Z"
}
```

---

#### POST /codeapi/v1/summaries/stats

Get summary statistics for a repository.

**Request:**
```json
{
  "repo_name": "spring-petclinic"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_name` | string | Yes | Name of the repository |

**Response:**
```json
{
  "repo_name": "spring-petclinic",
  "stats": {
    "total_summaries": 150,
    "by_type": {
      "function": 100,
      "class": 25,
      "file": 20,
      "folder": 4,
      "project": 1
    },
    "total_prompt_tokens": 15000,
    "total_output_tokens": 20000
  }
}
```

---

### Raw Cypher Endpoints

These endpoints allow executing raw Neo4j Cypher queries.

---

#### POST /codeapi/v1/cypher

Execute a read-only Cypher query.

**Request:**
```json
{
  "query": "MATCH (c:Class {name: $name}) RETURN c",
  "params": {
    "name": "OwnerController"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | string | Yes | Cypher query (read-only) |
| `params` | object | No | Query parameters |

**Response:**
```json
{
  "results": [
    {
      "c": {
        "id": 67890,
        "name": "OwnerController",
        ...
      }
    }
  ]
}
```

---

#### POST /codeapi/v1/cypher/write

Execute a write Cypher query.

**Request:**
```json
{
  "query": "MATCH (c:Class {id: $id}) SET c.custom_field = $value RETURN c",
  "params": {
    "id": 67890,
    "value": "custom_value"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | string | Yes | Cypher query (write) |
| `params` | object | No | Query parameters |

**Response:**
```json
{
  "results": [...]
}
```

---

## Error Responses

All endpoints return errors in the following format:

```json
{
  "error": "Error message",
  "details": "Additional error details (optional)"
}
```

### HTTP Status Codes

| Status Code | Description |
|-------------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid parameters |
| 404 | Not Found - Repository or entity not found |
| 500 | Internal Server Error |
| 503 | Service Unavailable - Required service not available |

---

## Common Types

### Range

Represents a position range in source code:

```json
{
  "start": {
    "line": 10,
    "character": 0
  },
  "end": {
    "line": 25,
    "character": 1
  }
}
```

### Node IDs

All entities have unique `id` fields (int64) that can be used to reference them in subsequent API calls.

### File IDs

The `file_id` field is a repository-scoped identifier for files, separate from the global node `id`.

### Metadata

All entity types (classes, methods, fields, files) include an optional `metadata` field containing additional attributes extracted during parsing. This field is only present when metadata exists for the entity.

**Common metadata fields:**

| Field | Description | Applicable To |
|-------|-------------|---------------|
| `annotations` | Array of JSON-encoded annotation objects | Classes, Methods (Java) |
| `is_interface` | Boolean indicating if the class is an interface | Classes (Java) |
| `is_record` | Boolean indicating if the class is a record | Classes (Java) |
| `is_enum` | Boolean indicating if the class is an enum | Classes (Java) |
| `is_constructor` | Boolean indicating if the method is a constructor | Methods (Java) |

**Annotation format:**

Annotations are stored as JSON strings with the following structure:

```json
{
  "name": "GetMapping",
  "arguments": ["/owners/{ownerId}"]
}
```

For annotations without arguments:
```json
{
  "name": "Override"
}
```

**Example with metadata:**

```json
{
  "id": 12345,
  "name": "OwnerController",
  "file_path": "src/main/java/org/example/OwnerController.java",
  "file_id": 1,
  "range": {...},
  "metadata": {
    "annotations": [
      "{\"name\":\"RestController\"}",
      "{\"name\":\"RequestMapping\",\"arguments\":[\"/api/owners\"]}"
    ]
  }
}
```

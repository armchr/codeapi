#!/bin/bash
# API Test Commands for CodeAPI
# Usage: ./scripts/test_api.sh [command]
# Or run individual curl commands directly

BASE_URL="${CODEAPI_URL:-http://localhost:8181}"
REPO_NAME="${CODEAPI_REPO:-spring-petclinic}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
NC='\033[0m'

# Debug mode (set DEBUG=1 to enable)
DEBUG="${DEBUG:-1}"

print_header() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Debug logging helper for API calls
# Usage: api_call "METHOD" "endpoint" ["request_body"]
api_call() {
    local method="$1"
    local endpoint="$2"
    local body="$3"
    local url="${BASE_URL}${endpoint}"

    if [[ "$DEBUG" == "1" ]]; then
        echo -e "${CYAN}┌─ REQUEST ─────────────────────────────────────────────${NC}"
        echo -e "${CYAN}│${NC} ${GREEN}${method}${NC} ${url}"
        if [[ -n "$body" ]]; then
            echo -e "${CYAN}│${NC} ${GRAY}Content-Type: application/json${NC}"
            echo -e "${CYAN}│${NC}"
            echo -e "${CYAN}│ Body:${NC}"
            echo "$body" | jq . 2>/dev/null | sed 's/^/│ /'
        fi
        echo -e "${CYAN}└────────────────────────────────────────────────────────${NC}"
    fi

    # Make the request and capture response with timing
    local start_time=$(date +%s%3N)
    local response
    local http_code

    if [[ "$method" == "GET" ]]; then
        response=$(curl -s -w "\n%{http_code}" "$url")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$body")
    fi

    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))

    # Split response body and http code
    http_code=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')

    if [[ "$DEBUG" == "1" ]]; then
        echo -e "${CYAN}┌─ RESPONSE ────────────────────────────────────────────${NC}"

        # Color code based on HTTP status
        if [[ "$http_code" =~ ^2 ]]; then
            echo -e "${CYAN}│${NC} Status: ${GREEN}${http_code}${NC}  Duration: ${duration}ms"
        elif [[ "$http_code" =~ ^4 ]]; then
            echo -e "${CYAN}│${NC} Status: ${YELLOW}${http_code}${NC}  Duration: ${duration}ms"
        else
            echo -e "${CYAN}│${NC} Status: ${RED}${http_code}${NC}  Duration: ${duration}ms"
        fi

        echo -e "${CYAN}│${NC}"
        echo -e "${CYAN}│ Body:${NC}"
        echo "$response" | jq . 2>/dev/null | sed 's/^/│ /' || echo "$response" | sed 's/^/│ /'
        echo -e "${CYAN}└────────────────────────────────────────────────────────${NC}"
    else
        # Non-debug mode: just output the response
        echo "$response" | jq . 2>/dev/null || echo "$response"
    fi
}

# =============================================================================
# HEALTH CHECKS
# =============================================================================

health_check() {
    print_header "Health Check - /api/v1/health"
    api_call "GET" "/api/v1/health"
}

codeapi_health() {
    print_header "CodeAPI Health - /codeapi/v1/health"
    api_call "GET" "/codeapi/v1/health"
}

# =============================================================================
# INDEXING & SEARCH API (/api/v1/)
# =============================================================================

build_index() {
    print_header "Build Index - /api/v1/buildIndex"
    api_call "POST" "/api/v1/buildIndex" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"use_head\": false
    }"
}

index_file() {
    print_header "Index File - /api/v1/indexFile"
    api_call "POST" "/api/v1/indexFile" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"relative_paths\": [
            \"src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java\"
        ]
    }"
}

search_similar_code() {
    print_header "Search Similar Code - /api/v1/searchSimilarCode"
    api_call "POST" "/api/v1/searchSimilarCode" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"code_snippet\": \"public List<Owner> findAll() { return ownerRepository.findAll(); }\",
        \"language\": \"java\",
        \"limit\": 5,
        \"include_code\": true
    }"
}

function_dependencies() {
    print_header "Function Dependencies - /api/v1/functionDependencies"
    api_call "POST" "/api/v1/functionDependencies" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"relative_path\": \"src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java\",
        \"function_name\": \"showOwner\",
        \"depth\": 2
    }"
}

process_directory() {
    print_header "Process Directory - /api/v1/processDirectory"
    api_call "POST" "/api/v1/processDirectory" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"collection_name\": \"${REPO_NAME}-embeddings\"
    }"
}

# =============================================================================
# CODE ANALYSIS API (/codeapi/v1/)
# =============================================================================

list_repos() {
    print_header "List Repositories - /codeapi/v1/repos"
    api_call "GET" "/codeapi/v1/repos"
}

list_files() {
    print_header "List Files - /codeapi/v1/files"
    api_call "POST" "/codeapi/v1/files" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"limit\": 20,
        \"offset\": 0
    }"
}

list_classes() {
    print_header "List Classes - /codeapi/v1/classes"
    api_call "POST" "/codeapi/v1/classes" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"limit\": 20
    }"
}

list_methods() {
    print_header "List Methods - /codeapi/v1/methods"
    api_call "POST" "/codeapi/v1/methods" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"limit\": 20
    }"
}

list_functions() {
    print_header "List Functions - /codeapi/v1/functions"
    api_call "POST" "/codeapi/v1/functions" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"limit\": 20
    }"
}

find_classes() {
    print_header "Find Classes by Pattern - /codeapi/v1/classes/find"
    api_call "POST" "/codeapi/v1/classes/find" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"pattern\": \".*Controller$\"
    }"
}

find_methods() {
    print_header "Find Methods by Pattern - /codeapi/v1/methods/find"
    api_call "POST" "/codeapi/v1/methods/find" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"pattern\": \"^find.*\"
    }"
}

get_class() {
    print_header "Get Class Details - /codeapi/v1/class"
    api_call "POST" "/codeapi/v1/class" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"class_name\": \"OwnerController\"
    }"
}

get_class_methods() {
    print_header "Get Class Methods - /codeapi/v1/class/methods"
    api_call "POST" "/codeapi/v1/class/methods" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"class_name\": \"OwnerController\"
    }"
}

get_class_fields() {
    print_header "Get Class Fields - /codeapi/v1/class/fields"
    api_call "POST" "/codeapi/v1/class/fields" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"class_name\": \"Owner\"
    }"
}

# =============================================================================
# CALL GRAPH API
# =============================================================================

get_callgraph() {
    print_header "Get Call Graph - /codeapi/v1/callgraph"
    api_call "POST" "/codeapi/v1/callgraph" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"function_name\": \"showOwner\",
        \"direction\": \"both\",
        \"max_depth\": 3
    }"
}

get_callers() {
    print_header "Get Callers - /codeapi/v1/callers"
    api_call "POST" "/codeapi/v1/callers" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"function_name\": \"findById\"
    }"
}

get_callees() {
    print_header "Get Callees - /codeapi/v1/callees"
    api_call "POST" "/codeapi/v1/callees" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"function_name\": \"showOwner\"
    }"
}

# =============================================================================
# DATA FLOW API
# =============================================================================

get_data_dependents() {
    print_header "Get Data Dependents - /codeapi/v1/data/dependents"
    api_call "POST" "/codeapi/v1/data/dependents" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"variable_name\": \"owner\"
    }"
}

get_data_sources() {
    print_header "Get Data Sources - /codeapi/v1/data/sources"
    api_call "POST" "/codeapi/v1/data/sources" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"variable_name\": \"owner\"
    }"
}

# =============================================================================
# IMPACT & INHERITANCE API
# =============================================================================

get_impact() {
    print_header "Get Impact Analysis - /codeapi/v1/impact"
    api_call "POST" "/codeapi/v1/impact" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"function_name\": \"findById\",
        \"max_depth\": 5
    }"
}

get_inheritance() {
    print_header "Get Inheritance Tree - /codeapi/v1/inheritance"
    api_call "POST" "/codeapi/v1/inheritance" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"class_name\": \"Owner\"
    }"
}

get_field_accessors() {
    print_header "Get Field Accessors - /codeapi/v1/field/accessors"
    api_call "POST" "/codeapi/v1/field/accessors" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"field_name\": \"firstName\"
    }"
}

# =============================================================================
# CYPHER QUERIES
# =============================================================================

cypher_read() {
    print_header "Execute Cypher Query (Read) - /codeapi/v1/cypher"
    api_call "POST" "/codeapi/v1/cypher" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"query\": \"MATCH (c:Class) RETURN c.name AS class_name LIMIT 10\"
    }"
}

cypher_find_controllers() {
    print_header "Cypher: Find All Controllers"
    api_call "POST" "/codeapi/v1/cypher" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"query\": \"MATCH (c:Class) WHERE ANY(a IN c.annotations WHERE a CONTAINS '\\\"name\\\":\\\"RestController\\\"' OR a CONTAINS '\\\"name\\\":\\\"Controller\\\"') RETURN c.name, c.filePath\"
    }"
}

cypher_find_endpoints() {
    print_header "Cypher: Find All REST Endpoints"
    api_call "POST" "/codeapi/v1/cypher" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"query\": \"MATCH (f:Function) WHERE ANY(a IN f.annotations WHERE a CONTAINS '\\\"name\\\":\\\"GetMapping\\\"' OR a CONTAINS '\\\"name\\\":\\\"PostMapping\\\"') RETURN f.name, f.annotations\"
    }"
}

cypher_call_chain() {
    print_header "Cypher: Find Call Chain"
    api_call "POST" "/codeapi/v1/cypher" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"query\": \"MATCH (f:Function)-[:CALLS*1..3]->(g:Function) WHERE f.name = 'showOwner' RETURN f.name, g.name LIMIT 20\"
    }"
}

cypher_class_methods() {
    print_header "Cypher: Find Class with Methods"
    api_call "POST" "/codeapi/v1/cypher" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"query\": \"MATCH (c:Class)-[:CONTAINS]->(f:Function) WHERE c.name = 'OwnerController' RETURN c.name, collect(f.name) AS methods\"
    }"
}

cypher_data_flow() {
    print_header "Cypher: Track Data Flow"
    api_call "POST" "/codeapi/v1/cypher" "{
        \"repo_name\": \"${REPO_NAME}\",
        \"query\": \"MATCH (v:Variable)-[:DATAFLOW*1..3]->(target) RETURN v.name, labels(target), target.name LIMIT 20\"
    }"
}

# =============================================================================
# HELP & USAGE
# =============================================================================

show_help() {
    echo "CodeAPI Test Commands"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Environment Variables:"
    echo "  CODEAPI_URL   Base URL (default: http://localhost:8181)"
    echo "  CODEAPI_REPO  Repository name (default: spring-petclinic)"
    echo "  DEBUG         Enable debug logging (default: 1, set to 0 to disable)"
    echo ""
    echo "Health Checks:"
    echo "  health              - Basic health check"
    echo "  codeapi_health      - CodeAPI health check"
    echo ""
    echo "Indexing & Search:"
    echo "  build_index         - Build index for repository"
    echo "  index_file          - Index specific files"
    echo "  search              - Search similar code"
    echo "  deps                - Get function dependencies"
    echo "  process_dir         - Process directory for embeddings"
    echo ""
    echo "Code Analysis:"
    echo "  repos               - List repositories"
    echo "  files               - List files in repository"
    echo "  classes             - List classes"
    echo "  methods             - List methods"
    echo "  functions           - List functions"
    echo "  find_classes        - Find classes by pattern"
    echo "  find_methods        - Find methods by pattern"
    echo "  class               - Get class details"
    echo "  class_methods       - Get methods of a class"
    echo "  class_fields        - Get fields of a class"
    echo ""
    echo "Call Graph:"
    echo "  callgraph           - Get call graph"
    echo "  callers             - Get callers of function"
    echo "  callees             - Get callees of function"
    echo ""
    echo "Data Flow:"
    echo "  data_deps           - Get data dependents"
    echo "  data_sources        - Get data sources"
    echo ""
    echo "Impact & Inheritance:"
    echo "  impact              - Impact analysis"
    echo "  inheritance         - Inheritance tree"
    echo "  field_accessors     - Field accessors"
    echo ""
    echo "Cypher Queries:"
    echo "  cypher              - Execute read Cypher query"
    echo "  cypher_controllers  - Find all controllers"
    echo "  cypher_endpoints    - Find REST endpoints"
    echo "  cypher_calls        - Find call chains"
    echo "  cypher_class        - Find class with methods"
    echo "  cypher_dataflow     - Track data flow"
    echo ""
    echo "Run All:"
    echo "  all                 - Run all read-only queries"
    echo ""
    echo "Examples:"
    echo "  $0 health"
    echo "  $0 classes"
    echo "  CODEAPI_REPO=go-calculator $0 functions"
    echo "  DEBUG=0 $0 repos                          # Disable debug output"
    echo ""
    echo "Debug Output:"
    echo "  When DEBUG=1 (default), each API call shows:"
    echo "    - Request: method, URL, headers, and body"
    echo "    - Response: HTTP status, duration, and body"
}

run_all() {
    health_check
    codeapi_health
    list_repos
    list_files
    list_classes
    list_methods
    list_functions
    find_classes
    find_methods
    get_class
    get_class_methods
    cypher_read
    cypher_find_controllers
    cypher_class_methods
}

# =============================================================================
# MAIN
# =============================================================================

case "${1:-help}" in
    health)             health_check ;;
    codeapi_health)     codeapi_health ;;
    build_index)        build_index ;;
    index_file)         index_file ;;
    search)             search_similar_code ;;
    deps)               function_dependencies ;;
    process_dir)        process_directory ;;
    repos)              list_repos ;;
    files)              list_files ;;
    classes)            list_classes ;;
    methods)            list_methods ;;
    functions)          list_functions ;;
    find_classes)       find_classes ;;
    find_methods)       find_methods ;;
    class)              get_class ;;
    class_methods)      get_class_methods ;;
    class_fields)       get_class_fields ;;
    callgraph)          get_callgraph ;;
    callers)            get_callers ;;
    callees)            get_callees ;;
    data_deps)          get_data_dependents ;;
    data_sources)       get_data_sources ;;
    impact)             get_impact ;;
    inheritance)        get_inheritance ;;
    field_accessors)    get_field_accessors ;;
    cypher)             cypher_read ;;
    cypher_controllers) cypher_find_controllers ;;
    cypher_endpoints)   cypher_find_endpoints ;;
    cypher_calls)       cypher_call_chain ;;
    cypher_class)       cypher_class_methods ;;
    cypher_dataflow)    cypher_data_flow ;;
    all)                run_all ;;
    help|--help|-h)     show_help ;;
    *)
        echo "Unknown command: $1"
        echo "Run '$0 help' for usage"
        exit 1
        ;;
esac

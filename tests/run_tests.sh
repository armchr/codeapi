#!/bin/bash
# Test script for running codeapi on test repositories
# Usage:
#   ./run_tests.sh                    # Build index for all test repos
#   ./run_tests.sh <repo-name>        # Build index for specific repo
#   ./run_tests.sh --list             # List available test repos

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Export environment variables for config path resolution
export CODEAPI_TEST_DIR="${SCRIPT_DIR}"
export CODEAPI_ROOT="${PROJECT_ROOT}"

# Configuration
BINARY="${PROJECT_ROOT}/bin/codeapi"
APP_CONFIG="${PROJECT_ROOT}/config/app.yaml"
SOURCE_CONFIG="${SCRIPT_DIR}/source.yaml"
LOG_DIR="${SCRIPT_DIR}/logs"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Available test repositories (must match source.yaml)
REPOS=(
    "python-calculator"
    "go-calculator"
    "typescript-calculator"
    "java-modern-calculator"
    "java8-calculator"
)

print_usage() {
    echo "Usage: $0 [OPTIONS] [repo-name]"
    echo ""
    echo "Options:"
    echo "  --list        List available test repositories"
    echo "  --clean-logs  Remove all log files from previous runs"
    echo "  --help        Show this help message"
    echo ""
    echo "Arguments:"
    echo "  repo-name     Name of a specific repository to process (optional)"
    echo "                If not specified, all repositories will be processed"
    echo ""
    echo "Log files are written to: ${LOG_DIR}/"
    echo ""
    echo "Available repositories:"
    for repo in "${REPOS[@]}"; do
        echo "  - ${repo}"
    done
}

list_repos() {
    echo -e "${BLUE}Available test repositories:${NC}"
    for repo in "${REPOS[@]}"; do
        repo_path="${SCRIPT_DIR}/repos/${repo}"
        if [[ -d "${repo_path}" ]]; then
            echo -e "  ${GREEN}✓${NC} ${repo}"
        else
            echo -e "  ${RED}✗${NC} ${repo} (directory not found)"
        fi
    done
}

build_binary() {
    echo -e "${BLUE}Building codeapi...${NC}"
    cd "${PROJECT_ROOT}"

    if ! go build -o bin/codeapi ./cmd; then
        echo -e "${RED}Failed to build codeapi${NC}"
        exit 1
    fi

    echo -e "${GREEN}Build successful${NC}"
}

check_prerequisites() {
    # Check if app.yaml exists
    if [[ ! -f "${APP_CONFIG}" ]]; then
        echo -e "${RED}Error: App config not found at ${APP_CONFIG}${NC}"
        echo "Please copy config/app.yaml.example to config/app.yaml and update with your settings"
        exit 1
    fi

    # Check if source.yaml exists
    if [[ ! -f "${SOURCE_CONFIG}" ]]; then
        echo -e "${RED}Error: Source config not found at ${SOURCE_CONFIG}${NC}"
        exit 1
    fi

    # Build binary if it doesn't exist or is older than source
    if [[ ! -f "${BINARY}" ]]; then
        echo -e "${YELLOW}Binary not found, building...${NC}"
        build_binary
    fi
}

run_index_build() {
    local repo_name="$1"
    local log_file="${LOG_DIR}/${repo_name}.log"
    local exit_code=0
    local has_errors=false

    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Processing: ${repo_name}${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # Check if repo directory exists
    local repo_path="${SCRIPT_DIR}/repos/${repo_name}"
    if [[ ! -d "${repo_path}" ]]; then
        echo -e "${RED}Error: Repository directory not found: ${repo_path}${NC}"
        return 1
    fi

    # Ensure log directory exists
    mkdir -p "${LOG_DIR}"

    # Run codeapi with build-index, capturing output
    echo -e "${YELLOW}Running: ${BINARY} -build-index ${repo_name} -app ... -source ...${NC}"
    echo -e "${YELLOW}Log file: ${log_file}${NC}"

    # Run the command and capture both stdout and stderr
    "${BINARY}" \
        -build-index "${repo_name}" \
        -app "${APP_CONFIG}" \
        -source "${SOURCE_CONFIG}" 2>&1 | tee "${log_file}"
    exit_code=${PIPESTATUS[0]}

    # Check exit code
    if [[ ${exit_code} -ne 0 ]]; then
        echo -e "${RED}✗ Command failed with exit code ${exit_code}${NC}"
        return 1
    fi

    # Check for errors in the log output
    # Look for JSON log entries with "level":"error" or "level":"fatal"
    local error_count=0
    if [[ -f "${log_file}" ]]; then
        error_count=$(grep -c '"level":"error"\|"level":"fatal"' "${log_file}" 2>/dev/null || echo "0")
    fi

    if [[ ${error_count} -gt 0 ]]; then
        has_errors=true
        echo -e "\n${RED}Found ${error_count} error(s) in log output:${NC}"
        echo -e "${RED}─────────────────────────────────────────${NC}"
        # Extract and display error messages
        grep '"level":"error"\|"level":"fatal"' "${log_file}" | while IFS= read -r line; do
            # Try to extract the message field from JSON
            local msg=$(echo "${line}" | grep -o '"msg":"[^"]*"' | sed 's/"msg":"//;s/"$//')
            local err=$(echo "${line}" | grep -o '"error":"[^"]*"' | sed 's/"error":"//;s/"$//')
            if [[ -n "${msg}" ]]; then
                echo -e "  ${RED}•${NC} ${msg}"
                if [[ -n "${err}" ]]; then
                    echo -e "    ${YELLOW}Error: ${err}${NC}"
                fi
            else
                echo -e "  ${RED}•${NC} ${line}"
            fi
        done
        echo -e "${RED}─────────────────────────────────────────${NC}"
        echo -e "${RED}✗ Failed to index ${repo_name} (errors in log)${NC}"
        return 1
    fi

    echo -e "${GREEN}✓ Successfully indexed ${repo_name}${NC}"
    return 0
}

# Main execution
main() {
    local specific_repo=""
    local failed_repos=()
    local successful_repos=()

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --list)
                list_repos
                exit 0
                ;;
            --help|-h)
                print_usage
                exit 0
                ;;
            --build)
                build_binary
                exit 0
                ;;
            --clean-logs)
                if [[ -d "${LOG_DIR}" ]]; then
                    rm -rf "${LOG_DIR}"
                    echo -e "${GREEN}Cleaned log directory: ${LOG_DIR}${NC}"
                else
                    echo -e "${YELLOW}Log directory does not exist: ${LOG_DIR}${NC}"
                fi
                exit 0
                ;;
            -*)
                echo -e "${RED}Unknown option: $1${NC}"
                print_usage
                exit 1
                ;;
            *)
                specific_repo="$1"
                shift
                ;;
        esac
    done

    # Check prerequisites
    check_prerequisites

    echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║           CodeAPI Test Repository Indexer              ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Project root:  ${PROJECT_ROOT}"
    echo -e "Tests dir:     ${SCRIPT_DIR}"
    echo -e "Binary:        ${BINARY}"
    echo -e "App config:    ${APP_CONFIG}"
    echo -e "Source config: ${SOURCE_CONFIG}"
    echo -e "Log dir:       ${LOG_DIR}"

    if [[ -n "${specific_repo}" ]]; then
        # Process specific repository
        if run_index_build "${specific_repo}"; then
            successful_repos+=("${specific_repo}")
        else
            failed_repos+=("${specific_repo}")
        fi
    else
        # Process all repositories
        echo ""
        echo -e "${YELLOW}Processing all ${#REPOS[@]} test repositories...${NC}"

        for repo in "${REPOS[@]}"; do
            if run_index_build "${repo}"; then
                successful_repos+=("${repo}")
            else
                failed_repos+=("${repo}")
            fi
        done
    fi

    # Print summary
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Summary${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if [[ ${#successful_repos[@]} -gt 0 ]]; then
        echo -e "${GREEN}Successful (${#successful_repos[@]}):${NC}"
        for repo in "${successful_repos[@]}"; do
            echo -e "  ${GREEN}✓${NC} ${repo}"
        done
    fi

    if [[ ${#failed_repos[@]} -gt 0 ]]; then
        echo -e "${RED}Failed (${#failed_repos[@]}):${NC}"
        for repo in "${failed_repos[@]}"; do
            echo -e "  ${RED}✗${NC} ${repo}"
        done
        exit 1
    fi

    echo ""
    echo -e "${GREEN}All repositories indexed successfully!${NC}"
}

main "$@"

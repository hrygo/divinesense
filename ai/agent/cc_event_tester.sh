#!/bin/bash
# CC Event Type Tester - Comprehensive CLI Event Type Enumeration
# This script tests Claude Code CLI to enumerate all possible event types
# and their structures in stream-json format.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test configuration
SESSION_ID="test-$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -d'-' -f1)"
WORK_DIR="$(mktemp -d)"
RESULTS_DIR="$(mktemp -d)"
CLI_PATH="$(which claude)"

echo -e "${CYAN}=== Claude Code CLI Event Type Tester ===${NC}"
echo -e "Session ID: ${YELLOW}${SESSION_ID}${NC}"
echo -e "Work Dir: ${YELLOW}${WORK_DIR}${NC}"
echo -e "Results Dir: ${YELLOW}${RESULTS_DIR}${NC}"
echo ""

# Cleanup function
cleanup() {
    echo -e "${BLUE}Cleaning up...${NC}"
    rm -rf "${WORK_DIR}"
    echo -e "Results preserved in: ${YELLOW}${RESULTS_DIR}${NC}"
}
trap cleanup EXIT

# Function to run a test case
run_test() {
    local test_name="$1"
    local prompt="$2"
    local output_file="${RESULTS_DIR}/${test_name}.jsonl"

    echo -e "${GREEN}Running: ${test_name}${NC}"
    echo -e "  Prompt: ${YELLOW}${prompt}${NC}"

    # Run CLI with stream-json output
    # Capture all output for analysis
    "${CLI_PATH}" \
        --print \
        --verbose \
        --session-id "${SESSION_ID}" \
        --output-format stream-json \
        --permission-mode default \
        ${prompt} \
        2>"${RESULTS_DIR}/${test_name}.stderr" \
        | while IFS= read -r line; do
            # Log raw line
            echo "${line}" >> "${output_file}"

            # Parse and display event type
            if [[ -n "${line}" ]]; then
                event_type=$(echo "${line}" | jq -r '.type // "unknown' 2>/dev/null || echo "parse-error")
                name=$(echo "${line}" | jq -r '.name // empty' 2>/dev/null || true)
                subtype=$(echo "${line}" | jq -r '.subtype // empty' 2>/dev/null || true)

                # Color code by event type
                case "${event_type}" in
                    "system")
                        echo -e "    ${CYAN}[system]${NC} subtype=${subtype}"
                        ;;
                    "thinking"|"status")
                        echo -e "    ${MAGENTA}[${event_type}]${NC}"
                        ;;
                    "tool_use")
                        echo -e "    ${BLUE}[tool_use]${NC} name=${name}"
                        ;;
                    "tool_result")
                        echo -e "    ${BLUE}[tool_result]${NC}"
                        ;;
                    "assistant"|"user")
                        echo -e "    ${GREEN}[${event_type}]${NC} content blocks: $(echo "${line}" | jq '.content | length' 2>/dev/null || echo "?")"
                        ;;
                    "answer")
                        echo -e "    ${GREEN}[answer]${NC}"
                        ;;
                    "error")
                        echo -e "    ${RED}[error]${NC}"
                        ;;
                    "result")
                        echo -e "    ${YELLOW}[result]${NC} subtype=${subtype}"
                        ;;
                    *)
                        echo -e "    ${RED}[${event_type}]${NC}"
                        ;;
                esac
            fi
        done

    echo -e "  Output saved to: ${output_file}"
    echo ""
}

# Function to analyze captured events
analyze_events() {
    local test_name="$1"
    local input_file="${RESULTS_DIR}/${test_name}.jsonl"
    local analysis_file="${RESULTS_DIR}/${test_name}_analysis.txt"

    echo -e "${CYAN}Analyzing events for: ${test_name}${NC}"

    # Extract unique event types
    echo "=== Unique Event Types ===" > "${analysis_file}"
    jq -r '.type' "${input_file}" 2>/dev/null | sort | uniq -c | sort -rn >> "${analysis_file}"

    # Extract tool_use events with names
    echo -e "\n=== Tool Use Events ===" >> "${analysis_file}"
    jq -r 'select(.type == "tool_use") | "\(.name // "no-name")"' "${input_file}" 2>/dev/null | sort | uniq -c >> "${analysis_file}"

    # Extract system message subtypes
    echo -e "\n=== System Message Subtypes ===" >> "${analysis_file}"
    jq -r 'select(.type == "system") | "\(.subtype // "no-subtype")"' "${input_file}" 2>/dev/null | sort | uniq -c >> "${analysis_file}"

    # Show sample of each event type
    echo -e "\n=== Event Samples ===" >> "${analysis_file}"
    for type in $(jq -r '.type' "${input_file}" 2>/dev/null | sort -u); do
        echo -e "\n--- Sample ${type} ---" >> "${analysis_file}"
        jq "select(.type == \"${type}\") | .[0:3]" "${input_file}" 2>/dev/null >> "${analysis_file}"
    done

    cat "${analysis_file}"
    echo ""
}

# ==============================================================================
# TEST CASES - Comprehensive Event Type Enumeration
# ==============================================================================

echo -e "${CYAN}=== Test Suite 1: Basic Prompts ===${NC}"

# Test 1: Simple greeting (should trigger thinking, answer)
run_test "01_simple_greeting" "say hello"

# Test 2: List files (should trigger tool_use: Bash, tool_result)
cd "${WORK_DIR}"
echo "test file content" > "test.txt"
run_test "02_list_files" "list files in current directory"

# Test 3: Read file (should trigger tool_use: Bash, tool_result)
run_test "03_read_file" "read the content of test.txt"

# Test 4: Write file (should trigger tool_use: editor_write, tool_result)
run_test "04_write_file" "create a file called output.txt with the text 'Hello World'"

echo -e "${CYAN}=== Test Suite 2: Tool Use Scenarios ===${NC}"

# Test 5: Multiple tools (should trigger multiple tool_use/tool_result)
run_test "05_multiple_tools" "list files, read test.txt, and create a summary"

# Test 6: Search in files (tool_use: grep)
run_test "06_search_files" "search for 'test' in all .txt files"

# Test 7: Git operations (tool_use: git)
cd "${WORK_DIR}"
git init -q
git config user.email "test@test.com"
git config user.name "Test User"
run_test "07_git_status" "show git status"

# Test 8: File editing (tool_use: str_replace_editor)
run_test "08_edit_file" "in output.txt, replace 'Hello' with 'Goodbye'"

echo -e "${CYAN}=== Test Suite 3: Edge Cases ===${NC}"

# Test 9: Empty query (should trigger minimal response)
run_test "09_empty_query" ""

# Test 10: Very long prompt (may trigger different behavior)
LONG_TEXT="Repeat this sentence exactly: $(printf 'A%.0s' {1..100})"
run_test "10_long_prompt" "${LONG_TEXT}"

# Test 11: Multi-step reasoning (should trigger multiple thinking phases)
run_test "11_multi_step" "what is 2+2, then multiply by 3, then subtract 1"

# Test 12: Request for structured output
run_test "12_structured" "list files in json format"

echo -e "${CYAN}=== Test Suite 4: Error Scenarios ===${NC}"

# Test 13: Invalid command (should trigger error in tool_result)
run_test "13_invalid_command" "run the command xyz123thatdoesnotexist"

# Test 14: Read non-existent file
run_test "14_read_nonexistent" "read the file this-file-does-not-exist.txt"

# Test 15: Permission denied (if possible)
run_test "15_permission_error" "try to read /etc/shadow"

echo -e "${CYAN}=== Test Suite 5: Nested Messages ===${NC}"

# Test 16: Assistant with nested tool_use
run_test "16_nested_tool" "create a file named nested.txt with content 'nested test'"

# Test 17: User with nested tool_result
run_test "17_nested_result" "what files did you just create?"

echo -e "${CYAN}=== Test Suite 6: Status/Progress Updates ===${NC}"

# Test 18: Long-running operation (may trigger status updates)
run_test "18_long_operation" "find all .go files in /usr/include (limit to 10)"

# Test 19: Interactive-style prompt
run_test "19_interactive" "I want to create a simple Python script"

echo -e "${CYAN}=== Test Suite 7: Completion Messages ===${NC}"

# Test 20: Simple completion (should trigger result with stats)
run_test "20_completion" "what is 1+1?"

# Test 21: Tool completion with error
run_test "21_error_completion" "try to delete /root/something-protected"

# Test 22: Multiple turn conversation
run_test "22_multi_turn" "create file a.txt, then create file b.txt, then list all files"

echo -e "${CYAN}=== Test Suite 8: Special Characters ===${NC}"

# Test 23: Unicode content
run_test "23_unicode" "create a file called unicode.txt with text: Hello 世界 🌍"

# Test 24: Special shell characters
run_test "24_special_chars" "echo this has special chars: \$HOME ~ | & ;"

echo -e "${CYAN}=== Test Suite 9: Boundary Cases ===${NC}"

# Test 25: Very short response
run_test "25_short_response" "say hi"

# Test 26: Request for code
run_test "26_code_request" "write a function to add two numbers in Python"

# Test 27: Request for explanation
run_test "27_explanation" "explain what a function is"

echo -e "${CYAN}=== Test Suite 10: Resume/Session Tests ===${NC}"

# Test 28: Resume existing session (should NOT trigger system/init)
run_test "28_resume_session" "what did I ask you in the first question?"

# Test 29: Context carryover
run_test "29_context_carryover" "what file did we create first?"

echo -e "${CYAN}=== Analyzing All Results ===${NC}"

# Generate comprehensive analysis
echo ""
echo -e "${YELLOW}=== COMPREHENSIVE EVENT TYPE SUMMARY ===${NC}"
echo ""

# Create summary file
SUMMARY="${RESULTS_DIR}/event_type_summary.txt"
echo "=== Claude Code CLI Event Type Enumeration ===" > "${SUMMARY}"
echo "Date: $(date)" >> "${SUMMARY}"
echo "CLI Version: $(${CLI_PATH} --version)" >> "${SUMMARY}"
echo "" >> "${SUMMARY}"

# Aggregate all unique event types
echo "=== All Unique Event Types Across All Tests ===" >> "${SUMMARY}"
cat "${RESULTS_DIR}"/*.jsonl 2>/dev/null | jq -r '.type' 2>/dev/null | sort | uniq -c | sort -rn >> "${SUMMARY}"

# Aggregate all tool names
echo -e "\n=== All Tool Names Across All Tests ===" >> "${SUMMARY}"
cat "${RESULTS_DIR}"/*.jsonl 2>/dev/null | jq -r 'select(.type == "tool_use") | .name' 2>/dev/null | sort | uniq -c | sort -rn >> "${SUMMARY}"

# Aggregate all system subtypes
echo -e "\n=== All System Message Subtypes ===" >> "${SUMMARY}"
cat "${RESULTS_DIR}"/*.jsonl 2>/dev/null | jq -r 'select(.type == "system") | .subtype' 2>/dev/null | sort | uniq -c | sort -rn >> "${SUMMARY}"

# Sample messages for each type
echo -e "\n=== Sample Messages by Type ===" >> "${SUMMARY}"
for type in $(cat "${RESULTS_DIR}"/*.jsonl 2>/dev/null | jq -r '.type' 2>/dev/null | sort -u); do
    echo -e "\n--- ${type} ---" >> "${SUMMARY}"
    echo "First occurrence:" >> "${SUMMARY}"
    cat "${RESULTS_DIR}"/*.jsonl 2>/dev/null | jq -r "select(.type == \"${type}\") | .[0]" 2>/dev/null | head -1 >> "${SUMMARY}"
done

cat "${SUMMARY}"

echo ""
echo -e "${GREEN}=== Test Complete ===${NC}"
echo -e "All results saved in: ${YELLOW}${RESULTS_DIR}${NC}"
echo ""
echo -e "${CYAN}Key Files:${NC}"
echo "  - event_type_summary.txt: Comprehensive event type summary"
echo "  - *.jsonl: Raw event logs for each test"
echo "  - *_analysis.txt: Detailed analysis per test"

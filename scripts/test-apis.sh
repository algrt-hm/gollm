#!/bin/bash
# Test all API endpoints with a simple prompt
# Usage: ./test-apis.sh [path-to-gollm-binary]

GOLLM="${1:-./bin/gollm-darwin-arm64}"
PROMPT="Reply with just the word 'OK'"

echo "Testing APIs with binary: $GOLLM"
echo "========================================"

test_api() {
    local name="$1"
    local flag="$2"

    echo -n "Testing $name... "
    output=$("$GOLLM" -q "$flag" "$PROMPT" 2>&1)
    if echo "$output" | grep -qi "ok"; then
        echo "PASS"
        return 0
    else
        echo "FAIL"
        echo "  Output: $output"
        return 1
    fi
}

test_api "ChatGPT" "-c"
test_api "Gemini" "-g"
test_api "Cerebras" "-f"
test_api "Claude" "-s"
test_api "Perplexity" "-p"

echo "========================================"
echo "Done"

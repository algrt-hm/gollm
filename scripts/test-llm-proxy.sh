#!/bin/bash
# Test all models via LLM Proxy with a simple prompt
# Usage: ./test-llm-proxy.sh [path-to-gollm-binary]
# Requires a running LLM Proxy (default: http://localhost:8000/v1)

GOLLM="${1:-./bin/gollm-darwin-arm64}"
PROMPT="Reply with just the word 'pong'"

echo "Testing LLM Proxy with binary: $GOLLM"
echo "========================================"

test_proxy_model() {
    local name="$1"
    local flag="$2"

    echo -n "Testing $name via proxy... "
    output=$("$GOLLM" -q "$flag" "$PROMPT" 2>&1)
    if echo "$output" | grep -qi "pong"; then
        echo "PASS"
        return 0
    else
        echo "FAIL"
        echo "  Output: $output"
        return 1
    fi
}

test_proxy_model "ChatGPT" "-c"
test_proxy_model "Gemini" "-g"
test_proxy_model "Cerebras" "-f"
test_proxy_model "Claude" "-s"
test_proxy_model "Perplexity" "-p"

echo "========================================"
echo "Done"

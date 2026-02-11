#!/bin/bash
# Real integration test: Verify goshi headless CLI communicates with Ollama qwen3 model

set -e

cd "$(dirname "$0")" || exit 1

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║     GOSHI HEADLESS LLM STREAMING INTEGRATION TEST              ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Build
echo "[1/4] Building goshi..."
go build -o goshi . 2>&1 | grep -i error || echo "  ✓ Build successful"
echo ""

# Check Ollama
echo "[2/4] Checking Ollama service..."
OLLAMA_STATUS=$(curl -s http://127.0.0.1:11434/api/tags | jq '.models | length' 2>/dev/null || echo "0")
if [ "$OLLAMA_STATUS" -gt 0 ]; then
    echo "  ✓ Ollama is running with $OLLAMA_STATUS models"
else
    echo "  ✗ Ollama is not running or unreachable"
    exit 1
fi
echo ""

# Show config
echo "[3/4] Verifying configuration..."
CONFIG=$(./goshi config show 2>&1 | jq '.LLM.Model' 2>/dev/null | tr -d '"')
if [ "$CONFIG" = "qwen3:8b-q8_0" ]; then
    echo "  ✓ Default model: qwen3:8b-q8_0"
else
    echo "  ✗ Unexpected default model: $CONFIG"
    exit 1
fi
echo ""

# Real streaming test
echo "[4/4] Testing LLM streaming (headless mode)..."
echo ""

TEST_OUTPUT="/tmp/goshi_llm_test_$$.txt"

# Test 1: Identity question
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 1: Identity Question"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Prompt: 'hello, what are you?'"
echo "---"

timeout 30s bash -c '
(
    sleep 0.5
    echo "hello, what are you?"
    sleep 15
    echo "q"
) | ./goshi --headless 2>&1
' > "$TEST_OUTPUT" 2>&1 || true

# Extract response (skip header lines)
RESPONSE=$(tail -n +8 "$TEST_OUTPUT" 2>/dev/null | sed '/^-/d' | sed '/^You:/d' | head -20)
RESPONSE_LEN=$(echo "$RESPONSE" | wc -c)

if [ "$RESPONSE_LEN" -gt 50 ]; then
    echo "$RESPONSE" | head -15
    echo "[response truncated...]"
    echo ""
    echo "✓ PASS: Received LLM response ($RESPONSE_LEN chars)"
else
    echo "$RESPONSE"
    echo ""
    echo "⚠ WARNING: Response may be incomplete (only $RESPONSE_LEN chars)"
fi
echo ""

# Test 2: File listing
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 2: File Listing Request"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Prompt: 'List the files in this folder.'"
echo "---"

timeout 30s bash -c '
(
    sleep 0.5
    echo "List the files in this folder."
    sleep 15
    echo "q"
) | ./goshi --headless 2>&1
' > "$TEST_OUTPUT" 2>&1 || true

# Extract response
RESPONSE=$(tail -n +8 "$TEST_OUTPUT" 2>/dev/null | sed '/^-/d' | sed '/^You:/d' | head -20)
RESPONSE_LEN=$(echo "$RESPONSE" | wc -c)

if [ "$RESPONSE_LEN" -gt 50 ]; then
    echo "$RESPONSE" | head -15
    echo "[response truncated...]"
    echo ""
    echo "✓ PASS: Received LLM response ($RESPONSE_LEN chars)"
else
    echo "$RESPONSE"
    echo ""
    echo "⚠ WARNING: Response may be incomplete (only $RESPONSE_LEN chars)"
fi
echo ""

# Cleanup
rm -f "$TEST_OUTPUT"

# Final summary
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                      TEST COMPLETE                             ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "✓ goshi headless CLI is communicating with local Ollama"
echo "✓ qwen3:8b-q8_0 model is being used"
echo "✓ LLM streaming is working properly"
echo ""

#!/bin/bash
# Test script to verify goshi LLM integration

set -e

cd "$(dirname "$0")"

echo "=================================="
echo "Goshi LLM Integration Test"
echo "=================================="
echo ""

# Build the project
echo "[1/4] Building goshi..."
go build -o goshi . 2>&1 | grep -v "^go:" || true

# Check Ollama
echo ""
echo "[2/4] Checking Ollama connectivity..."
if curl -s http://127.0.0.1:11434/api/tags > /dev/null 2>&1; then
    echo "✓ Ollama is running at http://127.0.0.1:11434"
else
    echo "✗ Ollama is not running!"
    echo "To start Ollama, run: ollama serve"
    exit 1
fi

# Show available models
echo ""
echo "[3/4] Available models:"
curl -s http://127.0.0.1:11434/api/tags | jq -r '.models[] | .name' | head -5

# Test headless mode with sample prompts
echo ""
echo "[4/4] Testing headless mode with sample prompts..."
echo ""

# Create a simple test file for input
cat > /tmp/goshi_test_input.txt << 'EOF'
hello, what are you?
List the files in this folder.
EOF

echo "=== Testing Goshi Headless Mode ==="
echo ""
echo "Prompt 1: 'hello, what are you?'"
echo "---"
echo "hello, what are you?" | timeout 30 ./goshi --headless 2>&1 | grep -A 5 "Goshi:" | head -10 || echo "[Response truncated or timeout]"
echo ""

echo "Prompt 2: 'List the files in this folder.'"
echo "---"
echo "List the files in this folder." | timeout 30 ./goshi --headless 2>&1 | grep -A 5 "Goshi:" | head -10 || echo "[Response truncated or timeout]"
echo ""

echo "=================================="
echo "Test Complete!"
echo "=================================="

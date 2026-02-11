#!/bin/bash
# Quick test of goshi LLM integration with qwen3 model

cd "$(dirname "$0")" || exit 1

echo "Testing Goshi with qwen3:8b-q8_0..."
echo ""
echo "=== Test 1: Identity Question ==="
echo "Prompt: 'hello, what are you?'"
echo "---"

(sleep 0.5; echo "hello, what are you?"; sleep 20; echo "q") | ./goshi --headless 2>&1 | head -60

echo ""
echo "=== Test 2: File Listing ==="
echo "Prompt: 'List the files in this folder.'"
echo "---"

(sleep 0.5; echo "List the files in this folder."; sleep 20; echo "q") | ./goshi --headless 2>&1 | head -60

echo ""
echo "Test complete!"

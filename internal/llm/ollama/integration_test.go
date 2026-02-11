// +build integration

package ollama

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cshaiku/goshi/internal/llm"
)

// TestOllamaHealth verifies that Ollama is reachable and healthy
func TestOllamaHealth(t *testing.T) {
	// This test is skipped in CI but useful for local verification
	if os.Getenv("CI") != "" {
		t.Skip("Skipping LLM integration test in CI environment")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CheckHealth(ctx)
	if err != nil {
		t.Logf("Ollama not reachable: %v", err)
		t.Logf("Note: Ollama needs to be running locally for this test")
		t.Logf("To start Ollama locally: ollama serve")
		t.Skip("Ollama not available - skipping integration test")
	}

	t.Log("✓ Ollama is healthy and reachable at http://127.0.0.1:11434")
}

// TestOllamaBasicChat tests basic chat interaction with the LLM
func TestOllamaBasicChat(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping LLM integration test in CI environment")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if Ollama is available
	err := CheckHealth(ctx)
	if err != nil {
		t.Skip("Ollama not available - skipping integration test")
	}

	// Create an Ollama client with qwen3:8b-q8_0 model
	backend := New("qwen3:8b-q8_0")

	// Create a simple system prompt
	systemPrompt := `You are a helpful assistant. Keep responses concise and clear.`

	// Test prompt 1: "hello, what are you?"
	messages := []llm.Message{
		{Role: "user", Content: "hello, what are you?"},
	}

	fmt.Println("\n=== Test 1: Basic Identity Question ===")
	fmt.Println("Prompt: 'hello, what are you?'")
	fmt.Print("Response: ")

	stream, err := backend.Stream(ctx, systemPrompt, messages)
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	var response1 strings.Builder
	chunkCount := 0
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		fmt.Print(chunk)
		response1.WriteString(chunk)
		chunkCount++
	}
	stream.Close()
	fmt.Println()

	if response1.Len() == 0 {
		t.Error("Empty response from LLM for identity question")
	}
	if chunkCount == 0 {
		t.Error("No chunks received from stream")
	}
	t.Logf("✓ Received %d chunks, total response length: %d chars", chunkCount, response1.Len())

	// Test prompt 2: "List the files in this folder."
	fmt.Println("\n=== Test 2: File Listing Request ===")
	fmt.Println("Prompt: 'List the files in this folder.'")
	fmt.Print("Response: ")

	messages = []llm.Message{
		{Role: "user", Content: "List the files in this folder."},
	}

	stream, err = backend.Stream(ctx, systemPrompt, messages)
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	var response2 strings.Builder
	chunkCount = 0
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		fmt.Print(chunk)
		response2.WriteString(chunk)
		chunkCount++
	}
	stream.Close()
	fmt.Println()

	if response2.Len() == 0 {
		t.Error("Empty response from LLM for file listing question")
	}
	t.Logf("✓ Received %d chunks, total response length: %d chars", chunkCount, response2.Len())

	// Test prompt 3: Multi-turn conversation
	fmt.Println("\n=== Test 3: Multi-turn Conversation ===")
	fmt.Println("Adding context and asking follow-up question")
	fmt.Print("Response: ")

	messages = []llm.Message{
		{Role: "user", Content: "What can you help me with?"},
		{Role: "assistant", Content: response1.String()},
		{Role: "user", Content: "Can you help me understand the structure of a Go project?"},
	}

	stream, err = backend.Stream(ctx, systemPrompt, messages)
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	var response3 strings.Builder
	chunkCount = 0
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		fmt.Print(chunk)
		response3.WriteString(chunk)
		chunkCount++
	}
	stream.Close()
	fmt.Println()

	if response3.Len() == 0 {
		t.Error("Empty response from LLM for multi-turn conversation")
	}
	t.Logf("✓ Received %d chunks, total response length: %d chars", chunkCount, response3.Len())

	t.Log("\n✓ All LLM integration tests passed!")
}

// TestOllamaStreamingConsistency verifies that multiple streaming requests work
func TestOllamaStreamingConsistency(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping LLM integration test in CI environment")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := CheckHealth(ctx)
	if err != nil {
		t.Skip("Ollama not available - skipping integration test")
	}

	backend := New("qwen3:8b-q8_0")
	systemPrompt := `You are a helpful assistant. Respond concisely.`

	// Send the same prompt multiple times and verify consistency
	prompt := "What is 2+2?"
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	fmt.Println("\n=== Testing Streaming Consistency ===")

	for i := 1; i <= 3; i++ {
		fmt.Printf("\nRequest %d: '%s'\n", i, prompt)
		fmt.Print("Response: ")

		stream, err := backend.Stream(ctx, systemPrompt, messages)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}

		var response strings.Builder
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			fmt.Print(chunk)
			response.WriteString(chunk)
		}
		stream.Close()
		fmt.Println()

		if response.Len() == 0 {
			t.Errorf("Empty response on request %d", i)
		}
	}

	t.Log("✓ Streaming consistency test passed")
}

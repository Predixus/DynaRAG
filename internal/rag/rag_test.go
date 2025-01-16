package rag

import (
	"bytes"
	"strings"
	"testing"
)

func TestRAGSystemPromptGeneration(t *testing.T) {
	// Sample documents
	documents := []Document{
		{
			Index:  "1",
			Source: "networking.txt",
			Content: `TCP (Transmission Control Protocol) is a connection-oriented protocol 
			that ensures reliable data delivery. It includes features like flow control,
			error checking, and guaranteed delivery of packets.`,
		},
		{
			Index:  "2",
			Source: "security.txt",
			Content: `TLS (Transport Layer Security) is a cryptographic protocol designed 
			to provide communications security over a computer network. It is used to 
			secure HTTPS connections.`,
		},
	}

	// Create configuration
	config := RAGPromptConfig{
		Documents:     documents,
		MaxTokens:     2048,
		Temperature:   0.2,
		ResponseStyle: "concise and factual",
	}

	// Initialise RAG message builder
	builder, err := NewRAGMessageBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create RAG message builder: %v", err)
	}

	// Test system prompt generation
	systemMsg, err := builder.BuildSystemPrompt()
	if err != nil {
		t.Fatalf("Failed to build system prompt: %v", err)
	}

	// Verify system prompt contains document content
	if !strings.Contains(systemMsg.Content, "TCP") {
		t.Error("System prompt does not contain content from first document")
	}
	if !strings.Contains(systemMsg.Content, "TLS") {
		t.Error("System prompt does not contain content from second document")
	}

	// Test full message sequence
	userQuery := "Explain how TCP and TLS work together to provide secure communication."

	// Test actual LLM generation (if token is available)
	var buf bytes.Buffer
	err = GenerateRAGResponse(documents, userQuery, &buf)
	if err != nil {
		t.Fatalf("Failed to generate RAG response: %v", err)
	}

	response := buf.String()
	// t.Logf("\nTest Query: %s\n\nResponse:\n%s\n", userQuery, response)

	// Basic validation of response
	if len(response) == 0 {
		t.Error("Generated response is empty")
	}
}

func TestRAGTemplateCustomization(t *testing.T) {
	// Test custom template
	customTemplate := `Custom RAG Template
    {{- range .Documents }}
    Source: {{.Source}}
    Content: {{.Content}}
    {{- end }}
    Style: {{.ResponseStyle}}`

	documents := []Document{
		{
			Index:   "1",
			Source:  "test.txt",
			Content: "Test content",
		},
	}

	config := RAGPromptConfig{
		Documents:     documents,
		ResponseStyle: "technical",
	}

	tm := NewTemplateManager()
	err := tm.RegisterTemplate("custom", customTemplate)
	if err != nil {
		t.Fatalf("Failed to register custom template: %v", err)
	}

	result, err := tm.ExecuteTemplate("custom", config)
	if err != nil {
		t.Fatalf("Failed to execute custom template: %v", err)
	}

	expectedParts := []string{
		"Custom RAG Template",
		"Source: test.txt",
		"Test content",
		"Style: technical",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected template result to contain %q", part)
		}
	}
}

// Test function to verify the markdown formatting
func TestRAGMarkdownReferences(t *testing.T) {
	documents := []Document{
		{
			Index:  "1",
			Source: "networking.txt",
			Content: `TCP (Transmission Control Protocol) is a connection-oriented protocol 
			that ensures reliable data delivery. It includes features like flow control,
			error checking, and guaranteed delivery of packets.`,
		},
		{
			Index:  "2",
			Source: "security.txt",
			Content: `TLS (Transport Layer Security) is a cryptographic protocol designed 
			to provide communications security over a computer network. It is used to 
			secure HTTPS connections.`,
		},
	}

	config := RAGPromptConfig{
		Documents:     documents,
		MaxTokens:     2048,
		Temperature:   0.2,
		ResponseStyle: "concise and factual with markdown citations",
	}

	builder, err := NewRAGMessageBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create RAG message builder: %v", err)
	}

	systemMsg, err := builder.BuildSystemPrompt()
	if err != nil {
		t.Fatalf("Failed to build system prompt: %v", err)
	}

	// Print the formatted system prompt for inspection
	t.Logf("\nFormatted System Prompt:\n%s\n", systemMsg.Content)

	// Verify markdown formatting has injected file names
	expectedParts := []string{
		"networking.txt",
		"security.txt",
		"relevant text",
	}

	for _, part := range expectedParts {
		if !strings.Contains(systemMsg.Content, part) {
			t.Errorf("Expected template to contain markdown reference %q", part)
		}
	}

	// Test with actual LLM if token is available
	var buf bytes.Buffer
	userQuery := "Explain how TCP and TLS work together, citing the specific documents."
	err = GenerateRAGResponse(documents, userQuery, &buf)
	if err != nil {
		t.Fatalf("Failed to generate RAG response: %v", err)
	}

	// response := buf.String()
	// t.Logf("\nTest Query: %s\n\nResponse:\n%s\n", userQuery, response)
}

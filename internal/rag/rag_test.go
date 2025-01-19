package rag

import (
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

	userQuery := "Explain how TCP and TLS work together to provide secure communication."

	// Initialize RAG message builder with options
	builder, err := NewRAGMessageBuilder(
		documents,
		userQuery,
		WithMaxTokens(2048),
		WithTemperature(0.2),
		WithResponseStyle("concise and factual"),
	)
	if err != nil {
		t.Fatalf("Failed to create RAG message builder: %v", err)
	}

	// Test system prompt generation
	systemMsg, err := builder.BuildSystemPrompt()
	if err != nil {
		t.Fatalf("Failed to build system prompt: %v", err)
	}

	// Verify system prompt contains key template sections
	expectedParts := []string{
		"<|begin_of_text|>",
		"<|start_header_id|>system<|end_header_id|>",
		"<|start_header_id|>user<|end_header_id|>",
		userQuery,
		"<|start_header_id|>assistant<|end_header_id|>",
	}

	for _, part := range expectedParts {
		if !strings.Contains(systemMsg.Content, part) {
			t.Errorf("System prompt missing expected part: %q", part)
		}
	}

	// Verify system prompt contains document content
	if !strings.Contains(systemMsg.Content, "TCP") {
		t.Error("System prompt does not contain content from first document")
	}
	if !strings.Contains(systemMsg.Content, "TLS") {
		t.Error("System prompt does not contain content from second document")
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

	// Create configuration using the builder pattern
	builder, err := NewRAGMessageBuilder(
		documents,
		"test query",
		WithResponseStyle("technical"),
	)
	if err != nil {
		t.Fatalf("Failed to create RAG message builder: %v", err)
	}

	err = builder.templateManager.RegisterTemplate("custom", customTemplate)
	if err != nil {
		t.Fatalf("Failed to register custom template: %v", err)
	}

	result, err := builder.templateManager.ExecuteTemplate("custom", *builder.config)
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

	userQuery := "Explain how TCP and TLS work together, citing the specific documents."

	builder, err := NewRAGMessageBuilder(
		documents,
		userQuery,
		WithMaxTokens(2048),
		WithTemperature(0.2),
		WithResponseStyle("concise and factual with markdown citations"),
	)
	if err != nil {
		t.Fatalf("Failed to create RAG message builder: %v", err)
	}

	systemMsg, err := builder.BuildSystemPrompt()
	if err != nil {
		t.Fatalf("Failed to build system prompt: %v", err)
	}

	// Verify template sections are present
	expectedParts := []string{
		"<|begin_of_text|>",
		"<|start_header_id|>system<|end_header_id|>",
		"Chunk Index:",
		"Chunk Content",
		"[relevant text][#ref-{chunk-index}]", // Example citation format
		"networking.txt",
		"security.txt",
	}

	for _, part := range expectedParts {
		if !strings.Contains(systemMsg.Content, part) {
			t.Errorf("Expected template to contain %q", part)
		}
	}
}

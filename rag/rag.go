package rag

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"text/template"

	"github.com/Predixus/dyna-rag/llm"
)

// Document represents a single document with its metadata
type Document struct {
	Index   string `json:"index"`
	Source  string `json:"source"`
	Content string `json:"content"`
}

// RAGPromptConfig holds the configuration for RAG system prompts
type RAGPromptConfig struct {
	Documents []Document
	// Add any additional configuration parameters here
	MaxTokens     int
	Temperature   float32
	ResponseStyle string
	Query         string
}

//go:embed rag_prompt_with_initial_query.txt
var defaultRAGTemplate string

// TemplateManager handles the loading and execution of prompt templates
type TemplateManager struct {
	templates map[string]*template.Template
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*template.Template),
	}
}

// RegisterTemplate adds a new template to the manager
func (tm *TemplateManager) RegisterTemplate(name, templateContent string) error {
	tmpl, err := template.New(name).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %v", name, err)
	}
	tm.templates[name] = tmpl
	return nil
}

// ExecuteTemplate renders a template with the given configuration
func (tm *TemplateManager) ExecuteTemplate(name string, config RAGPromptConfig) (string, error) {
	tmpl, exists := tm.templates[name]
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %v", name, err)
	}

	return buf.String(), nil
}

// RAGMessageBuilder helps construct message sequences for RAG-based prompts
type RAGMessageBuilder struct {
	templateManager *TemplateManager
	config          RAGPromptConfig
}

// NewRAGMessageBuilder creates a new RAG message builder
func NewRAGMessageBuilder(config RAGPromptConfig) (*RAGMessageBuilder, error) {
	tm := NewTemplateManager()

	// Register the default template
	err := tm.RegisterTemplate("default_rag", defaultRAGTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to register default template: %v", err)
	}

	return &RAGMessageBuilder{
		templateManager: tm,
		config:          config,
	}, nil
}

// BuildSystemPrompt generates the system prompt with the provided documents
func (rb *RAGMessageBuilder) BuildSystemPrompt() (llm.Message, error) {
	systemPrompt, err := rb.templateManager.ExecuteTemplate("default_rag", rb.config)
	if err != nil {
		return llm.Message{}, fmt.Errorf("failed to build system prompt: %v", err)
	}

	return llm.Message{
		Role:    llm.SystemRole,
		Content: systemPrompt,
	}, nil
}

func GenerateRAGResponse(documents []Document, userQuery string, writer io.Writer) error {
	// Create configuration
	config := RAGPromptConfig{
		Documents:     documents,
		MaxTokens:     2048,
		Temperature:   0.2,
		ResponseStyle: "concise and factual",
		Query:         userQuery,
	}

	// Initialise RAG message builder
	builder, err := NewRAGMessageBuilder(config)
	if err != nil {
		return fmt.Errorf("failed to create RAG message builder: %v", err)
	}
	systemPrompt, err := builder.BuildSystemPrompt()
	messages := []llm.Message{
		systemPrompt,
	}

	// Create LLM instance
	llm, err := llm.NewLLM()
	if err != nil {
		return fmt.Errorf("failed to create LLM: %v", err)
	}

	// Generate response
	return llm.Generate(messages, writer)
}

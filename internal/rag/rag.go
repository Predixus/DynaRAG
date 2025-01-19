package rag

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/Predixus/DynaRAG/internal/llm"
)

// Document represents a single document with its metadata
type Document struct {
	Index   string `json:"index"`
	Source  string `json:"source"`
	Content string `json:"content"`
}

// RAGConfig holds the configuration for RAG system prompts
type RAGConfig struct {
	Documents     []Document // Changed from documents to Documents
	MaxTokens     int        // Changed from maxTokens to MaxTokens
	Temperature   float32    // Changed from temperature to Temperature
	ResponseStyle string     // Changed from responseStyle to ResponseStyle
	Query         string     // Changed from query to Query
}

// Option is a function type that modifies RAGConfig
type Option func(*RAGConfig)

// WithMaxTokens sets the maximum tokens for the RAG configuration
func WithMaxTokens(tokens int) Option {
	return func(c *RAGConfig) {
		c.MaxTokens = tokens
	}
}

// WithTemperature sets the temperature for the RAG configuration
func WithTemperature(temp float32) Option {
	return func(c *RAGConfig) {
		c.Temperature = temp
	}
}

// WithResponseStyle sets the response style for the RAG configuration
func WithResponseStyle(style string) Option {
	return func(c *RAGConfig) {
		c.ResponseStyle = style
	}
}

// defaultConfig returns a RAGConfig with default values
func defaultConfig() *RAGConfig {
	return &RAGConfig{
		MaxTokens:     2048,
		Temperature:   0.2,
		ResponseStyle: "concise and factual",
	}
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
func (tm *TemplateManager) ExecuteTemplate(name string, config RAGConfig) (string, error) {
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
	config          *RAGConfig
}

// NewRAGMessageBuilder creates a new RAG message builder
func NewRAGMessageBuilder(
	documents []Document,
	query string,
	opts ...Option,
) (*RAGMessageBuilder, error) {
	config := defaultConfig()
	config.Documents = documents
	config.Query = query

	for _, opt := range opts {
		opt(config)
	}

	tm := NewTemplateManager()
	if err := tm.RegisterTemplate("default_rag", defaultRAGTemplate); err != nil {
		return nil, fmt.Errorf("failed to register default template: %w", err)
	}

	return &RAGMessageBuilder{
		templateManager: tm,
		config:          config,
	}, nil
}

// BuildSystemPrompt generates the system prompt with the provided documents
func (rb *RAGMessageBuilder) BuildSystemPrompt() (llm.Message, error) {
	systemPrompt, err := rb.templateManager.ExecuteTemplate("default_rag", *rb.config)
	if err != nil {
		return llm.Message{}, fmt.Errorf("failed to build system prompt: %v", err)
	}

	return llm.Message{
		Role:    llm.RoleSystem,
		Content: systemPrompt,
	}, nil
}

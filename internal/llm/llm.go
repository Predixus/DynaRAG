package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Provider represents supported LLM providers
type Provider string

// Role represents the role of a message participant
type Role string

// Message represents a chat message
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Config holds the configuration for the LLM client
type Config struct {
	provider    Provider
	model       string
	token       string
	temperature float32
	endpoint    string
}

// Option is a function that modifies Config
type Option func(*Config)

// ResponseParser interface that all streaming responses must implement
type ResponseParser interface {
	GetContent() string
}

// GroqStreamingChatCompletion describes a chunk from a stream
type GroqStreamingChatCompletion struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []GroqChoice `json:"choices"`
}

// GroqChoice lower level chunk structure to the groq api response
type GroqChoice struct {
	Index        int     `json:"index"`
	Delta        Message `json:"delta"`
	LogProbs     *any    `json:"logprobs"`
	FinishReason string  `json:"finish_reason"`
}

// LLMModel with generic type parameter T that must implement ResponseParser
type LLMModel[T ResponseParser] struct {
	provider    Provider
	model       string
	token       string
	temperature float32
	endpoint    string
}

type LLM interface {
	Generate(messages []Message, writer io.Writer) error
}

const (
	// Provider constants
	ProviderGroq Provider = "groq"

	// Role constants
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"

	// Default values
	defaultTemperature  = 0.2
	defaultGroqModel    = "llama-3.3-70b-versatile"
	defaultGroqEndpoint = "https://api.groq.com/openai/v1/chat/completions"
)

// Option functions for configuration
func WithModel(model string) Option {
	return func(c *Config) {
		c.model = model
	}
}

func WithTemperature(temp float32) Option {
	return func(c *Config) {
		c.temperature = temp
	}
}

func WithEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.endpoint = endpoint
	}
}

// parseProvider validates and returns a Provider
func parseProvider(s string) (Provider, error) {
	provider := Provider(strings.ToLower(s))
	if !provider.isSupported() {
		return "", fmt.Errorf("invalid provider %q. Supported providers are: %v",
			s, []Provider{ProviderGroq})
	}
	return provider, nil
}

// isSupported checks if the provider is supported
func (p Provider) isSupported() bool {
	switch p {
	case ProviderGroq:
		return true
	default:
		return false
	}
}

func (g GroqStreamingChatCompletion) GetContent() string {
	if len(g.Choices) > 0 {
		return g.Choices[0].Delta.Content
	}
	return ""
}

func (r Role) isValid() bool {
	switch r {
	case RoleSystem, RoleUser, RoleAssistant:
		return true
	default:
		return false
	}
}

// NewClient creates a new LLM client with the given provider, token, and options
func NewClient(provider string, token string, opts ...Option) (LLM, error) {
	p, err := parseProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider: %w", err)
	}

	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	// Create default config based on provider
	config := &Config{
		provider:    p,
		token:       token,
		temperature: defaultTemperature,
	}

	// Set provider-specific defaults
	switch p {
	case ProviderGroq:
		config.model = defaultGroqModel
		config.endpoint = defaultGroqEndpoint
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Create appropriate client based on provider
	switch p {
	case ProviderGroq:
		return &LLMModel[GroqStreamingChatCompletion]{
			provider:    p,
			model:       config.model,
			token:       config.token,
			endpoint:    config.endpoint,
			temperature: config.temperature,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (l *LLMModel[T]) Generate(messages []Message, writer io.Writer) error {
	if len(messages) == 0 {
		return errors.New("Messages cannot be empty.")
	}

	// so far this seems common accross all the LLM providers.
	// if that changes, we will need to move the responsibility of
	// this payload to the generic
	payload := map[string]interface{}{
		"messages":    messages,
		"model":       l.model,
		"temperature": l.temperature,
		"stream":      true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		l.endpoint,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("groq request failed with status: %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %v", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		line = strings.TrimPrefix(line, "data: ")

		if line == "[DONE]" {
			break
		}

		var chunk T
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return fmt.Errorf("error parsing chunk: %v", err)
		}

		if content := chunk.GetContent(); content != "" {
			if _, err := writer.Write([]byte(content)); err != nil {
				return fmt.Errorf("error writing to output: %v", err)
			}
		}
	}

	return nil
}

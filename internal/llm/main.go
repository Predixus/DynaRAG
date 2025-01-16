package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// local types
type (
	LLMProvider string
	Role        string
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

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
	provider    LLMProvider
	model       string
	token       string
	temperature float32
	endpoint    string
}

type LLM interface {
	Generate(messages []Message, writer io.Writer) error
}

// local consts
const (
	GroqProvider  LLMProvider = "groq"
	SystemRole    Role        = "system"
	UserRole      Role        = "user"
	AssistantRole Role        = "assistant"
)

// functions

// setup Sets up the environment. Required environment variables:
// - LLM_PROVIDER - the LLM provider, either groq or ...
// - LLM_TOKEN - token to the provider
func setup(envFile string) (LLMProvider, string) {
	if envFile != "" {
		godotenv.Load(envFile)
	}

	llmProvider := os.Getenv("LLM_PROVIDER")
	if llmProvider == "" {
		panic("`LLM_PROVIDER` not provided")
	}

	llmToken := os.Getenv("LLM_TOKEN")
	if llmToken == "" {
		panic("`LLM_TOKEN` not provided")
	}

	provider, err := ParseLLMProvider(llmProvider)
	if err != nil {
		panic("Could not parse requested provider")
	}

	return provider, llmToken
}

// ParseLLMProvider handle the requested LLM provider, if not supported raise an error
func ParseLLMProvider(s string) (LLMProvider, error) {
	provider := LLMProvider(strings.ToLower(s))
	if !provider.isSupported() {
		return "", fmt.Errorf("invalid provider %q. Supported providers are: %v",
			s, []LLMProvider{GroqProvider})
	}
	return provider, nil
}

// isSupported Contains all supported LLM providers
func (p LLMProvider) isSupported() bool {
	switch p {
	case GroqProvider:
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
	case SystemRole, UserRole, AssistantRole:
		return true
	default:
		return false
	}
}

// NewLLM returns a specific LLMModel with the appropriate response format type
func NewLLM(provider LLMProvider, token string) (LLM, error) {
	switch provider {
	case GroqProvider:
		return &LLMModel[GroqStreamingChatCompletion]{
			model:       "llama-3.3-70b-versatile",
			provider:    GroqProvider,
			token:       token,
			endpoint:    "https://api.groq.com/openai/v1/chat/completions",
			temperature: 0.2,
		}, nil
	default:
		return nil, fmt.Errorf("Unsupported provider: %s", provider)
	}
}

func NewLLMFromEnv() (LLM, error) {
	provider, token := setup("../.env")
	return NewLLM(provider, token)
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

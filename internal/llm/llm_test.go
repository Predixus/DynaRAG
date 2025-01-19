package llm

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		token       string
		opts        []Option
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid groq config",
			provider: "groq",
			token:    "test-token",
			opts:     []Option{WithTemperature(0.5)},
			wantErr:  false,
		},
		{
			name:        "empty token",
			provider:    "groq",
			token:       "",
			wantErr:     true,
			errContains: "token cannot be empty",
		},
		{
			name:        "invalid provider",
			provider:    "invalid",
			token:       "test-token",
			wantErr:     true,
			errContains: "invalid provider",
		},
		{
			name:     "with custom options",
			provider: "groq",
			token:    "test-token",
			opts: []Option{
				WithModel("custom-model"),
				WithTemperature(0.7),
				WithEndpoint("https://custom.endpoint"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.provider, tt.token, tt.opts...)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, client)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}

func TestGenerate(t *testing.T) {
	// Skip integration tests if no token is provided
	token := os.Getenv("GROQ_API_TOKEN")
	if token == "" {
		t.Skip("GROQ_API_TOKEN not set")
	}

	tests := []struct {
		name     string
		messages []Message
		wantErr  bool
	}{
		{
			name: "valid message",
			messages: []Message{
				{
					Role:    RoleSystem,
					Content: "This is a test. Only ever return the string, 'testing, testing, 123'",
				},
			},
			wantErr: false,
		},
		{
			name:     "empty messages",
			messages: []Message{},
			wantErr:  true,
		},
		{
			name: "invalid role",
			messages: []Message{
				{
					Role:    Role("invalid"),
					Content: "test",
				},
			},
			wantErr: true,
		},
	}

	client, err := NewClient("groq", token)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := client.Generate(tt.messages, &buf)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, buf.String())
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, buf.String())
		})
	}
}

func TestConfigOptions(t *testing.T) {
	config := &Config{
		provider:    ProviderGroq,
		token:       "test-token",
		temperature: defaultTemperature,
		model:       defaultGroqModel,
		endpoint:    defaultGroqEndpoint,
	}

	tests := []struct {
		name   string
		option Option
		check  func(*testing.T, *Config)
	}{
		{
			name:   "WithModel",
			option: WithModel("custom-model"),
			check: func(t *testing.T, c *Config) {
				assert.Equal(t, "custom-model", c.model)
			},
		},
		{
			name:   "WithTemperature",
			option: WithTemperature(0.7),
			check: func(t *testing.T, c *Config) {
				assert.Equal(t, float32(0.7), c.temperature)
			},
		},
		{
			name:   "WithEndpoint",
			option: WithEndpoint("https://custom.endpoint"),
			check: func(t *testing.T, c *Config) {
				assert.Equal(t, "https://custom.endpoint", c.endpoint)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.option(config)
			tt.check(t, config)
		})
	}
}

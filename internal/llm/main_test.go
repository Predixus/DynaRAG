package llm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroqProvider(t *testing.T) {
	llm, err := NewLLMFromEnv()
	assert.NoError(t, err)

	messages := []Message{
		{
			Role:    SystemRole,
			Content: "This is a test. Only ever return the string, 'testing, testing, 123'",
		},
	}

	var buf bytes.Buffer
	err = llm.Generate(messages, &buf)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestGroqNoMessageGivesEmptyResponse(t *testing.T) {
	llm, err := NewLLMFromEnv()
	assert.NoError(t, err)

	messages := []Message{}
	var buf bytes.Buffer
	err = llm.Generate(messages, &buf)
	assert.Error(t, err, "Error returned")
	assert.Empty(t, buf.String(), "Buffer should be empty")
}

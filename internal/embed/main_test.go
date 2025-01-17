package embed

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected EmbedderConfig
	}{
		{
			name: "with custom model dir",
			opts: []Option{
				WithModelDir("/custom/dir"),
			},
			expected: EmbedderConfig{
				ModelDir:  "/custom/dir",
				ModelName: "optimum/all-MiniLM-L6-v2",
			},
		},
		{
			name: "with custom model name",
			opts: []Option{
				WithModelName("custom-model"),
			},
			expected: EmbedderConfig{
				ModelDir:  "../../models",
				ModelName: "custom-model",
			},
		},
		{
			name: "with both options",
			opts: []Option{
				WithModelDir("/custom/dir"),
				WithModelName("custom-model"),
			},
			expected: EmbedderConfig{
				ModelDir:  "/custom/dir",
				ModelName: "custom-model",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			for _, opt := range tt.opts {
				opt(&cfg)
			}
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestEmbedder(t *testing.T) {
	// Create a temporary directory for test models
	tmpDir, err := os.MkdirTemp("./", "embedder-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("initialisation", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping model initialisation test")
		}
		embedder, err := GetEmbedder(WithModelDir(tmpDir))
		require.NoError(t, err)
		require.NotNil(t, embedder)
		defer embedder.Close()

		// Verify model was downloaded
		files, err := filepath.Glob(filepath.Join(tmpDir, "*", "*.onnx"))
		require.NoError(t, err)
		assert.NotEmpty(t, files)
	})

	t.Run("singleton pattern", func(t *testing.T) {
		first, err := GetEmbedder()
		require.NoError(t, err)
		defer first.Close()

		second, err := GetEmbedder()
		require.NoError(t, err)

		assert.Same(t, first, second, "GetEmbedder should return the same instance")
	})

	t.Run("embedding generation", func(t *testing.T) {
		embedder, err := GetEmbedder()
		require.NoError(t, err)
		defer embedder.Close()

		tests := []struct {
			name     string
			input    []string
			wantErr  bool
			validate func(t *testing.T, embeddings [][]float32)
		}{
			{
				name:    "single text",
				input:   []string{"Hello, world!"},
				wantErr: false,
				validate: func(t *testing.T, embeddings [][]float32) {
					assert.Len(t, embeddings, 1)
					assert.NotEmpty(t, embeddings[0])
					// MiniLM-L6-v2 produces 384-dimensional vectors
					assert.Len(t, embeddings[0], 384)
				},
			},
			// {
			// 	name:    "multiple texts",
			// 	input:   []string{"Hello, world!", "This is a test"},
			// 	wantErr: false,
			// 	validate: func(t *testing.T, embeddings [][]float32) {
			// 		assert.Len(t, embeddings, 2)
			// 		assert.NotEmpty(t, embeddings[0])
			// 		assert.NotEmpty(t, embeddings[1])
			// 		assert.Len(t, embeddings[0], 384)
			// 		assert.Len(t, embeddings[1], 384)
			// 	},
			// },
			// {
			// 	name:    "empty input",
			// 	input:   []string{},
			// 	wantErr: true,
			// 	validate: func(t *testing.T, embeddings [][]float32) {
			// 		assert.Nil(t, embeddings)
			// 	},
			// },
			// {
			// 	name:    "nil input",
			// 	input:   nil,
			// 	wantErr: true,
			// 	validate: func(t *testing.T, embeddings [][]float32) {
			// 		assert.Nil(t, embeddings)
			// 	},
			// },
		}

		for _, tt := range tests {
			embeddings, err := embedder.GetEmbeddings(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			tt.validate(t, embeddings)
		}
	})
	//
	// t.Run("embedding consistency", func(t *testing.T) {
	// 	embedder, err := GetEmbedder()
	// 	require.NoError(t, err)
	// 	defer embedder.Close()
	//
	// 	// Same text should produce same embedding
	// 	text := "Hello, world!"
	// 	embedding1, err := embedder.GetEmbeddings([]string{text})
	// 	require.NoError(t, err)
	//
	// 	embedding2, err := embedder.GetEmbeddings([]string{text})
	// 	require.NoError(t, err)
	//
	// 	assert.Equal(t, embedding1, embedding2)
	//
	// 	// Different texts should produce different embeddings
	// 	differentText := "Different text"
	// 	embedding3, err := embedder.GetEmbeddings([]string{differentText})
	// 	require.NoError(t, err)
	//
	// 	assert.NotEqual(t, embedding1[0], embedding3[0])
	// })
	//
	// t.Run("concurrent access", func(t *testing.T) {
	// 	embedder, err := GetEmbedder()
	// 	require.NoError(t, err)
	// 	defer embedder.Close()
	//
	// 	done := make(chan bool)
	// 	for i := 0; i < 10; i++ {
	// 		go func() {
	// 			_, err := embedder.GetEmbeddings([]string{"concurrent test"})
	// 			assert.NoError(t, err)
	// 			done <- true
	// 		}()
	// 	}
	//
	// 	// Wait for all goroutines to complete
	// 	for i := 0; i < 10; i++ {
	// 		<-done
	// 	}
	// })
}

//	func TestEmbedderErrors(t *testing.T) {
//		t.Run("invalid model directory", func(t *testing.T) {
//			_, err := GetEmbedder(
//				WithModelDir("/nonexistent/directory"),
//				WithModelName("invalid-model"),
//			)
//			assert.Error(t, err)
//		})
//
//		t.Run("cleanup", func(t *testing.T) {
//			embedder, err := GetEmbedder()
//			require.NoError(t, err)
//
//			// Test double close
//			assert.NoError(t, embedder.Close())
//			assert.NoError(t, embedder.Close())
//
//			// Verify operations after close fail
//			_, err = embedder.GetEmbeddings([]string{"test"})
//			assert.Error(t, err)
//		})
//	}
//

// // TestMain handles setup and teardown for all tests
// func TestMain(m *testing.M) {
// 	// Setup code if needed
//
// 	// Run tests
// 	code := m.Run()
//
// 	// Cleanup code if needed
//
// 	os.Exit(code)
// }

package embed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

// EmbedderConfig holds the configuration for the Embedder
type EmbedderConfig struct {
	ModelDir  string
	ModelName string
}

// Option is a functional option for configuring the Embedder
type Option func(*EmbedderConfig)

// Embedder handles text embedding operations
type Embedder struct {
	modelPath string
	pipeline  *pipelines.FeatureExtractionPipeline
	session   *hugot.Session
	mu        sync.RWMutex
}

// DefaultConfig returns the default configuration
func DefaultConfig() EmbedderConfig {
	return EmbedderConfig{
		ModelDir:  "../../models",
		ModelName: "optimum/all-MiniLM-L6-v2",
	}
}

var (
	singleton *Embedder
	once      sync.Once
)

// Configurations - functional option config pattern

// WithModelDir sets the model directory
func WithModelDir(dir string) Option {
	return func(c *EmbedderConfig) {
		c.ModelDir = dir
	}
}

// WithModelName sets the model name
func WithModelName(name string) Option {
	return func(c *EmbedderConfig) {
		c.ModelName = name
	}
}

// GetEmbedder returns a singleton instance of Embedder with the given options
func GetEmbedder(opts ...Option) (*Embedder, error) {
	var initError error
	once.Do(func() {
		singleton, initError = newEmbedder(opts...)
	})
	return singleton, initError
}

func newEmbedder(opts ...Option) (*Embedder, error) {
	// Apply configuration options
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}

	session, err := hugot.NewORTSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create ORT session: %w", err)
	}

	modelPath := filepath.Join(config.ModelDir, strings.Replace(config.ModelName, "/", "_", 1))
	if err := os.MkdirAll(modelPath, 0775); err != nil {
		return nil, fmt.Errorf("failed to create model directory: %w", err)
	}

	// Check if model exists, download if it doesn't
	files, err := filepath.Glob(filepath.Join(modelPath, "*.onnx"))
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing model: %w", err)
	}

	if len(files) == 0 {
		modelPath, err = hugot.DownloadModel(
			config.ModelName,
			config.ModelDir,
			hugot.NewDownloadOptions(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to download model: %w", err)
		}
	}

	pipelineConfig := hugot.FeatureExtractionConfig{
		ModelPath: modelPath,
		Name:      "embedder",
	}

	pipeline, err := hugot.NewPipeline(session, pipelineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	return &Embedder{
		modelPath: modelPath,
		pipeline:  pipeline,
		session:   session,
	}, nil
}

func (e *Embedder) GetEmbeddings(texts []string) ([][]float32, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result, err := e.pipeline.RunPipeline(texts)
	if err != nil {
		return nil, err
	}

	return result.Embeddings, nil
}

func (e *Embedder) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.session != nil {
		return e.session.Destroy()
	}
	return nil
}

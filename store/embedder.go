package store

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

type Embedder struct {
	modelPath string
	pipeline  *pipelines.FeatureExtractionPipeline
	session   *hugot.Session
	mu        sync.RWMutex
}

var (
	singleton *Embedder
	once      sync.Once
)

func GetEmbedder() (*Embedder, error) {
	var initError error
	once.Do(func() {
		singleton, initError = newEmbedder()
	})
	return singleton, initError
}

func newEmbedder() (*Embedder, error) {
	session, err := hugot.NewORTSession()
	if err != nil {
		return nil, err
	}

	modelDir := "./models"
	model := "optimum/all-MiniLM-L6-v2"

	// TODO manage more models
	modelPath := filepath.Join(modelDir, strings.Replace(model, "/", "_", 1))
	err = os.MkdirAll(modelPath, 0775)
	if err != nil {
		panic(err)
	}

	// check if model exists, download if it doesn't
	files, err := filepath.Glob(filepath.Join(modelPath, "*.onnx"))
	if err != nil || len(files) == 0 {
		modelPath, err = hugot.DownloadModel(
			model,
			modelDir,
			hugot.NewDownloadOptions(),
		)
		if err != nil {
			return nil, err
		}
	}

	config := hugot.FeatureExtractionConfig{
		ModelPath: modelPath,
		Name:      "embedder",
	}

	pipeline, err := hugot.NewPipeline(session, config)
	if err != nil {
		return nil, err
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

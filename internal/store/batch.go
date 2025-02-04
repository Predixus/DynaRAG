package store

import (
	"context"
	"log"
	"sync"
)

type Chunk struct {
	FilePath      string
	ChunkText     string
	EmbeddingText *string
	Metadata      map[string]interface{}
}

type BatchProgress struct {
	Total     int
	Completed int
	Failed    []string
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func InitializeBatchProgress(total int) *BatchProgress {
	ctx, cancel := context.WithCancel(context.Background())
	return &BatchProgress{
		Total:  total,
		Failed: []string{},
		ctx:    ctx,
		cancel: cancel,
	}
}

// using userId as a dummy for now

func BatchProgressEmbeddings(ctx context.Context, userId string, chunks []Chunk) *BatchProgress {
	progress := InitializeBatchProgress(len(chunks))
	var wg sync.WaitGroup

	for _, chunk := range chunks {
		select {
		case <-progress.ctx.Done():
			log.Println("Batch process canceled")
			return progress
		default:
			wg.Add(1)
			go func(chunk Chunk) {
				defer wg.Done()
				if err := processSingleChunk(ctx, userId, chunk, progress); err != nil {
					progress.mu.Lock()
					progress.Failed = append(progress.Failed, chunk.FilePath)
					progress.mu.Unlock()
				}
			}(chunk)
		}
	}

	wg.Wait()
	return progress
}

func processSingleChunk(
	ctx context.Context,
	userId string,
	chunk Chunk,
	progress *BatchProgress,
) error {
	_, err := AddEmbedding(
		ctx,
		userId,
		chunk.FilePath,
		chunk.ChunkText,
		chunk.EmbeddingText,
		chunk.Metadata,
	)
	progress.mu.Lock()
	defer progress.mu.Unlock()
	if err != nil {
		log.Printf("Failed to process chunk for file: %s, error: %v", chunk.FilePath, err)
		return err
	}
	progress.Completed++
	return nil
}

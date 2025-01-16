package dynarag

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/Predixus/DynaRAG/internal/rag"
	"github.com/Predixus/DynaRAG/internal/store"
	"github.com/Predixus/DynaRAG/types"
)

//go:generate sqlc vet
//go:generate sqlc generate

var once sync.Once

func (c *Client) Chunk(
	ctx context.Context,
	chunk string,
	filePath string,
	metadata *types.JSONMap,
) error {
	_, err := store.AddEmbedding(ctx, c.config.PostgresConnStr, filePath, chunk, metadata)
	if err != nil {
		slog.Error("Could not process embedding", "error", err)
		return err
	}
	return nil
}

func (c *Client) Similar(
	ctx context.Context,
	text string,
	k int8,
	metadata *types.JSONMap,
) ([]store.FindTopKNNEmbeddingsRow, error) {
	slog.Info("Gathering similar documents")

	res, err := store.GetTopKEmbeddings(ctx, c.config.PostgresConnStr, text, k, metadata)
	if err != nil {
		slog.Error("Could not get top K embeddings", "error", err)
		return nil, err
	}
	return res, nil
}

func (c *Client) Query(
	ctx context.Context,
	query string,
	k *int8,
	metadata *types.JSONMap,
	writer io.Writer,
) error {
	slog.Info("Gathering similar documents")

	topN := int8(10) // Default number of chunks to factor into the response
	if k != nil {
		topN = *k
	}

	res, err := store.GetTopKEmbeddings(ctx, c.config.PostgresConnStr, query, topN, metadata)
	if err != nil {
		slog.Error("Could not get top K embeddings", "error", err)
		return err
	}

	var documents []rag.Document

	for ii, doc := range res {
		documents = append(documents, rag.Document{
			Index:   strconv.Itoa(ii),
			Source:  doc.FilePath,
			Content: doc.ChunkText,
		})
	}

	err = rag.GenerateRAGResponse(documents, query, writer)
	if err != nil {
		slog.Error("Failed to generate RAG response", "error", err)
		return err
	}
	return nil
}

func (c *Client) PurgeChunks(ctx context.Context, dryRun *bool) (*store.DeletionStats, error) {
	doDryRun := false

	if dryRun != nil {
		doDryRun = *dryRun
	}

	stats, err := store.DeleteUserEmbeddings(ctx, c.config.PostgresConnStr, doDryRun)
	if err != nil {
		slog.Error("Failed to delete embeddings", "error", err)
		return nil, err
	}
	return stats, nil
}

func (c *Client) GetStats(
	ctx context.Context,
) (*store.GetStatsRow, error) {
	stats, err := store.GetStats(ctx, c.config.PostgresConnStr)
	if err != nil {
		slog.Error("Failed to get user stats", "error", err)
		return nil, err
	}
	return stats, nil
}

func (c *Client) ListChunks(
	ctx context.Context,
	metadata *types.JSONMap,
) ([]store.ListChunksRow, error) {
	chunks, err := store.ListUserChunks(ctx, c.config.PostgresConnStr, metadata)
	if err != nil {

		slog.Error("Failed to list user chunks: %v", err)
		noChunks := make([]store.ListChunksRow, 0, 0)

		return noChunks, err
	}
	return chunks, nil
}

type Config struct {
	PostgresConnStr string
}

type Client struct {
	config Config
}

func New(cfg Config) (*Client, error) {
	if cfg.PostgresConnStr == "" {
		return nil, ErrMissingConnStr
	}

	client := &Client{
		config: cfg,
	}

	if err := client.Initialise(); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return client, nil
}

// Initialise migrations and other necessary infrastructure for DynaRAG
func (c *Client) Initialise() error {
	return initMigrations(c.config.PostgresConnStr)
}

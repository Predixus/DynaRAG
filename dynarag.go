package dynarag

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/Predixus/DynaRAG/rag"
	"github.com/Predixus/DynaRAG/store"
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
	_, err := store.AddEmbedding(ctx, filePath, chunk, metadata)
	if err != nil {
		slog.Error("Could not process embedding: %v", err)
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

	res, err := store.GetTopKEmbeddings(ctx, text, k, metadata)
	if err != nil {
		slog.Error("Could not get top K embeddings: %v", err)
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

	res, err := store.GetTopKEmbeddings(ctx, query, topN, metadata)
	if err != nil {
		slog.Error("Could not get top K embeddings: %v", err)
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
		slog.Error("Failed to generate RAG response: ", err)
		return err
	}
	return nil
}

func (c *Client) PurgeChunks(ctx context.Context, dryRun *bool) (*store.DeletionStats, error) {
	doDryRun := false

	if dryRun != nil {
		doDryRun = *dryRun
	}

	stats, err := store.DeleteUserEmbeddings(ctx, doDryRun)
	if err != nil {
		slog.Error("Failed to delete embeddings: %v", err)
		return nil, err
	}
	return stats, nil
}

func (c *Client) GetStats(
	ctx context.Context,
) (*store.GetUserStatsRow, error) {
	stats, err := store.GetStats(ctx)
	if err != nil {
		slog.Error("Failed to get user stats: %v", err)
		return nil, err
	}
	return stats, nil
}

func (c *Client) ListChunks(
	ctx context.Context,
	metadata *types.JSONMap,
) ([]store.ListUserChunksRow, error) {
	chunks, err := store.ListUserChunks(ctx, metadata)
	if err != nil {

		slog.Error("Failed to list user chunks: %v", err)
		noChunks := make([]store.ListUserChunksRow, 0, 0)

		return noChunks, err
	}
	return chunks, nil
}

type Config struct {
	PostgresConnStr string
	MigrationsPath  string
}

type Client struct {
	config Config
}

func New(cfg Config) (*Client, error) {
	if cfg.PostgresConnStr == "" {
		return nil, ErrMissingConnStr
	}

	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "file://store/migrations"
	}

	client := &Client{
		config: cfg,
	}

	if err := client.Initialise(); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return client, nil
}

// Initiliase migrations and other neccessary infrastructure for DynaRAG
func (c *Client) Initialise() error {
	// run migrations
	m, err := migrate.New(
		c.config.MigrationsPath,
		c.config.PostgresConnStr,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

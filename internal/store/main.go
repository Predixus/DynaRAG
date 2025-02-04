package store

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"

	"github.com/Predixus/DynaRAG/internal/embed"
	"github.com/Predixus/DynaRAG/internal/utils"
	"github.com/Predixus/DynaRAG/types"
)

var (
	once_v2  sync.Once
	embedder *embed.Embedder
)

func init() {
	var err error
	embedder, err = embed.NewEmbedder()
	if err != nil {
		slog.Error("Failed to initialise embedder", "error", err)
		return
	}
}

func GetSingleEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := embedder.GetEmbeddings([]string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

func AddEmbedding(
	ctx context.Context,
	postgresConnStr string,
	filePath string,
	chunkText string,
	embeddingText *string, // using nil for default behavior
	metadata *types.JSONMap,
) (*Embedding, error) {
	textToEmbed := chunkText
	if embeddingText != nil {
		textToEmbed = *embeddingText
	}
	embedding, err := GetSingleEmbedding(ctx, textToEmbed)
	if err != nil {
		return nil, err
	}

	conn, err := pgx.Connect(ctx, postgresConnStr)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := New(tx)

	doc, err := q.CreateDocument(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// calculate hash
	var metadataHash string = ""
	if metadata != nil {
		metadataJsonBytes, err := json.Marshal(metadata)
		if err != nil {
			slog.Error("Error marshalling metadata", "error", err)
			return nil, err
		}

		metadataHash, err = utils.CalculateMetadataHash(metadataJsonBytes)
		if err != nil {
			slog.Error("Error calculating hash on Metadata", "error", err)
			return nil, err
		}
	} else {
		metadataRaw := types.JSONMap(make(map[string]interface{}))
		metadata = &metadataRaw
	}

	embeddingRecord, err := q.CreateEmbedding(ctx, CreateEmbeddingParams{
		DocumentID:   pgtype.Int8{Int64: doc.ID, Valid: true},
		ModelName:    "all-MiniLM-L6-v2",
		ChunkText:    chunkText,
		Embedding:    pgvector.NewVector(embedding),
		Metadata:     *metadata,
		MetadataHash: pgtype.Text{String: metadataHash, Valid: true},
		EmbeddingText: pgtype.Text{
			String: func() string {
				if embeddingText != nil {
					return *embeddingText
				}
				return ""
			}(),
			Valid: embeddingText != nil,
		},
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &embeddingRecord, nil
}

func GetTopKEmbeddings(
	ctx context.Context,
	postgresConnStr string,
	text string,
	k int8,
	metadata *types.JSONMap,
) ([]FindTopKNNEmbeddingsRow, error) {
	embedding, err := GetSingleEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	conn, err := pgx.Connect(ctx, postgresConnStr)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	q := New(conn)

	// calculate metadatahash
	var metadataHashPtr *string
	if metadata != nil {
		jsonMetadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		metadataHash, err := utils.CalculateMetadataHash(jsonMetadataBytes)
		metadataHashPtr = &metadataHash
		if err != nil {
			return nil, err
		}
	}

	return q.FindTopKNNEmbeddings(ctx, FindTopKNNEmbeddingsParams{
		QueryEmbedding: pgvector.NewVector(embedding),
		ModelName:      "all-MiniLM-L6-v2",
		K:              int32(k),
		MetadataHash: pgtype.Text{
			Valid: metadataHashPtr != nil,
			String: func() string {
				if metadataHashPtr != nil {
					return *metadataHashPtr
				}
				return ""
			}(),
		},
	})
}

// DeletionStats provides information about what would be/was deleted
type DeletionStats struct {
	EmbeddingCount int64    // Number of embeddings that would be deleted
	DocumentCount  int64    // Number of documents that would be affected
	TotalBytes     int64    // Total storage space that would be freed
	FilePaths      []string // List of file paths that would be affected
}

// DeleteUserEmbeddings deletes all embeddings and documents for a given user
// If dryRun is true, returns what would be deleted without actually deleting
func DeleteUserEmbeddings(
	ctx context.Context,
	postgresConnStr string,
	dryRun bool,
) (*DeletionStats, error) {
	conn, err := pgx.Connect(ctx, postgresConnStr)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := New(tx)

	// Get statistics first
	stats, err := q.GetStorageStats(ctx)
	if err != nil {
		return nil, err
	}

	// Get affected file paths
	docs, err := q.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}

	filePaths := make([]string, len(docs))
	for i, doc := range docs {
		filePaths[i] = doc.FilePath
	}

	deletionStats := &DeletionStats{
		EmbeddingCount: stats.EmbeddingCount,
		DocumentCount:  stats.DocumentCount,
		TotalBytes:     stats.TotalBytes,
		FilePaths:      filePaths,
	}

	// If this is just a dry run, return the stats without deleting
	if dryRun {
		return deletionStats, nil
	}

	// Actually perform the deletion
	err = q.DeleteEmbeddings(ctx)
	if err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return deletionStats, nil
}

func GetStats(ctx context.Context, postgresConnStr string) (*GetStatsRow, error) {
	conn, err := pgx.Connect(ctx, postgresConnStr)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	q := New(conn)

	stats, err := q.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func ListUserChunks(
	ctx context.Context,
	postgresConnStr string,
	metadata *types.JSONMap,
) ([]ListChunksRow, error) {
	conn, err := pgx.Connect(ctx, postgresConnStr)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	q := New(conn)

	// calculate metadatahash
	var metadataHashPtr *string
	if metadata != nil {
		jsonMetadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		metadataHash, err := utils.CalculateMetadataHash(jsonMetadataBytes)
		metadataHashPtr = &metadataHash
		if err != nil {
			return nil, err
		}
	}
	// Get all chunks for the user
	chunks, err := q.ListChunks(ctx, pgtype.Text{String: func() string {
		if metadataHashPtr != nil {
			return *metadataHashPtr
		}
		return ""
	}(), Valid: metadataHashPtr != nil},
	)
	if err != nil {
		slog.Error("Error when listing user chunks", "error", err)
		return nil, err
	}

	return chunks, nil
}

package store

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"github.com/pgvector/pgvector-go"

	"github.com/Predixus/DynaRAG/embed"
	"github.com/Predixus/DynaRAG/types"
	"github.com/Predixus/DynaRAG/utils"
)

var (
	once_v2           sync.Once
	postgres_conn_str string
	embedder          *embed.Embedder
)

func setup() string {
	postgres_conn_str := os.Getenv("POSTGRES_CONN_STR")
	if postgres_conn_str == "" {
		panic("`POSTGRES_CONN_STR` not set")
	}
	return postgres_conn_str
}

func init() {
	godotenv.Load()
	postgres_conn_str = setup()
	var err error
	embedder, err = embed.GetEmbedder()
	if err != nil {
		log.Fatalf("Failed to initialize embedder: %v", err)
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
	filePath string,
	text string,
	metadata map[string]interface{},
) (*Embedding, error) {
	embedding, err := GetSingleEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	conn, err := pgx.Connect(ctx, postgres_conn_str)
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

	user, err := q.CreateUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	doc, err := q.CreateDocument(ctx, CreateDocumentParams{
		UserID: pgtype.Int8{
			Int64: user.ID,
			Valid: true,
		},
		FilePath: filePath,
	})
	if err != nil {
		return nil, err
	}

	// calculate hash
	var metadataHash string = ""
	if metadata != nil {
		metadataJsonBytes, err := json.Marshal(metadata)
		if err != nil {
			log.Println("Error marshalling metadata: ", err)
			return nil, err
		}

		metadataHash, err = utils.CalculateMetadataHash(metadataJsonBytes)
		if err != nil {
			log.Println("Error calculating hash on Metadata: ", err)
			return nil, err
		}
	} else {
		metadata = make(map[string]interface{})
	}

	embeddingRecord, err := q.CreateEmbedding(ctx, CreateEmbeddingParams{
		DocumentID: pgtype.Int8{
			Int64: doc.ID,
			Valid: true,
		},
		ModelName: "all-MiniLM-L6-v2",
		ChunkText: text,
		Embedding: pgvector.NewVector(embedding),
		Metadata:  metadata,
		MetadataHash: pgtype.Text{
			String: metadataHash,
			Valid:  true,
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
	text string,
	k int8,
	metadata *types.JSONMap,
) ([]FindTopKNNEmbeddingsRow, error) {
	embedding, err := GetSingleEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	conn, err := pgx.Connect(ctx, postgres_conn_str)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	q := New(conn)

	user, err := q.CreateUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	// calculate metadatahash
	var metadataHashPtr *string
	if metadata != nil {
		jsonMetadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		metadataHash, err := utils.CalculateMetadataHash(jsonMetadataBytes)
		log.Println("MetadataHash: ", metadataHash)
		metadataHashPtr = &metadataHash
		if err != nil {
			return nil, err
		}
	}

	return q.FindTopKNNEmbeddings(ctx, FindTopKNNEmbeddingsParams{
		QueryEmbedding: pgvector.NewVector(embedding),
		ModelName:      "all-MiniLM-L6-v2",
		UserID: pgtype.Int8{
			Int64: user.ID,
			Valid: true,
		},
		K: int32(k),
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
func DeleteUserEmbeddings(ctx context.Context, dryRun bool) (*DeletionStats, error) {
	conn, err := pgx.Connect(ctx, postgres_conn_str)
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

	// Get user ID first
	user, err := q.CreateUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	// Get statistics first
	stats, err := q.GetUserStorageStats(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// Get affected file paths
	docs, err := q.ListUserDocuments(ctx, pgtype.Int8{
		Int64: user.ID,
		Valid: true,
	})
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
		TotalBytes:     stats.TotalBytes.Int64,
		FilePaths:      filePaths,
	}

	// If this is just a dry run, return the stats without deleting
	if dryRun {
		return deletionStats, nil
	}

	// Actually perform the deletion
	err = q.DeleteUserEmbeddings(ctx, pgtype.Int8{
		Int64: user.ID,
		Valid: true,
	})
	if err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return deletionStats, nil
}

func GetStats(ctx context.Context) (*GetUserStatsRow, error) {
	conn, err := pgx.Connect(ctx, postgres_conn_str)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	q := New(conn)

	user, err := q.CreateUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	stats, err := q.GetUserStats(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func ListUserChunks(
	ctx context.Context,
	metadata *types.JSONMap,
) ([]ListUserChunksRow, error) {
	conn, err := pgx.Connect(ctx, postgres_conn_str)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	q := New(conn)

	// Get user ID first
	user, err := q.CreateUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	// calculate metadatahash
	var metadataHashPtr *string
	if metadata != nil {
		jsonMetadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		metadataHash, err := utils.CalculateMetadataHash(jsonMetadataBytes)
		log.Println("MetadataHash: ", metadataHash)
		metadataHashPtr = &metadataHash
		if err != nil {
			return nil, err
		}
	}
	// Get all chunks for the user
	chunks, err := q.ListUserChunks(ctx, ListUserChunksParams{
		UserID: pgtype.Int8{
			Int64: user.ID,
			Valid: true,
		},
		MetadataHash: pgtype.Text{String: func() string {
			if metadataHashPtr != nil {
				return *metadataHashPtr
			}
			return ""
		}(), Valid: metadataHashPtr != nil},
	})
	if err != nil {
		log.Println("Error when listing user chunks")
		return nil, err
	}

	return chunks, nil
}

func IncrementAPIUsage(ctx context.Context, userId string) (*ApiUsage, error) {
	conn, err := pgx.Connect(ctx, postgres_conn_str)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	q := New(conn)

	user, err := q.CreateUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	usage, err := q.IncrementAPIUsage(ctx, pgtype.Int8{
		Int64: user.ID,
		Valid: true,
	})
	if err != nil {
		return nil, err
	}

	return &usage, nil
}

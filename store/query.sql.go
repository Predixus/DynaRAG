// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: query.sql

package store

import (
	"context"

	"github.com/Predixus/DynaRAG/types"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

const createDocument = `-- name: CreateDocument :one
INSERT INTO documents (user_id, file_path)
VALUES ($1, $2)
ON CONFLICT (user_id, file_path) DO UPDATE 
SET updated_at = CURRENT_TIMESTAMP
RETURNING id, user_id, file_path, total_chunk_size, created_at, updated_at
`

type CreateDocumentParams struct {
	UserID   pgtype.Int8
	FilePath string
}

func (q *Queries) CreateDocument(ctx context.Context, arg CreateDocumentParams) (Document, error) {
	row := q.db.QueryRow(ctx, createDocument, arg.UserID, arg.FilePath)
	var i Document
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.FilePath,
		&i.TotalChunkSize,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createEmbedding = `-- name: CreateEmbedding :one
INSERT INTO embeddings (
    document_id,
    model_name,
    chunk_text, 
    embedding,
    chunk_size, 
    metadata, 
    metadata_hash
) VALUES (
    $1, $2, $3, $4, length($3), $5, $6
)
RETURNING id, document_id, model_name, embedding, chunk_text, chunk_size, created_at, metadata, metadata_hash
`

type CreateEmbeddingParams struct {
	DocumentID   pgtype.Int8
	ModelName    EmbeddingModel
	ChunkText    string
	Embedding    pgvector.Vector
	Metadata     types.JSONMap
	MetadataHash pgtype.Text
}

func (q *Queries) CreateEmbedding(ctx context.Context, arg CreateEmbeddingParams) (Embedding, error) {
	row := q.db.QueryRow(ctx, createEmbedding,
		arg.DocumentID,
		arg.ModelName,
		arg.ChunkText,
		arg.Embedding,
		arg.Metadata,
		arg.MetadataHash,
	)
	var i Embedding
	err := row.Scan(
		&i.ID,
		&i.DocumentID,
		&i.ModelName,
		&i.Embedding,
		&i.ChunkText,
		&i.ChunkSize,
		&i.CreatedAt,
		&i.Metadata,
		&i.MetadataHash,
	)
	return i, err
}

const createUser = `-- name: CreateUser :one
INSERT INTO users (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE 
SET updated_at = CURRENT_TIMESTAMP
RETURNING id, user_id, total_chunk_size, created_at, updated_at
`

func (q *Queries) CreateUser(ctx context.Context, userID string) (User, error) {
	row := q.db.QueryRow(ctx, createUser, userID)
	var i User
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.TotalChunkSize,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteDocument = `-- name: DeleteDocument :exec
DELETE FROM documents
WHERE id = $1 AND user_id = $2
`

type DeleteDocumentParams struct {
	ID     int64
	UserID pgtype.Int8
}

func (q *Queries) DeleteDocument(ctx context.Context, arg DeleteDocumentParams) error {
	_, err := q.db.Exec(ctx, deleteDocument, arg.ID, arg.UserID)
	return err
}

const deleteUserEmbeddings = `-- name: DeleteUserEmbeddings :exec
DELETE FROM embeddings e
USING documents d
WHERE d.user_id = $1 
AND e.document_id = d.id
`

func (q *Queries) DeleteUserEmbeddings(ctx context.Context, userID pgtype.Int8) error {
	_, err := q.db.Exec(ctx, deleteUserEmbeddings, userID)
	return err
}

const findSimilarEmbeddingsInDocument = `-- name: FindSimilarEmbeddingsInDocument :many
WITH similarity_scores AS (
    SELECT 
        e.id,
        e.document_id,
        e.chunk_text,
        e.chunk_size,
        e.metadata,
        d.file_path,
        1 - (e.embedding <=> $2::vector) as similarity
    FROM embeddings e
    JOIN documents d ON d.id = e.document_id
    WHERE e.document_id = $3
      AND d.user_id = $4
      AND e.model_name = $5
      AND 1 - (e.embedding <=> $2::vector) > $6
      AND ($7::text IS NULL OR $7::text = e.metadata_hash)
)
SELECT id, document_id, chunk_text, chunk_size, metadata, file_path, similarity
FROM similarity_scores
ORDER BY similarity DESC
LIMIT $1
`

type FindSimilarEmbeddingsInDocumentParams struct {
	MaxResults          int32
	QueryEmbedding      pgvector.Vector
	DocumentID          pgtype.Int8
	UserID              pgtype.Int8
	ModelName           EmbeddingModel
	SimilarityThreshold pgvector.Vector
	MetadataHash        pgtype.Text
}

type FindSimilarEmbeddingsInDocumentRow struct {
	ID         int64
	DocumentID pgtype.Int8
	ChunkText  string
	ChunkSize  int32
	Metadata   types.JSONMap
	FilePath   string
	Similarity int32
}

func (q *Queries) FindSimilarEmbeddingsInDocument(ctx context.Context, arg FindSimilarEmbeddingsInDocumentParams) ([]FindSimilarEmbeddingsInDocumentRow, error) {
	rows, err := q.db.Query(ctx, findSimilarEmbeddingsInDocument,
		arg.MaxResults,
		arg.QueryEmbedding,
		arg.DocumentID,
		arg.UserID,
		arg.ModelName,
		arg.SimilarityThreshold,
		arg.MetadataHash,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FindSimilarEmbeddingsInDocumentRow
	for rows.Next() {
		var i FindSimilarEmbeddingsInDocumentRow
		if err := rows.Scan(
			&i.ID,
			&i.DocumentID,
			&i.ChunkText,
			&i.ChunkSize,
			&i.Metadata,
			&i.FilePath,
			&i.Similarity,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const findTopKNNEmbeddings = `-- name: FindTopKNNEmbeddings :many
SELECT 
    e.id,
    e.document_id,
    e.chunk_text,
    e.chunk_size,
    d.file_path,
    e.metadata,
    (e.embedding <-> $1::vector)::float8 as distance,
    (1 - (e.embedding <-> $1::vector))::float8 as similarity
FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.model_name = $2
  AND ($3::text IS NULL OR $3::text = e.metadata_hash)
AND d.user_id = $4
ORDER BY e.embedding <-> $1::vector ASC
LIMIT $5
`

type FindTopKNNEmbeddingsParams struct {
	QueryEmbedding pgvector.Vector
	ModelName      EmbeddingModel
	MetadataHash   pgtype.Text
	UserID         pgtype.Int8
	K              int32
}

type FindTopKNNEmbeddingsRow struct {
	ID         int64
	DocumentID pgtype.Int8
	ChunkText  string
	ChunkSize  int32
	FilePath   string
	Metadata   types.JSONMap
	Distance   float64
	Similarity float64
}

func (q *Queries) FindTopKNNEmbeddings(ctx context.Context, arg FindTopKNNEmbeddingsParams) ([]FindTopKNNEmbeddingsRow, error) {
	rows, err := q.db.Query(ctx, findTopKNNEmbeddings,
		arg.QueryEmbedding,
		arg.ModelName,
		arg.MetadataHash,
		arg.UserID,
		arg.K,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FindTopKNNEmbeddingsRow
	for rows.Next() {
		var i FindTopKNNEmbeddingsRow
		if err := rows.Scan(
			&i.ID,
			&i.DocumentID,
			&i.ChunkText,
			&i.ChunkSize,
			&i.FilePath,
			&i.Metadata,
			&i.Distance,
			&i.Similarity,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDocument = `-- name: GetDocument :one
SELECT id, user_id, file_path, total_chunk_size, created_at, updated_at FROM documents
WHERE id = $1 AND user_id = $2 LIMIT 1
`

type GetDocumentParams struct {
	ID     int64
	UserID pgtype.Int8
}

func (q *Queries) GetDocument(ctx context.Context, arg GetDocumentParams) (Document, error) {
	row := q.db.QueryRow(ctx, getDocument, arg.ID, arg.UserID)
	var i Document
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.FilePath,
		&i.TotalChunkSize,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getEmbedding = `-- name: GetEmbedding :one
SELECT e.id, e.document_id, e.model_name, e.embedding, e.chunk_text, e.chunk_size, e.created_at, e.metadata, e.metadata_hash FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.id = $1 AND d.user_id = $2 LIMIT 1
`

type GetEmbeddingParams struct {
	ID     int64
	UserID pgtype.Int8
}

func (q *Queries) GetEmbedding(ctx context.Context, arg GetEmbeddingParams) (Embedding, error) {
	row := q.db.QueryRow(ctx, getEmbedding, arg.ID, arg.UserID)
	var i Embedding
	err := row.Scan(
		&i.ID,
		&i.DocumentID,
		&i.ModelName,
		&i.Embedding,
		&i.ChunkText,
		&i.ChunkSize,
		&i.CreatedAt,
		&i.Metadata,
		&i.MetadataHash,
	)
	return i, err
}

const getUserStats = `-- name: GetUserStats :one
SELECT 
    u.total_chunk_size as total_bytes,
    COALESCE(a.request_count, 0) as api_requests,
    COUNT(DISTINCT d.id) as document_count,
    COUNT(e.id) as chunk_count
FROM users u
LEFT JOIN api_usage a ON a.user_id = u.id
LEFT JOIN documents d ON d.user_id = u.id
LEFT JOIN embeddings e ON e.document_id = d.id
WHERE u.id = $1
GROUP BY u.id, u.total_chunk_size, a.request_count
`

type GetUserStatsRow struct {
	TotalBytes    pgtype.Int8
	ApiRequests   int64
	DocumentCount int64
	ChunkCount    int64
}

func (q *Queries) GetUserStats(ctx context.Context, userID int64) (GetUserStatsRow, error) {
	row := q.db.QueryRow(ctx, getUserStats, userID)
	var i GetUserStatsRow
	err := row.Scan(
		&i.TotalBytes,
		&i.ApiRequests,
		&i.DocumentCount,
		&i.ChunkCount,
	)
	return i, err
}

const getUserStorageStats = `-- name: GetUserStorageStats :one
SELECT 
    COUNT(e.id) as embedding_count,
    u.total_chunk_size as total_bytes,
    COUNT(DISTINCT d.id) as document_count
FROM users u
LEFT JOIN documents d ON d.user_id = u.id
LEFT JOIN embeddings e ON e.document_id = d.id
WHERE u.id = $1
GROUP BY u.id, u.total_chunk_size
`

type GetUserStorageStatsRow struct {
	EmbeddingCount int64
	TotalBytes     pgtype.Int8
	DocumentCount  int64
}

func (q *Queries) GetUserStorageStats(ctx context.Context, id int64) (GetUserStorageStatsRow, error) {
	row := q.db.QueryRow(ctx, getUserStorageStats, id)
	var i GetUserStorageStatsRow
	err := row.Scan(&i.EmbeddingCount, &i.TotalBytes, &i.DocumentCount)
	return i, err
}

const incrementAPIUsage = `-- name: IncrementAPIUsage :one
INSERT INTO api_usage (
    user_id,
    request_count,
    last_request_at
)
VALUES (
    $1,
    1,
    CURRENT_TIMESTAMP
)
ON CONFLICT (user_id) DO UPDATE 
SET 
    request_count = api_usage.request_count + 1,
    last_request_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
RETURNING id, user_id, request_count, last_request_at, created_at, updated_at
`

func (q *Queries) IncrementAPIUsage(ctx context.Context, userID pgtype.Int8) (ApiUsage, error) {
	row := q.db.QueryRow(ctx, incrementAPIUsage, userID)
	var i ApiUsage
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.RequestCount,
		&i.LastRequestAt,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listDocumentEmbeddings = `-- name: ListDocumentEmbeddings :many
SELECT e.id, e.document_id, e.model_name, e.embedding, e.chunk_text, e.chunk_size, e.created_at, e.metadata, e.metadata_hash FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.document_id = $1 AND d.user_id = $2
  AND ($3::text IS NULL OR $3::text = e.metadata_hash)
`

type ListDocumentEmbeddingsParams struct {
	DocumentID   pgtype.Int8
	UserID       pgtype.Int8
	MetadataHash pgtype.Text
}

func (q *Queries) ListDocumentEmbeddings(ctx context.Context, arg ListDocumentEmbeddingsParams) ([]Embedding, error) {
	rows, err := q.db.Query(ctx, listDocumentEmbeddings, arg.DocumentID, arg.UserID, arg.MetadataHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Embedding
	for rows.Next() {
		var i Embedding
		if err := rows.Scan(
			&i.ID,
			&i.DocumentID,
			&i.ModelName,
			&i.Embedding,
			&i.ChunkText,
			&i.ChunkSize,
			&i.CreatedAt,
			&i.Metadata,
			&i.MetadataHash,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listUserChunks = `-- name: ListUserChunks :many
SELECT 
    e.id,
    e.chunk_text,
    e.metadata,
    e.chunk_size,
    e.model_name,
    e.created_at,
    d.file_path,
    d.id as document_id
FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE d.user_id = $1
  AND ($2::text IS NULL OR e.metadata_hash = $2::text)
ORDER BY e.created_at DESC
`

type ListUserChunksParams struct {
	UserID       pgtype.Int8
	MetadataHash pgtype.Text
}

type ListUserChunksRow struct {
	ID         int64
	ChunkText  string
	Metadata   types.JSONMap
	ChunkSize  int32
	ModelName  EmbeddingModel
	CreatedAt  pgtype.Timestamptz
	FilePath   string
	DocumentID int64
}

func (q *Queries) ListUserChunks(ctx context.Context, arg ListUserChunksParams) ([]ListUserChunksRow, error) {
	rows, err := q.db.Query(ctx, listUserChunks, arg.UserID, arg.MetadataHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListUserChunksRow
	for rows.Next() {
		var i ListUserChunksRow
		if err := rows.Scan(
			&i.ID,
			&i.ChunkText,
			&i.Metadata,
			&i.ChunkSize,
			&i.ModelName,
			&i.CreatedAt,
			&i.FilePath,
			&i.DocumentID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listUserDocuments = `-- name: ListUserDocuments :many
SELECT id, user_id, file_path, total_chunk_size, created_at, updated_at FROM documents
WHERE user_id = $1
ORDER BY created_at DESC
`

func (q *Queries) ListUserDocuments(ctx context.Context, userID pgtype.Int8) ([]Document, error) {
	rows, err := q.db.Query(ctx, listUserDocuments, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Document
	for rows.Next() {
		var i Document
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.FilePath,
			&i.TotalChunkSize,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

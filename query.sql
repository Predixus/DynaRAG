-- name: CreateDocument :one
INSERT INTO documents (file_path)
VALUES ($1)
ON CONFLICT (file_path) DO UPDATE 
SET updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetDocument :one
SELECT * FROM documents WHERE id = $1 LIMIT 1;

-- name: ListDocuments :many
SELECT * FROM documents ORDER BY created_at DESC;

-- name: DeleteDocument :exec
DELETE FROM documents
WHERE id = $1;

-- name: CreateEmbedding :one
INSERT INTO embeddings (
    document_id,
    model_name,
    embedding,
    chunk_text,
    chunk_size,
    created_at,
    metadata,
    metadata_hash,
    embedding_text
) VALUES (
    $1, $2, $3, $4, length($4), DEFAULT, $5, $6, $7
)
RETURNING id, document_id, model_name, embedding, chunk_text, chunk_size, created_at, metadata, metadata_hash, embedding_text;

-- name: GetEmbedding :one
SELECT e.* FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.id = $1 LIMIT 1;

-- name: DeleteEmbeddings :exec
DELETE FROM embeddings;

-- name: ListDocumentEmbeddings :many
SELECT e.* FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.document_id = $1
  AND (sqlc.narg(metadata_hash)::text IS NULL OR sqlc.narg(metadata_hash)::text = e.metadata_hash);

-- name: GetStorageStats :one
SELECT 
    COUNT(e.id) as embedding_count,
    SUM(e.chunk_size) as total_bytes,
    COUNT(DISTINCT d.id) as document_count
FROM embeddings e
LEFT JOIN documents d ON e.document_id = d.id;

-- name: FindTopKNNEmbeddings :many
SELECT 
    e.id,
    e.document_id,
    e.chunk_text,
    e.chunk_size,
    d.file_path,
    e.metadata,
    (e.embedding <=> sqlc.arg(query_embedding)::vector)::float8 as distance,
    (1 - (e.embedding <=> sqlc.arg(query_embedding)::vector))::float8 as similarity
FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.model_name = sqlc.arg(model_name)
  AND (sqlc.narg(metadata_hash)::text IS NULL OR sqlc.narg(metadata_hash)::text = e.metadata_hash)
ORDER BY e.embedding <=> sqlc.arg(query_embedding)::vector ASC
LIMIT sqlc.arg(k);

-- name: FindSimilarEmbeddingsInDocument :many
WITH similarity_scores AS (
    SELECT 
        e.id,
        e.document_id,
        e.chunk_text,
        e.chunk_size,
        e.metadata,
        d.file_path,
        1 - (e.embedding <=> sqlc.arg(query_embedding)::vector) as similarity
    FROM embeddings e
    JOIN documents d ON d.id = e.document_id
    WHERE e.document_id = sqlc.arg(document_id)
      AND e.model_name = sqlc.arg(model_name)
      AND 1 - (e.embedding <=> sqlc.arg(query_embedding)::vector) > sqlc.arg(similarity_threshold)
      AND (sqlc.narg(metadata_hash)::text IS NULL OR sqlc.narg(metadata_hash)::text = e.metadata_hash)
)
SELECT *
FROM similarity_scores
ORDER BY similarity DESC
LIMIT sqlc.arg(max_results);

-- name: GetStats :one
SELECT 
    COUNT(DISTINCT d.id) as document_count,
    COUNT(e.id) as chunk_count,
    COALESCE(SUM(e.chunk_size), 0) as total_bytes
FROM documents d
LEFT JOIN embeddings e ON e.document_id = d.id;

-- name: ListChunks :many
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
WHERE (sqlc.narg(metadata_hash)::text IS NULL OR e.metadata_hash = sqlc.narg(metadata_hash)::text)
ORDER BY e.created_at DESC;

-- name: SearchDocumentsBM25 :many
SELECT 
    d.id,
    d.file_path,
    d.total_chunk_size,
    d.created_at,
    d.updated_at,
    -- BM25 rank:
    ts_rank_cd(d.text_searchable_column, to_tsquery('english', $1)) AS rank
FROM documents d
WHERE d.text_searchable_column @@ to_tsquery('english', $1)
ORDER BY rank DESC
LIMIT 50;
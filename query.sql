-- name: CreateUser :one
INSERT INTO users (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE 
SET updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: CreateDocument :one
INSERT INTO documents (user_id, file_path)
VALUES ($1, $2)
ON CONFLICT (user_id, file_path) DO UPDATE 
SET updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetDocument :one
SELECT * FROM documents
WHERE id = $1 AND user_id = $2 LIMIT 1;

-- name: ListUserDocuments :many
SELECT * FROM documents
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: DeleteDocument :exec
DELETE FROM documents
WHERE id = $1 AND user_id = $2;

-- name: CreateEmbedding :one
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
RETURNING *;

-- name: GetEmbedding :one
SELECT e.* FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.id = $1 AND d.user_id = $2 LIMIT 1;

-- name: DeleteUserEmbeddings :exec
DELETE FROM embeddings e
USING documents d
WHERE d.user_id = $1 
AND e.document_id = d.id;

-- name: ListDocumentEmbeddings :many
SELECT e.* FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.document_id = $1 AND d.user_id = $2
  AND (sqlc.narg(metadata_hash)::text IS NULL OR sqlc.narg(metadata_hash)::text = e.metadata_hash);

-- name: GetUserStorageStats :one
SELECT 
    COUNT(e.id) as embedding_count,
    u.total_chunk_size as total_bytes,
    COUNT(DISTINCT d.id) as document_count
FROM users u
LEFT JOIN documents d ON d.user_id = u.id
LEFT JOIN embeddings e ON e.document_id = d.id
WHERE u.id = $1
GROUP BY u.id, u.total_chunk_size;

-- name: FindTopKNNEmbeddings :many
SELECT 
    e.id,
    e.document_id,
    e.chunk_text,
    e.chunk_size,
    d.file_path,
    e.metadata,
    (e.embedding <-> sqlc.arg(query_embedding)::vector)::float8 as distance,
    (1 - (e.embedding <-> sqlc.arg(query_embedding)::vector))::float8 as similarity
FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE e.model_name = sqlc.arg(model_name)
  AND d.user_id = sqlc.arg(user_id)
  AND (sqlc.narg(metadata_hash)::text IS NULL OR sqlc.narg(metadata_hash)::text = e.metadata_hash)
ORDER BY e.embedding <-> sqlc.arg(query_embedding)::vector ASC
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
      AND d.user_id = sqlc.arg(user_id)
      AND e.model_name = sqlc.arg(model_name)
      AND 1 - (e.embedding <=> sqlc.arg(query_embedding)::vector) > sqlc.arg(similarity_threshold)
      AND (sqlc.narg(metadata_hash)::text IS NULL OR sqlc.narg(metadata_hash)::text = e.metadata_hash)
)
SELECT *
FROM similarity_scores
ORDER BY similarity DESC
LIMIT sqlc.arg(max_results);

-- name: GetUserStats :one
SELECT 
    u.total_chunk_size as total_bytes,
    COALESCE(a.request_count, 0) as api_requests,
    COUNT(DISTINCT d.id) as document_count,
    COUNT(e.id) as chunk_count
FROM users u
LEFT JOIN api_usage a ON a.user_id = u.id
LEFT JOIN documents d ON d.user_id = u.id
LEFT JOIN embeddings e ON e.document_id = d.id
WHERE u.id = sqlc.arg(user_id)
GROUP BY u.id, u.total_chunk_size, a.request_count;

-- name: ListUserChunks :many
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
WHERE d.user_id = sqlc.arg(user_id)
  AND (sqlc.narg(metadata_hash)::text IS NULL OR e.metadata_hash = sqlc.narg(metadata_hash)::text)
ORDER BY e.created_at DESC;

-- name: IncrementAPIUsage :one
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
RETURNING *;

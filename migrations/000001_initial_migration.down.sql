-- Reset TOAST settings if needed (though this will be dropped with the table anyway)
ALTER TABLE IF EXISTS embeddings RESET (toast_tuple_target);

-- Drop indexes first
DROP INDEX IF EXISTS embeddings_embedding_idx;
DROP INDEX IF EXISTS embeddings_chunk_text_idx;
DROP INDEX IF EXISTS api_usage_user_id_idx;

-- Drop triggers
DROP TRIGGER IF EXISTS update_chunk_sizes_trigger ON embeddings;
DROP TRIGGER IF EXISTS delete_empty_documents ON embeddings;
DROP TRIGGER IF EXISTS create_api_usage_record ON users;

-- Drop trigger functions
DROP FUNCTION IF EXISTS update_chunk_sizes();
DROP FUNCTION IF EXISTS check_document_embeddings();
DROP FUNCTION IF EXISTS initialize_api_usage();
DROP FUNCTION IF EXISTS increment_api_usage(BIGINT);

-- Drop tables (in correct dependency order)
DROP TABLE IF EXISTS embeddings;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS api_usage;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS index_requests;

-- Drop types
DROP TYPE IF EXISTS embedding_model;

-- Drop extensions (in reverse order of creation)
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS vector;

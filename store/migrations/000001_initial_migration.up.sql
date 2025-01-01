CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- For text search optimisation

CREATE TYPE embedding_model AS ENUM (
    'all-MiniLM-L6-v2',
    'all-mpnet-base-v2',
    'multi-CAUTION-MiniLM-L6-cos-v1'
);

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    total_chunk_size BIGINT DEFAULT 0,  -- Track total size in bytes
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE documents (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    total_chunk_size BIGINT DEFAULT 0,  -- Track size per document
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, file_path)         -- Composite unique constraint
);

CREATE TABLE embeddings (
    id BIGSERIAL PRIMARY KEY,
    document_id BIGINT REFERENCES documents(id) ON DELETE CASCADE,
    model_name embedding_model NOT NULL,
    embedding vector(384) NOT NULL,
    chunk_text TEXT NOT NULL,       -- Postgres will automatically use TOAST for large text values
    chunk_size INTEGER NOT NULL,    -- Size of chunk in bytes
    created_at TIMESTAMPTZ DEFAULT CURRENT_ [!TIP]
    > EATE TABLE api_usage (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    request_count BIGINT DEFAULT 0,  
    last_request_at TIMESTAMPTZ,     
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create index on user_id for faster lookups
CREATE INDEX api_usage_user_id_idx ON api_usage(user_id);

-- Function to initialize API usage tracking for new users
CREATE OR REPLACE FUNCTION initialize_api_usage()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO api_usage (user_id)
    VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to create API usage record when new user is created
CREATE TRIGGER create_api_usage_record
AFTER INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION initialize_api_usage();

-- Function to increment API usage
CREATE OR REPLACE FUNCTION increment_api_usage(user_id_param BIGINT)
RETURNS VOID AS $$
BEGIN
    UPDATE api_usage
    SET 
        request_count = request_count + 1,
        last_request_at = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP
    WHERE user_id = user_id_param;
END;
$$ LANGUAGE plpgsql;

-- Enable TOAST compression for the embeddings table
ALTER TABLE embeddings SET (toast_tuple_target = 128);

CREATE INDEX embeddings_embedding_idx ON embeddings USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX embeddings_chunk_text_idx ON embeddings USING gin (chunk_text gin_trgm_ops);

-- Function to update chunk sizes
CREATE OR REPLACE FUNCTION update_chunk_sizes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- First update the documents total_chunk_size
        UPDATE documents 
        SET 
            total_chunk_size = total_chunk_size + NEW.chunk_size,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.document_id;
        
        -- Then update the users total_chunk_size in a single step
        UPDATE users 
        SET 
            total_chunk_size = total_chunk_size + NEW.chunk_size,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = (
            SELECT user_id 
            FROM documents 
            WHERE id = NEW.document_id
        );
        
    ELSIF TG_OP = 'DELETE' THEN
        -- First update the documents total_chunk_size
        UPDATE documents 
        SET 
            total_chunk_size = GREATEST(0, total_chunk_size - OLD.chunk_size),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = OLD.document_id;
        
        -- Then update the users total_chunk_size in a single step
        UPDATE users 
        SET 
            total_chunk_size = GREATEST(0, total_chunk_size - OLD.chunk_size),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = (
            SELECT user_id 
            FROM documents 
            WHERE id = OLD.document_id
        );
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger for chunk size updates
CREATE TRIGGER update_chunk_sizes_trigger
AFTER INSERT OR DELETE ON embeddings
FOR EACH ROW
EXECUTE FUNCTION update_chunk_sizes();

-- Function to check if document has embeddings
CREATE OR REPLACE FUNCTION check_document_embeddings()
RETURNS TRIGGER AS $$
BEGIN
    -- If no embeddings exist for this document
    IF NOT EXISTS (
        SELECT 1 
        FROM embeddings 
        WHERE document_id = OLD.document_id
    ) THEN
        -- Get the document's user_id before deleting it
        DELETE FROM documents WHERE id = OLD.document_id;
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Trigger to delete documents with no embeddings
CREATE TRIGGER delete_empty_documents
AFTER DELETE ON embeddings
FOR EACH ROW
EXECUTE FUNCTION check_document_embeddings();

-- Function to reset user chunk size
CREATE OR REPLACE FUNCTION reset_user_chunk_size()
RETURNS TRIGGER AS $$
BEGIN
    -- After a document is deleted, check if the user has any remaining documents
    IF NOT EXISTS (
        SELECT 1 
        FROM documents 
        WHERE user_id = OLD.user_id
    ) THEN
        -- If no documents remain, reset the user's total_chunk_size to 0
        UPDATE users 
        SET 
            total_chunk_size = 0,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = OLD.user_id;
    ELSE
        -- If documents remain, recalculate the total from existing documents
        UPDATE users u
        SET 
            total_chunk_size = COALESCE((
                SELECT SUM(e.chunk_size)
                FROM documents d
                JOIN embeddings e ON d.id = e.document_id
                WHERE d.user_id = OLD.user_id
            ), 0),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = OLD.user_id;
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Trigger to reset user chunk size
CREATE TRIGGER reset_user_chunk_size_trigger
AFTER DELETE ON documents
FOR EACH ROW
EXECUTE FUNCTION reset_user_chunk_size();

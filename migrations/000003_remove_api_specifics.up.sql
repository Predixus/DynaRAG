-- Drop triggers first
DROP TRIGGER IF EXISTS reset_user_chunk_size_trigger ON documents;
DROP TRIGGER IF EXISTS create_api_usage_record ON users;

-- Drop functions
DROP FUNCTION IF EXISTS initialize_api_usage();
DROP FUNCTION IF EXISTS increment_api_usage(BIGINT);
DROP FUNCTION IF EXISTS reset_user_chunk_size();

-- Drop foreign key constraints
ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_user_id_fkey;
ALTER TABLE api_usage DROP CONSTRAINT IF EXISTS api_usage_user_id_fkey;

-- Drop indexes
DROP INDEX IF EXISTS api_usage_user_id_idx;

-- Drop the column from documents
ALTER TABLE documents DROP COLUMN IF EXISTS user_id;

-- Now we can safely drop the tables
DROP TABLE IF EXISTS api_usage;
DROP TABLE IF EXISTS users;
DROP INDEX api_usage_user_id_idx;

-- Drop functions and triggers
DROP FUNCTION initialize_api_usage;
DROP FUNCTION increment_api_usage;
DROP FUNCTION reset_user_chunk_size;
DROP TRIGGER reset_user_chunk_size_trigger ON documents;

-- Modify document table
ALTER TABLE documents
DROP COLUMN user_id,
DROP CONSTRAINT documents_user_id_fkey;

-- Update update_chunk_sizes function to remove user references
CREATE OR REPLACE FUNCTION update_chunk_sizes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Update the documents total_chunk_size
        UPDATE documents 
        SET 
            total_chunk_size = total_chunk_size + NEW.chunk_size,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.document_id;
        
    ELSIF TG_OP = 'DELETE' THEN
        -- Update the documents total_chunk_size
        UPDATE documents 
        SET 
            total_chunk_size = GREATEST(0, total_chunk_size - OLD.chunk_size),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = OLD.document_id;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

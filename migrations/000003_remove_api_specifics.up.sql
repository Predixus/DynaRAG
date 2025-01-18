-- drop triggers that depend on the functions
DROP TRIGGER IF EXISTS reset_user_chunk_size_trigger ON documents;
DROP TRIGGER IF EXISTS create_api_usage_record ON users;
DROP TRIGGER IF EXISTS update_chunk_sizes_trigger ON embeddings;

-- drop functions that reference user-related tables
DROP FUNCTION IF EXISTS initialize_api_usage();
DROP FUNCTION IF EXISTS increment_api_usage(BIGINT);
DROP FUNCTION IF EXISTS reset_user_chunk_size();
DROP FUNCTION IF EXISTS update_chunk_sizes();

-- drop indexes
DROP INDEX IF EXISTS api_usage_user_id_idx;

-- drop foreign key constraints
ALTER TABLE documents 
    DROP CONSTRAINT IF EXISTS documents_user_id_fkey;
ALTER TABLE api_usage 
    DROP CONSTRAINT IF EXISTS api_usage_user_id_fkey;

-- drop dependent tables first
DROP TABLE IF EXISTS api_usage;

-- modify documents table to remove user_id
ALTER TABLE documents 
    DROP COLUMN IF EXISTS user_id;

-- drop users table last since other tables depended on it
DROP TABLE IF EXISTS users;

-- create new update_chunk_sizes function without user references
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

-- recreate the trigger for chunk size updates
CREATE TRIGGER update_chunk_sizes_trigger
AFTER INSERT OR DELETE ON embeddings
FOR EACH ROW
EXECUTE FUNCTION update_chunk_sizes();

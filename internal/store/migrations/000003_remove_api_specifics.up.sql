-- Drop user-related tables and functions
DROP TABLE users;
DROP TABLE api_usage;

-- Drop indexes
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

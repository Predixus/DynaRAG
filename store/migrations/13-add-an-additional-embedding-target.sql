DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'embeddings'
        AND column_name = 'embedding_text'
    ) THEN
        ALTER TABLE embeddings
        ADD COLUMN embedding_text TEXT;
    END IF;
END $$;

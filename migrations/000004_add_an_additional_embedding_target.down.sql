ALTER TABLE embeddings
ADD COLUMN IF NOT EXISTS embedding_text TEXT;

ALTER TABLE embeddings
ADD COLUMN metadata JSONB DEFAULT '{}'::jsonb NOT NULL,
ADD COLUMN metadata_hash TEXT;

CREATE INDEX  embeddings_metadata_hash_idx ON embeddings(metadata_hash);

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS found_item_embeddings (
  found_item_id UUID PRIMARY KEY,
  embedding vector(1536) NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

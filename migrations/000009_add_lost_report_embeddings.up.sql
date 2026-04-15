-- Stores pgvector embeddings for lost reports to support similarity search.
-- Matches the dimensions of OpenAI text-embedding-3-small (1536).

CREATE TABLE IF NOT EXISTS lost_report_embeddings (
  lost_report_id UUID PRIMARY KEY REFERENCES lost_reports(id) ON DELETE CASCADE,
  embedding vector(1536) NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


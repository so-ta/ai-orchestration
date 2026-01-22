-- Block Embeddings table for vector search
-- This enables semantic search across thousands of blocks

-- Enable pgvector extension if not already enabled
CREATE EXTENSION IF NOT EXISTS vector;

-- Create block_embeddings table
CREATE TABLE IF NOT EXISTS block_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_definition_id UUID REFERENCES block_definitions(id) ON DELETE CASCADE,
    slug VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    tags TEXT[],
    -- 1536 dimensions for OpenAI text-embedding-3-small
    embedding vector(1536),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(block_definition_id)
);

-- Create indexes for efficient querying
-- IVFFlat index for approximate nearest neighbor search
CREATE INDEX IF NOT EXISTS idx_block_embeddings_vector
    ON block_embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Category filter index
CREATE INDEX IF NOT EXISTS idx_block_embeddings_category
    ON block_embeddings(category);

-- Slug lookup index
CREATE INDEX IF NOT EXISTS idx_block_embeddings_slug
    ON block_embeddings(slug);

-- Tags index using GIN for array containment queries
CREATE INDEX IF NOT EXISTS idx_block_embeddings_tags
    ON block_embeddings USING GIN(tags);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_block_embeddings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update updated_at on row update
DROP TRIGGER IF EXISTS trigger_block_embeddings_updated_at ON block_embeddings;
CREATE TRIGGER trigger_block_embeddings_updated_at
    BEFORE UPDATE ON block_embeddings
    FOR EACH ROW
    EXECUTE FUNCTION update_block_embeddings_updated_at();

-- Add comment to explain the table purpose
COMMENT ON TABLE block_embeddings IS 'Vector embeddings for semantic block search. Used by Copilot to find relevant blocks based on natural language queries.';
COMMENT ON COLUMN block_embeddings.embedding IS 'OpenAI text-embedding-3-small vector (1536 dimensions)';

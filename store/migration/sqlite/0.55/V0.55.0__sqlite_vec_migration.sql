-- Migration to sqlite-vec for efficient vector search
-- This migration converts the memo_embedding table to use vec0 virtual tables

-- Step 1: Create new memo_embedding_vec table with vec0 support
CREATE TABLE IF NOT EXISTS memo_embedding_vec (
  memo_id INTEGER PRIMARY KEY,
  model TEXT NOT NULL,
  embedding BLOB NOT NULL,  -- vec0 format
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE
);

-- Step 2: Create vec0 virtual table for efficient KNN search
-- The vec0 table stores vectors in a format optimized for fast similarity search
CREATE VIRTUAL TABLE IF NOT EXISTS vec0 USING vec0(
  embedding float32[1024]  -- BAAI/bge-m3 produces 1024-dimensional vectors
);

-- Step 3: Create an index on vec0 for fast KNN queries
-- This enables vec0_knn functions to run in O(log n) instead of O(n)
CREATE INDEX IF NOT EXISTS vec0_index ON vec0(vec0_distance_cosine(embedding));

-- Step 4: Migrate existing data (if any)
INSERT OR IGNORE INTO memo_embedding_vec (memo_id, model, embedding, created_ts, updated_ts)
SELECT
  memo_id,
  model,
  embedding,  -- Keep as-is for now, will convert in VectorSearch
  created_ts,
  updated_ts
FROM memo_embedding;

-- Step 5: Populate vec0 table with existing embeddings
-- Note: This requires converting from JSON to vec0 binary format
-- We'll do this lazily in the VectorSearch function

-- Step 6: Drop old table (after successful migration)
-- DROP TABLE IF EXISTS memo_embedding;
-- ALTER TABLE memo_embedding_vec RENAME TO memo_embedding;

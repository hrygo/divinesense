-- Add memo_tags table for AI-generated memo tags
-- Stores AI-suggested tags with confidence scores

CREATE TABLE memo_tags (
  id SERIAL PRIMARY KEY,
  memo_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  confidence REAL NOT NULL DEFAULT 1.0,
  source VARCHAR(20) NOT NULL DEFAULT 'llm',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
  CONSTRAINT fk_memo_tags_memo
    FOREIGN KEY (memo_id)
    REFERENCES memo(id)
    ON DELETE CASCADE,
  CONSTRAINT uq_memo_tags_memo_tag
    UNIQUE (memo_id, tag)
);

-- Index for memo_id lookup
CREATE INDEX idx_memo_tags_memo_id
ON memo_tags (memo_id);

-- Index for tag filtering
CREATE INDEX idx_memo_tags_tag
ON memo_tags (tag);

COMMENT ON TABLE memo_tags IS 'Stores AI-generated tags for memos with confidence scores';
COMMENT ON COLUMN memo_tags.source IS 'Tag source: llm, rules, statistics, user';

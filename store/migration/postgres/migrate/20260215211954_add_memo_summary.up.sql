-- Add memo_summary table for AI-generated memo summaries
-- Stores summary (≤200 chars) with generation status

CREATE TABLE memo_summary (
  id SERIAL PRIMARY KEY,
  memo_id INTEGER NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
  error_message TEXT,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
  CONSTRAINT fk_memo_summary_memo
    FOREIGN KEY (memo_id)
    REFERENCES memo(id)
    ON DELETE CASCADE,
  CONSTRAINT uq_memo_summary_memo
    UNIQUE (memo_id)
);

-- Index for memo_id lookup
CREATE INDEX idx_memo_summary_memo_id
ON memo_summary (memo_id);

-- Index for status filtering
CREATE INDEX idx_memo_summary_status
ON memo_summary (status);

-- Auto-update timestamp trigger
CREATE OR REPLACE FUNCTION update_memo_summary_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_memo_summary_updated_ts
  BEFORE UPDATE ON memo_summary
  FOR EACH ROW
  EXECUTE FUNCTION update_memo_summary_updated_ts();

COMMENT ON TABLE memo_summary IS 'Stores AI-generated summaries for memos (max 200 characters)';
COMMENT ON COLUMN memo_summary.status IS 'Generation status: PENDING, GENERATING, COMPLETED, FAILED';

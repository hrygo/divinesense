-- This migration adds FTS5 full-text search with Unicode support for Chinese text.
-- It creates the memo_fts virtual table with unicode61 tokenizer.
--
-- Unicode61 tokenizer features:
-- - Tokenizes by Unicode characters (works for all languages including Chinese)
-- - Normalizes diacritics
-- - Case-insensitive
--
-- Migration: 0.56.0

-- Step 1: Create FTS5 virtual table with unicode61 tokenizer
-- This table stores the full-text search index for memo content
CREATE VIRTUAL TABLE IF NOT EXISTS memo_fts USING fts5(
    content,
    content='memo',
    content_rowid='id',
    tokenize='unicode61'
);

-- Step 2: Create triggers to keep FTS5 in sync with memo table
-- Trigger for INSERT
CREATE TRIGGER IF NOT EXISTS memo_fts_insert AFTER INSERT ON memo BEGIN
    INSERT INTO memo_fts(rowid, content) VALUES (new.id, new.content);
END;

-- Trigger for DELETE
CREATE TRIGGER IF NOT EXISTS memo_fts_delete AFTER DELETE ON memo BEGIN
    INSERT INTO memo_fts(memo_fts, rowid, content) VALUES('delete', old.id, old.content);
END;

-- Trigger for UPDATE
CREATE TRIGGER IF NOT EXISTS memo_fts_update AFTER UPDATE ON memo BEGIN
    INSERT INTO memo_fts(memo_fts, rowid, content) VALUES('delete', old.id, old.content);
    INSERT INTO memo_fts(rowid, content) VALUES (new.id, new.content);
END;

-- Step 3: Rebuild FTS5 index with existing data
-- This populates the FTS5 table with existing memo content
INSERT INTO memo_fts(memo_fts) VALUES('rebuild');

-- Step 4: Optimize FTS5 index for better query performance
INSERT INTO memo_fts(memo_fts) VALUES('optimize');

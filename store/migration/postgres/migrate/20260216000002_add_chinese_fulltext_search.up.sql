-- Chinese full-text search migration
-- Uses simple text search configuration (works for both Chinese and English)
--
-- Migration: 20260216000002

-- Step 1: Create text search configuration
CREATE TEXT SEARCH CONFIGURATION IF NOT EXISTS chinese (
    parser = default
);

-- Step 2: Add token mappings
ALTER TEXT SEARCH CONFIGURATION chinese
    ADD MAPPING FOR word WITH simple;

-- Step 3: Create helper function for tsvector
CREATE OR REPLACE FUNCTION to_tsvector_chinese(text)
RETURNS tsvector AS $$
BEGIN
    RETURN to_tsvector('simple', COALESCE($1, ''));
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Step 4: Create helper function for tsquery
CREATE OR REPLACE FUNCTION to_tsquery_chinese(text)
RETURNS tsquery AS $$
BEGIN
    RETURN plainto_tsquery('simple', $1);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

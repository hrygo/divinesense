-- This migration improves PostgreSQL full-text search for Chinese text.
-- It switches from 'simple' to 'ngram' tokenizer which works better for CJK languages.
--
-- Migration: 20260216000002

-- Step 1: Drop existing function if exists (for idempotency)
DROP FUNCTION IF EXISTS to_tsvector_chinese(CSTRING);
DROP TEXT SEARCH CONFIGURATION IF EXISTS chinese;

-- Step 2: Create ngram tokenizer configuration for Chinese
-- The ngram tokenizer breaks text into n-grams, which works well for Chinese
CREATE TEXT SEARCH CONFIGURATION chinese (
    parser = ngram
);

-- Step 3: Add token mappings for ngram (normalize to lowercase)
ALTER TEXT SEARCH CONFIGURATION chinese
    ADD MAPPING FOR word WITH simple;
ALTER TEXT SEARCH CONFIGURATION chinese
    ADD MAPPING FOR 1 WITH simple;
ALTER TEXT SEARCH CONFIGURATION chinese
    ADD MAPPING FOR 2 WITH simple;

-- Step 4: Create a helper function for bilingual search
-- This function uses 'chinese' config for Chinese text and 'simple' for English
CREATE OR REPLACE FUNCTION to_tsvector_chinese(text)
RETURNS tsvector AS $$
BEGIN
    -- If text contains Chinese characters (Unicode range), use chinese config
    IF text ~ '[^\x00-\x7F]' THEN
        RETURN to_tsvector('chinese', COALESCE($1, ''));
    ELSE
        -- Otherwise use simple config (better for English)
        RETURN to_tsvector('simple', COALESCE($1, ''));
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Step 5: Create a helper function for bilingual query
CREATE OR REPLACE FUNCTION to_tsquery_chinese(text)
RETURNS tsquery AS $$
BEGIN
    -- If text contains Chinese characters, use chinese config
    IF $1 ~ '[^\x00-\x7F]' THEN
        RETURN plainto_tsquery('chinese', $1);
    ELSE
        RETURN plainto_tsquery('simple', $1);
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

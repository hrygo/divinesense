-- Rollback migration for Chinese full-text search.
-- Restores to simple configuration.
--
-- Migration: 20260216000002

-- Step 1: Drop helper functions
DROP FUNCTION IF EXISTS to_tsvector_chinese(CASCADE);
DROP FUNCTION IF EXISTS to_tsquery_chinese(CASCADE);

-- Step 2: Drop text search configuration
DROP TEXT SEARCH CONFIGURATION IF EXISTS chinese;

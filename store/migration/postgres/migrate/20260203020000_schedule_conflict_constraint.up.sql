-- Add atomic conflict detection constraint for schedules
-- This migration adds an EXCLUDE constraint to prevent overlapping schedules
-- at the database level, providing atomic conflict detection

-- Enable btree_gist extension if not already enabled
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Add the EXCLUDE constraint to prevent overlapping schedules
-- This ensures that no two NORMAL status schedules for the same creator
-- can have overlapping time ranges
ALTER TABLE schedule
ADD CONSTRAINT IF NOT EXISTS no_overlapping_schedules
EXCLUDE USING gist (
    creator_id WITH =,
    tsrange(start_ts, COALESCE(end_ts, start_ts + 3600), '[)') WITH &&
)
WHERE (row_status = 'NORMAL');

-- Add index for better query performance
CREATE INDEX IF NOT EXISTS idx_schedule_creator_time
ON schedule(creator_id, start_ts)
WHERE row_status = 'NORMAL';

-- Add index for conflict checking
CREATE INDEX IF NOT EXISTS idx_schedule_creator_time_range
ON schedule USING gist (
    creator_id,
    tsrange(start_ts, COALESCE(end_ts, start_ts + 3600), '[)')
)
WHERE row_status = 'NORMAL';

-- Add comment to document the constraint
COMMENT ON CONSTRAINT no_overlapping_schedules ON schedule IS
'Prevents overlapping schedules for the same user. Only applies to NORMAL status schedules.';

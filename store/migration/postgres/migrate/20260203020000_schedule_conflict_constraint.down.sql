-- Rollback: Remove atomic conflict detection constraint
DROP INDEX IF EXISTS idx_schedule_creator_time_range;
DROP INDEX IF EXISTS idx_schedule_creator_time;
ALTER TABLE schedule DROP CONSTRAINT IF EXISTS no_overlapping_schedules;

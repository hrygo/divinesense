-- Rollback router feedback and dynamic weight tracking tables

DROP TRIGGER IF EXISTS trigger_router_weight_updated_ts ON router_weight;
DROP FUNCTION IF EXISTS update_router_weight_updated_ts();

DROP INDEX IF EXISTS idx_router_weight_user_category;
DROP INDEX IF EXISTS idx_router_weight_user;
DROP TABLE IF EXISTS router_weight;

DROP INDEX IF EXISTS idx_router_feedback_intent;
DROP INDEX IF EXISTS idx_router_feedback_user_type;
DROP INDEX IF EXISTS idx_router_feedback_user_timestamp;
DROP TABLE IF EXISTS router_feedback;

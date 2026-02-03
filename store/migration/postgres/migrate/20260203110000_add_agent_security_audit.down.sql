-- Rollback: Drop agent_security_audit table
DROP INDEX IF EXISTS idx_audit_session;
DROP INDEX IF EXISTS idx_audit_operation;
DROP INDEX IF EXISTS idx_audit_risk;
DROP INDEX IF EXISTS idx_audit_user_date;
DROP TABLE IF EXISTS agent_security_audit;

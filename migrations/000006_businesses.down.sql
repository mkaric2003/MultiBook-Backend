ALTER TABLE users DROP CONSTRAINT IF EXISTS users_selected_business_id_fkey;
ALTER TABLE users ALTER COLUMN selected_business_id TYPE TEXT USING selected_business_id::TEXT;

DROP TABLE IF EXISTS business_audit_events;
DROP TRIGGER IF EXISTS businesses_set_updated_at ON businesses;
DROP TABLE IF EXISTS businesses;
DROP TYPE IF EXISTS business_status;
DROP TYPE IF EXISTS business_type;

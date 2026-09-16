ALTER TABLE users ADD COLUMN IF NOT EXISTS selected_business_id TEXT;
CREATE INDEX IF NOT EXISTS users_active_selected_business_idx ON users (selected_business_id) WHERE deleted_at IS NULL AND selected_business_id IS NOT NULL;

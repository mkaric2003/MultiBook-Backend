DROP INDEX IF EXISTS users_active_selected_business_idx;
ALTER TABLE users DROP COLUMN IF EXISTS selected_business_id;

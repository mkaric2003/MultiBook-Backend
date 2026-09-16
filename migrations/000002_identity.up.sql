CREATE TYPE user_role AS ENUM ('customer', 'provider', 'admin');

CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email CITEXT,
  display_name TEXT,
  phone_e164 TEXT,
  avatar_storage_path TEXT,
  role user_role,
  selected_business_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT users_id_not_blank CHECK (length(trim(id)) > 0)
);

CREATE INDEX users_email_idx ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX users_active_role_idx ON users (role) WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

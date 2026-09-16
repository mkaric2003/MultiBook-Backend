CREATE TYPE business_type AS ENUM ('stay', 'service');
CREATE TYPE business_status AS ENUM ('draft', 'active', 'inactive', 'archived');

CREATE TABLE businesses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  type business_type NOT NULL,
  status business_status NOT NULL DEFAULT 'active',
  name TEXT NOT NULL,
  name_normalized TEXT NOT NULL,
  category_id TEXT NOT NULL,
  currency CHAR(3) NOT NULL,
  short_description TEXT,
  average_rating NUMERIC(3,2) NOT NULL DEFAULT 0,
  review_count INTEGER NOT NULL DEFAULT 0,
  is_promotion_active BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT businesses_name_not_blank CHECK (length(btrim(name)) > 0),
  CONSTRAINT businesses_name_normalized_not_blank CHECK (length(btrim(name_normalized)) > 0),
  CONSTRAINT businesses_category_not_blank CHECK (length(btrim(category_id)) > 0),
  CONSTRAINT businesses_currency_check CHECK (currency IN ('BAM', 'USD', 'EUR', 'CHF', 'GBP')),
  CONSTRAINT businesses_average_rating_check CHECK (average_rating >= 0 AND average_rating <= 5),
  CONSTRAINT businesses_review_count_check CHECK (review_count >= 0)
);

CREATE INDEX businesses_active_type_idx
  ON businesses (type, created_at DESC)
  WHERE deleted_at IS NULL AND status = 'active';
CREATE INDEX businesses_active_owner_idx
  ON businesses (owner_id, created_at DESC)
  WHERE deleted_at IS NULL;
CREATE INDEX businesses_active_category_idx
  ON businesses (type, category_id, created_at DESC)
  WHERE deleted_at IS NULL AND status = 'active';
CREATE INDEX businesses_active_name_normalized_idx
  ON businesses (name_normalized)
  WHERE deleted_at IS NULL AND status = 'active';

CREATE TRIGGER businesses_set_updated_at
BEFORE UPDATE ON businesses
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE business_audit_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
  actor_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  action TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT business_audit_events_action_not_blank CHECK (length(btrim(action)) > 0)
);

CREATE INDEX business_audit_events_business_created_idx
  ON business_audit_events (business_id, created_at DESC);

-- Firestore business document IDs cannot become relational references. The
-- PostgreSQL business module starts with no imported records, so an existing
-- transient selection is cleared and will be set again only after ownership is
-- checked by the API.
UPDATE users SET selected_business_id = NULL WHERE selected_business_id IS NOT NULL;

ALTER TABLE users
  ALTER COLUMN selected_business_id TYPE UUID USING selected_business_id::UUID;

ALTER TABLE users
  ADD CONSTRAINT users_selected_business_id_fkey
  FOREIGN KEY (selected_business_id) REFERENCES businesses(id) ON DELETE SET NULL;

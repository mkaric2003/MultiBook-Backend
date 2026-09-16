CREATE TYPE promotion_type AS ENUM ('percentage', 'fixedAmount', 'couponCode');

CREATE TABLE business_promotions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  type promotion_type NOT NULL,
  value BIGINT NOT NULL,
  starts_at TIMESTAMPTZ NOT NULL,
  ends_at TIMESTAMPTZ NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  code CITEXT,
  minimum_amount_minor BIGINT NOT NULL DEFAULT 0,
  minimum_nights INTEGER NOT NULL DEFAULT 1,
  usage_limit INTEGER,
  usage_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT business_promotions_name_check CHECK (length(trim(name)) BETWEEN 1 AND 160),
  CONSTRAINT business_promotions_value_check CHECK (value > 0),
  CONSTRAINT business_promotions_percentage_check CHECK (type = 'fixedAmount' OR value <= 100),
  CONSTRAINT business_promotions_dates_check CHECK (ends_at >= starts_at),
  CONSTRAINT business_promotions_code_check CHECK ((type = 'couponCode' AND code IS NOT NULL AND length(trim(code::TEXT)) > 0) OR (type <> 'couponCode')),
  CONSTRAINT business_promotions_minimums_check CHECK (minimum_amount_minor >= 0 AND minimum_nights >= 0),
  CONSTRAINT business_promotions_usage_check CHECK (usage_count >= 0 AND (usage_limit IS NULL OR (usage_limit > 0 AND usage_count <= usage_limit)))
);

CREATE INDEX business_promotions_owner_created_idx
  ON business_promotions (owner_id, business_id, created_at DESC, id DESC);

CREATE INDEX business_promotions_active_lookup_idx
  ON business_promotions (business_id, type, value DESC, created_at DESC)
  WHERE is_active;

CREATE TRIGGER business_promotions_set_updated_at
BEFORE UPDATE ON business_promotions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

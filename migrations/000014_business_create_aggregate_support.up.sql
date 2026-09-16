CREATE TABLE business_featured_collections (
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  collection_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (business_id, collection_id),
  CONSTRAINT business_featured_collections_not_blank CHECK (length(btrim(collection_id)) > 0)
);

CREATE TABLE stay_extras (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  extra_type TEXT NOT NULL,
  price_minor BIGINT NOT NULL,
  pricing_unit TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT stay_extras_type_not_blank CHECK (length(btrim(extra_type)) > 0),
  CONSTRAINT stay_extras_price_check CHECK (price_minor >= 0),
  CONSTRAINT stay_extras_pricing_unit_check CHECK (pricing_unit IN ('per_night', 'one_time', 'per_hour')),
  UNIQUE (business_id, extra_type)
);

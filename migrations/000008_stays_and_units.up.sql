CREATE TYPE stay_inventory_type AS ENUM ('single_unit', 'multiple_units');

CREATE TABLE stay_details (
  business_id UUID PRIMARY KEY REFERENCES businesses(id) ON DELETE CASCADE,
  inventory_type stay_inventory_type NOT NULL,
  base_price_minor BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT stay_details_base_price_check CHECK (base_price_minor IS NULL OR base_price_minor >= 0),
  CONSTRAINT stay_details_single_price_check CHECK (
    inventory_type <> 'single_unit' OR base_price_minor IS NOT NULL
  )
);

CREATE TRIGGER stay_details_set_updated_at
BEFORE UPDATE ON stay_details
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE stay_amenities (
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  amenity_code TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (business_id, amenity_code),
  CONSTRAINT stay_amenities_code_not_blank CHECK (length(btrim(amenity_code)) > 0)
);

-- A unit type is the provider-facing room/apartment configuration. Its
-- quantity is materialized as physical stay_units so bookings can later be
-- protected by an exclusion constraint per concrete unit.
CREATE TABLE stay_unit_types (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  max_guests SMALLINT NOT NULL,
  size_square_meters INTEGER NOT NULL,
  price_per_night_minor BIGINT NOT NULL,
  quantity SMALLINT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT stay_unit_types_name_not_blank CHECK (length(btrim(name)) > 0),
  CONSTRAINT stay_unit_types_max_guests_check CHECK (max_guests > 0),
  CONSTRAINT stay_unit_types_size_check CHECK (size_square_meters >= 0),
  CONSTRAINT stay_unit_types_price_check CHECK (price_per_night_minor >= 0),
  CONSTRAINT stay_unit_types_quantity_check CHECK (quantity > 0)
);

CREATE INDEX stay_unit_types_active_business_idx
  ON stay_unit_types (business_id, created_at DESC)
  WHERE deleted_at IS NULL AND is_active;

CREATE TRIGGER stay_unit_types_set_updated_at
BEFORE UPDATE ON stay_unit_types
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE stay_units (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  -- A single-unit stay owns one physical unit without a provider-facing unit
  -- type. Multiple-unit stays reference their provider-facing unit type.
  stay_unit_type_id UUID REFERENCES stay_unit_types(id) ON DELETE RESTRICT,
  sequence_number SMALLINT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  UNIQUE (stay_unit_type_id, sequence_number),
  CONSTRAINT stay_units_sequence_check CHECK (sequence_number > 0)
);

CREATE UNIQUE INDEX stay_units_one_single_unit_idx
  ON stay_units (business_id)
  WHERE stay_unit_type_id IS NULL AND deleted_at IS NULL;

CREATE INDEX stay_units_available_type_idx
  ON stay_units (stay_unit_type_id)
  WHERE deleted_at IS NULL AND is_active;

CREATE TRIGGER stay_units_set_updated_at
BEFORE UPDATE ON stay_units
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TYPE stay_booking_status AS ENUM ('confirmed', 'declined', 'cancelled', 'completed', 'no_show');
CREATE TYPE stay_payment_status AS ENUM ('pending', 'paid', 'refunded', 'failed');

-- Kept only for the one-time Firebase import. These mappings make the import
-- idempotent and preserve legacy booking business and room references.
CREATE TABLE legacy_firebase_business_ids (
  firebase_business_id TEXT PRIMARY KEY,
  business_id UUID NOT NULL UNIQUE REFERENCES businesses(id) ON DELETE CASCADE,
  migrated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE legacy_firebase_stay_unit_type_ids (
  firebase_room_id TEXT PRIMARY KEY,
  stay_unit_type_id UUID NOT NULL UNIQUE REFERENCES stay_unit_types(id) ON DELETE CASCADE,
  migrated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE stay_bookings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
  business_owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  customer_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  stay_unit_type_id UUID REFERENCES stay_unit_types(id) ON DELETE RESTRICT,
  customer_name TEXT NOT NULL,
  customer_email TEXT NOT NULL,
  customer_avatar_path TEXT,
  check_in DATE NOT NULL,
  check_out DATE NOT NULL,
  adults SMALLINT NOT NULL,
  children SMALLINT NOT NULL DEFAULT 0,
  infants SMALLINT NOT NULL DEFAULT 0,
  price_per_night_minor BIGINT NOT NULL,
  room_subtotal_minor BIGINT NOT NULL,
  discount_minor BIGINT NOT NULL DEFAULT 0,
  cleaning_fee_minor BIGINT NOT NULL DEFAULT 0,
  service_fee_minor BIGINT NOT NULL DEFAULT 0,
  taxes_minor BIGINT NOT NULL DEFAULT 0,
  total_minor BIGINT NOT NULL,
  original_total_minor BIGINT NOT NULL,
  payment_status stay_payment_status NOT NULL,
  payment_method TEXT NOT NULL,
  confirmation_code TEXT NOT NULL UNIQUE,
  currency CHAR(3) NOT NULL,
  status stay_booking_status NOT NULL DEFAULT 'confirmed',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT stay_bookings_dates_check CHECK (check_out > check_in),
  CONSTRAINT stay_bookings_guest_counts_check CHECK (adults > 0 AND children >= 0 AND infants >= 0),
  CONSTRAINT stay_bookings_amounts_check CHECK (price_per_night_minor >= 0 AND room_subtotal_minor >= 0 AND discount_minor >= 0 AND cleaning_fee_minor >= 0 AND service_fee_minor >= 0 AND taxes_minor >= 0 AND total_minor >= 0 AND original_total_minor >= 0),
  CONSTRAINT stay_bookings_customer_name_not_blank CHECK (length(btrim(customer_name)) > 0),
  CONSTRAINT stay_bookings_customer_email_not_blank CHECK (length(btrim(customer_email)) > 0),
  CONSTRAINT stay_bookings_payment_method_not_blank CHECK (length(btrim(payment_method)) > 0)
);

-- Range indexes make date-overlap checks in stay discovery index-backed rather
-- than requiring the Firebase-era candidate list to be filtered in memory.
CREATE INDEX stay_bookings_active_business_dates_idx
  ON stay_bookings USING GIST (business_id, daterange(check_in, check_out, '[)'))
  WHERE status = 'confirmed';
CREATE INDEX stay_bookings_active_unit_dates_idx
  ON stay_bookings USING GIST (stay_unit_type_id, daterange(check_in, check_out, '[)'))
  WHERE status = 'confirmed' AND stay_unit_type_id IS NOT NULL;
CREATE INDEX stay_bookings_customer_created_idx ON stay_bookings (customer_id, created_at DESC);
CREATE INDEX stay_bookings_owner_created_idx ON stay_bookings (business_owner_id, created_at DESC);

CREATE TRIGGER stay_bookings_set_updated_at
BEFORE UPDATE ON stay_bookings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

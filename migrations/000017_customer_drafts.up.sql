CREATE TABLE booking_drafts (
  customer_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
  check_in DATE NOT NULL,
  check_out DATE NOT NULL,
  adults SMALLINT NOT NULL,
  children SMALLINT NOT NULL DEFAULT 0,
  infants SMALLINT NOT NULL DEFAULT 0,
  room_type_id UUID REFERENCES stay_unit_types(id) ON DELETE SET NULL,
  selected_extras JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT booking_drafts_dates_check CHECK (check_out > check_in),
  CONSTRAINT booking_drafts_guest_counts_check CHECK (adults > 0 AND children >= 0 AND infants >= 0),
  CONSTRAINT booking_drafts_selected_extras_array_check CHECK (jsonb_typeof(selected_extras) = 'array')
);

CREATE TRIGGER booking_drafts_set_updated_at
BEFORE UPDATE ON booking_drafts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE appointment_drafts (
  customer_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
  selected_offering_ids UUID[] NOT NULL DEFAULT '{}',
  selected_provider_id UUID REFERENCES service_staff(id) ON DELETE SET NULL,
  appointment_date DATE NOT NULL,
  start_minutes SMALLINT,
  selected_add_on_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT appointment_drafts_start_minutes_check CHECK (start_minutes IS NULL OR (start_minutes >= 0 AND start_minutes < 1440)),
  CONSTRAINT appointment_drafts_selected_add_ons_array_check CHECK (jsonb_typeof(selected_add_on_ids) = 'array')
);

CREATE TRIGGER appointment_drafts_set_updated_at
BEFORE UPDATE ON appointment_drafts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

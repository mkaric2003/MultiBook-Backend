CREATE TABLE service_details (
  business_id UUID PRIMARY KEY REFERENCES businesses(id) ON DELETE CASCADE,
  time_zone TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT service_details_time_zone_not_blank CHECK (length(btrim(time_zone)) > 0)
);

CREATE TRIGGER service_details_set_updated_at
BEFORE UPDATE ON service_details FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE service_offerings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  duration_minutes SMALLINT NOT NULL,
  price_minor BIGINT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT service_offerings_name_not_blank CHECK (length(btrim(name)) > 0),
  CONSTRAINT service_offerings_duration_check CHECK (duration_minutes BETWEEN 5 AND 1440),
  CONSTRAINT service_offerings_price_check CHECK (price_minor >= 0)
);

CREATE INDEX service_offerings_active_business_idx
  ON service_offerings (business_id, created_at ASC)
  WHERE deleted_at IS NULL AND is_active;
CREATE TRIGGER service_offerings_set_updated_at
BEFORE UPDATE ON service_offerings FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE service_staff (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  title TEXT,
  commission_rate NUMERIC(5,2) NOT NULL DEFAULT 100,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT service_staff_name_not_blank CHECK (length(btrim(name)) > 0),
  CONSTRAINT service_staff_commission_check CHECK (commission_rate >= 0 AND commission_rate <= 100)
);

CREATE INDEX service_staff_active_business_idx
  ON service_staff (business_id, created_at ASC)
  WHERE deleted_at IS NULL AND is_active;
CREATE TRIGGER service_staff_set_updated_at
BEFORE UPDATE ON service_staff FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE service_staff_offerings (
  staff_id UUID NOT NULL REFERENCES service_staff(id) ON DELETE CASCADE,
  offering_id UUID NOT NULL REFERENCES service_offerings(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (staff_id, offering_id)
);

CREATE TABLE service_staff_weekly_availability (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  staff_id UUID NOT NULL REFERENCES service_staff(id) ON DELETE CASCADE,
  weekday SMALLINT NOT NULL,
  start_minutes SMALLINT NOT NULL,
  end_minutes SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT service_staff_weekly_availability_weekday_check CHECK (weekday BETWEEN 0 AND 6),
  CONSTRAINT service_staff_weekly_availability_minutes_check CHECK (
    start_minutes >= 0 AND end_minutes <= 1440 AND start_minutes < end_minutes
  ),
  EXCLUDE USING GIST (staff_id WITH =, weekday WITH =, int4range(start_minutes, end_minutes, '[)') WITH &&)
);

CREATE INDEX service_staff_weekly_availability_staff_idx ON service_staff_weekly_availability (staff_id, weekday);

CREATE TABLE service_staff_availability_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  staff_id UUID NOT NULL REFERENCES service_staff(id) ON DELETE CASCADE,
  blocked_range TSTZRANGE NOT NULL,
  reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT service_staff_availability_blocks_not_empty CHECK (NOT isempty(blocked_range)),
  EXCLUDE USING GIST (staff_id WITH =, blocked_range WITH &&)
);

CREATE INDEX service_staff_availability_blocks_range_idx
  ON service_staff_availability_blocks USING GIST (staff_id, blocked_range);

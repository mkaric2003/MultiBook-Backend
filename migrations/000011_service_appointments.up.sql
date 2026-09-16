CREATE TYPE appointment_status AS ENUM ('confirmed', 'declined', 'cancelled', 'completed', 'no_show');
CREATE TYPE appointment_payment_status AS ENUM ('pending', 'paid', 'refunded');

CREATE TABLE service_appointments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
  customer_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  staff_id UUID NOT NULL REFERENCES service_staff(id) ON DELETE RESTRICT,
  customer_name TEXT NOT NULL,
  customer_email TEXT NOT NULL,
  customer_phone TEXT NOT NULL,
  business_name TEXT NOT NULL,
  provider_name TEXT NOT NULL,
  appointment_date DATE NOT NULL,
  start_minutes SMALLINT NOT NULL,
  end_minutes SMALLINT NOT NULL,
  scheduled_range TSTZRANGE NOT NULL,
  service_cost_minor BIGINT NOT NULL,
  original_service_cost_minor BIGINT NOT NULL,
  discount_minor BIGINT NOT NULL DEFAULT 0,
  service_fee_minor BIGINT NOT NULL,
  taxes_minor BIGINT NOT NULL,
  total_minor BIGINT NOT NULL,
  provider_commission_rate NUMERIC(5,2) NOT NULL,
  provider_earnings_minor BIGINT NOT NULL,
  currency CHAR(3) NOT NULL,
  payment_status appointment_payment_status NOT NULL,
  payment_method TEXT NOT NULL,
  confirmation_code TEXT NOT NULL,
  status appointment_status NOT NULL DEFAULT 'confirmed',
  customer_reschedule_count SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT service_appointments_customer_name_not_blank CHECK (length(btrim(customer_name)) > 0),
  CONSTRAINT service_appointments_customer_email_not_blank CHECK (length(btrim(customer_email)) > 0),
  CONSTRAINT service_appointments_customer_phone_not_blank CHECK (length(btrim(customer_phone)) > 0),
  CONSTRAINT service_appointments_business_name_not_blank CHECK (length(btrim(business_name)) > 0),
  CONSTRAINT service_appointments_provider_name_not_blank CHECK (length(btrim(provider_name)) > 0),
  CONSTRAINT service_appointments_minutes_check CHECK (start_minutes >= 0 AND end_minutes <= 1440 AND start_minutes < end_minutes),
  CONSTRAINT service_appointments_range_not_empty CHECK (NOT isempty(scheduled_range)),
  CONSTRAINT service_appointments_costs_check CHECK (service_cost_minor >= 0 AND original_service_cost_minor >= 0 AND discount_minor >= 0 AND service_fee_minor >= 0 AND taxes_minor >= 0 AND total_minor >= 0 AND provider_earnings_minor >= 0),
  CONSTRAINT service_appointments_commission_check CHECK (provider_commission_rate >= 0 AND provider_commission_rate <= 100),
  CONSTRAINT service_appointments_currency_check CHECK (currency IN ('BAM', 'USD', 'EUR', 'CHF', 'GBP')),
  CONSTRAINT service_appointments_payment_method_not_blank CHECK (length(btrim(payment_method)) > 0),
  CONSTRAINT service_appointments_confirmation_code_not_blank CHECK (length(btrim(confirmation_code)) > 0),
  EXCLUDE USING GIST (staff_id WITH =, scheduled_range WITH &&) WHERE (status = 'confirmed')
);

CREATE INDEX service_appointments_business_date_idx ON service_appointments (business_id, appointment_date, start_minutes) WHERE status = 'confirmed';
CREATE INDEX service_appointments_customer_created_idx ON service_appointments (customer_id, created_at DESC);
CREATE INDEX service_appointments_staff_date_idx ON service_appointments (staff_id, appointment_date, start_minutes) WHERE status = 'confirmed';
CREATE TRIGGER service_appointments_set_updated_at BEFORE UPDATE ON service_appointments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE service_appointment_offerings (
  appointment_id UUID NOT NULL REFERENCES service_appointments(id) ON DELETE CASCADE,
  offering_id UUID NOT NULL REFERENCES service_offerings(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  duration_minutes SMALLINT NOT NULL,
  price_minor BIGINT NOT NULL,
  PRIMARY KEY (appointment_id, offering_id),
  CONSTRAINT service_appointment_offerings_name_not_blank CHECK (length(btrim(name)) > 0),
  CONSTRAINT service_appointment_offerings_duration_check CHECK (duration_minutes BETWEEN 5 AND 1440),
  CONSTRAINT service_appointment_offerings_price_check CHECK (price_minor >= 0)
);

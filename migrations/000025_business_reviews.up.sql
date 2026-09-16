ALTER TABLE service_appointments ADD COLUMN customer_avatar_path TEXT;

CREATE TABLE business_reviews (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
  business_owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  customer_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  customer_name TEXT NOT NULL,
  customer_avatar_path TEXT,
  stay_booking_id UUID REFERENCES stay_bookings(id) ON DELETE RESTRICT,
  service_appointment_id UUID REFERENCES service_appointments(id) ON DELETE RESTRICT,
  rating SMALLINT NOT NULL,
  comment TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT business_reviews_one_per_customer_business UNIQUE (business_id, customer_id),
  CONSTRAINT business_reviews_exactly_one_source CHECK (
    (stay_booking_id IS NOT NULL)::INTEGER +
    (service_appointment_id IS NOT NULL)::INTEGER = 1
  ),
  CONSTRAINT business_reviews_rating_check CHECK (rating BETWEEN 1 AND 5),
  CONSTRAINT business_reviews_comment_check CHECK (comment IS NULL OR char_length(comment) <= 1000)
);

CREATE UNIQUE INDEX business_reviews_stay_booking_idx
  ON business_reviews (stay_booking_id) WHERE stay_booking_id IS NOT NULL;
CREATE UNIQUE INDEX business_reviews_service_appointment_idx
  ON business_reviews (service_appointment_id) WHERE service_appointment_id IS NOT NULL;
CREATE INDEX business_reviews_business_created_idx
  ON business_reviews (business_id, created_at DESC, id DESC);

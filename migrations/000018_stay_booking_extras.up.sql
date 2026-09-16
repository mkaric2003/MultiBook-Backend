ALTER TABLE stay_bookings
  ADD COLUMN selected_extras JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE stay_bookings
  ADD CONSTRAINT stay_bookings_selected_extras_array_check
  CHECK (jsonb_typeof(selected_extras) = 'array');

ALTER TABLE stay_bookings DROP CONSTRAINT IF EXISTS stay_bookings_selected_extras_array_check;
ALTER TABLE stay_bookings DROP COLUMN IF EXISTS selected_extras;

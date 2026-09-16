DROP TABLE IF EXISTS service_staff_availability_blocks;
DROP TABLE IF EXISTS service_staff_weekly_availability;
DROP TABLE IF EXISTS service_staff_offerings;
DROP TRIGGER IF EXISTS service_staff_set_updated_at ON service_staff;
DROP TABLE IF EXISTS service_staff;
DROP TRIGGER IF EXISTS service_offerings_set_updated_at ON service_offerings;
DROP TABLE IF EXISTS service_offerings;
DROP TRIGGER IF EXISTS service_details_set_updated_at ON service_details;
DROP TABLE IF EXISTS service_details;

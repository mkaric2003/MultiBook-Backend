DROP TRIGGER IF EXISTS service_appointments_notify_metrics_changed ON service_appointments;
DROP TRIGGER IF EXISTS stay_bookings_notify_metrics_changed ON stay_bookings;
DROP FUNCTION IF EXISTS notify_business_metrics_changed();
DROP INDEX IF EXISTS service_appointments_dashboard_metrics_idx;
DROP INDEX IF EXISTS stay_bookings_dashboard_metrics_idx;

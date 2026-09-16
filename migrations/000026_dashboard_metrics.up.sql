-- Dashboard metrics are calculated from reservation source-of-truth rows.
-- These partial indexes keep current-month aggregation index-backed without
-- introducing a second aggregate table that can drift from reservations.
CREATE INDEX stay_bookings_dashboard_metrics_idx
  ON stay_bookings (business_id, created_at)
  INCLUDE (total_minor, payment_status, payment_method)
  WHERE status IN ('confirmed', 'completed');

CREATE INDEX service_appointments_dashboard_metrics_idx
  ON service_appointments (business_id, created_at)
  INCLUDE (total_minor, payment_status, payment_method)
  WHERE status IN ('confirmed', 'completed');

-- PostgreSQL delivers NOTIFY messages only after the surrounding transaction
-- commits. The API treats them as invalidations and reloads the full snapshot,
-- so dropped/coalesced notifications cannot corrupt metric values.
CREATE FUNCTION notify_business_metrics_changed() RETURNS TRIGGER
LANGUAGE plpgsql
SET search_path = public
AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    PERFORM pg_notify('business_metrics_changed', OLD.business_id::TEXT);
  ELSE
    PERFORM pg_notify('business_metrics_changed', NEW.business_id::TEXT);
    IF TG_OP = 'UPDATE' AND OLD.business_id IS DISTINCT FROM NEW.business_id THEN
      PERFORM pg_notify('business_metrics_changed', OLD.business_id::TEXT);
    END IF;
  END IF;
  RETURN NULL;
END;
$$;

CREATE TRIGGER stay_bookings_notify_metrics_changed
AFTER INSERT OR UPDATE OR DELETE ON stay_bookings
FOR EACH ROW EXECUTE FUNCTION notify_business_metrics_changed();

CREATE TRIGGER service_appointments_notify_metrics_changed
AFTER INSERT OR UPDATE OR DELETE ON service_appointments
FOR EACH ROW EXECUTE FUNCTION notify_business_metrics_changed();

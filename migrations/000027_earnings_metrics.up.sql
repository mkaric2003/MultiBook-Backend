CREATE INDEX service_appointments_staff_earnings_metrics_idx
  ON service_appointments (business_id, staff_id, created_at)
  INCLUDE (total_minor, provider_earnings_minor, payment_status, payment_method)
  WHERE status IN ('confirmed', 'completed');

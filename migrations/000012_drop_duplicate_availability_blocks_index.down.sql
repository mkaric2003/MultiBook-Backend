CREATE INDEX service_staff_availability_blocks_range_idx
  ON service_staff_availability_blocks USING GIST (staff_id, blocked_range);

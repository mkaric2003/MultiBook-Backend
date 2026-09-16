-- The exclusion constraint on service_staff_availability_blocks already owns
-- a GiST index over (staff_id, blocked_range). This separately-created index
-- has identical access-path semantics and only adds write/storage overhead.
DROP INDEX IF EXISTS service_staff_availability_blocks_range_idx;

-- Physical units are soft-deleted when a provider reduces quantity. The
-- original table-level unique constraint prevented a later quantity increase
-- from recreating the same sequence number, so uniqueness must apply only to
-- currently materialized units.
ALTER TABLE stay_units
  DROP CONSTRAINT stay_units_stay_unit_type_id_sequence_number_key;

CREATE UNIQUE INDEX stay_units_active_type_sequence_idx
  ON stay_units (stay_unit_type_id, sequence_number)
  WHERE deleted_at IS NULL AND stay_unit_type_id IS NOT NULL;

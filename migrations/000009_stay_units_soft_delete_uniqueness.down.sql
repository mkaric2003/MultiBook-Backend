DROP INDEX IF EXISTS stay_units_active_type_sequence_idx;

ALTER TABLE stay_units
  ADD CONSTRAINT stay_units_stay_unit_type_id_sequence_number_key
  UNIQUE (stay_unit_type_id, sequence_number);

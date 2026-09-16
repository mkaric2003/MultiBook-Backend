DROP INDEX IF EXISTS users_active_city_normalized_idx;

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_business_currency_check,
  DROP COLUMN IF EXISTS business_currency,
  DROP COLUMN IF EXISTS city_normalized,
  DROP COLUMN IF EXISTS city,
  DROP COLUMN IF EXISTS address,
  DROP COLUMN IF EXISTS date_of_birth,
  DROP COLUMN IF EXISTS country_code,
  DROP COLUMN IF EXISTS last_name,
  DROP COLUMN IF EXISTS first_name;

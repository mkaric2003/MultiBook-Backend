ALTER TABLE users
  ADD COLUMN first_name TEXT,
  ADD COLUMN last_name TEXT,
  ADD COLUMN country_code TEXT,
  ADD COLUMN date_of_birth DATE,
  ADD COLUMN address TEXT,
  ADD COLUMN city TEXT,
  ADD COLUMN city_normalized TEXT,
  ADD COLUMN business_currency CHAR(3) NOT NULL DEFAULT 'BAM',
  ADD CONSTRAINT users_business_currency_check CHECK (business_currency IN ('BAM', 'USD', 'EUR', 'CHF', 'GBP'));

CREATE INDEX users_active_city_normalized_idx ON users (city_normalized) WHERE deleted_at IS NULL;

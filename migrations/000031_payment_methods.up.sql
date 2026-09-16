CREATE TYPE saved_card_brand AS ENUM ('visa', 'mastercard', 'amex', 'other');

CREATE TABLE customer_payment_methods (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  brand saved_card_brand NOT NULL,
  last4 CHAR(4) NOT NULL,
  expiry_month SMALLINT NOT NULL,
  expiry_year SMALLINT NOT NULL,
  holder_name TEXT NOT NULL,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT customer_payment_methods_last4_check CHECK (last4 ~ '^[0-9]{4}$'),
  CONSTRAINT customer_payment_methods_expiry_month_check CHECK (expiry_month BETWEEN 1 AND 12),
  CONSTRAINT customer_payment_methods_expiry_year_check CHECK (expiry_year BETWEEN 2000 AND 2200),
  CONSTRAINT customer_payment_methods_holder_name_check CHECK (length(trim(holder_name)) BETWEEN 1 AND 120)
);

CREATE UNIQUE INDEX customer_payment_methods_one_default_idx
  ON customer_payment_methods (customer_id)
  WHERE is_default;

CREATE INDEX customer_payment_methods_customer_created_idx
  ON customer_payment_methods (customer_id, created_at DESC, id DESC);

CREATE TRIGGER customer_payment_methods_set_updated_at
BEFORE UPDATE ON customer_payment_methods
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

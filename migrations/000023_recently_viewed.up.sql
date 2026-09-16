CREATE TABLE recently_viewed_businesses (
  customer_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  viewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (customer_id, business_id)
);
CREATE INDEX recently_viewed_businesses_customer_viewed_idx ON recently_viewed_businesses(customer_id, viewed_at DESC);

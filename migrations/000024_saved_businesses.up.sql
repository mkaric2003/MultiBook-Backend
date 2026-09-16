CREATE TABLE saved_businesses (
 customer_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
 saved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY (customer_id, business_id)
);
CREATE INDEX saved_businesses_customer_saved_idx ON saved_businesses(customer_id, saved_at DESC, business_id);

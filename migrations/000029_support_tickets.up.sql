CREATE TYPE support_ticket_category AS ENUM (
  'account',
  'booking',
  'appointment',
  'payment',
  'technical',
  'other'
);

CREATE TYPE support_ticket_status AS ENUM ('open', 'inProgress', 'resolved');

CREATE TABLE support_tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  customer_name TEXT NOT NULL,
  customer_email CITEXT NOT NULL,
  category support_ticket_category NOT NULL,
  status support_ticket_status NOT NULL DEFAULT 'open',
  subject TEXT NOT NULL,
  message TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT support_tickets_customer_name_not_blank CHECK (length(trim(customer_name)) > 0),
  CONSTRAINT support_tickets_customer_email_not_blank CHECK (length(trim(customer_email::TEXT)) > 0),
  CONSTRAINT support_tickets_subject_check CHECK (length(trim(subject)) BETWEEN 1 AND 160),
  CONSTRAINT support_tickets_message_check CHECK (length(trim(message)) BETWEEN 1 AND 5000)
);

CREATE INDEX support_tickets_customer_created_idx
  ON support_tickets (customer_id, created_at DESC, id DESC);

CREATE INDEX support_tickets_status_created_idx
  ON support_tickets (status, created_at ASC, id ASC);

CREATE TRIGGER support_tickets_set_updated_at
BEFORE UPDATE ON support_tickets
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

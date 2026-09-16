CREATE TYPE chat_participant_role AS ENUM ('customer', 'business');
CREATE TYPE chat_push_status AS ENUM ('pending', 'processing', 'sent', 'skipped', 'failed');

CREATE TABLE chat_conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID REFERENCES businesses(id) ON DELETE RESTRICT,
  legacy_business_id TEXT,
  business_owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  customer_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  business_name_snapshot TEXT NOT NULL,
  business_image_path_snapshot TEXT,
  customer_name_snapshot TEXT NOT NULL,
  customer_image_path_snapshot TEXT,
  last_message_id UUID,
  last_message_text TEXT NOT NULL DEFAULT '',
  last_message_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chat_conversations_business_reference_check CHECK (
    business_id IS NOT NULL OR length(btrim(legacy_business_id)) > 0
  ),
  CONSTRAINT chat_conversations_distinct_participants_check CHECK (business_owner_id <> customer_id),
  CONSTRAINT chat_conversations_business_name_not_blank CHECK (length(btrim(business_name_snapshot)) > 0),
  CONSTRAINT chat_conversations_customer_name_not_blank CHECK (length(btrim(customer_name_snapshot)) > 0),
  CONSTRAINT chat_conversations_last_message_text_length_check CHECK (char_length(last_message_text) <= 4000),
  CONSTRAINT chat_conversations_last_message_shape_check CHECK (
    (last_message_id IS NULL AND last_message_at IS NULL AND last_message_text = '')
    OR (last_message_id IS NOT NULL AND last_message_at IS NOT NULL)
  )
);

CREATE UNIQUE INDEX chat_conversations_business_customer_idx
  ON chat_conversations (business_id, customer_id)
  WHERE business_id IS NOT NULL;
CREATE UNIQUE INDEX chat_conversations_legacy_customer_idx
  ON chat_conversations (legacy_business_id, customer_id)
  WHERE business_id IS NULL;
CREATE INDEX chat_conversations_customer_list_idx
  ON chat_conversations (customer_id, COALESCE(last_message_at, created_at) DESC, id DESC);
CREATE INDEX chat_conversations_owner_list_idx
  ON chat_conversations (business_owner_id, COALESCE(last_message_at, created_at) DESC, id DESC);

CREATE TRIGGER chat_conversations_set_updated_at
BEFORE UPDATE ON chat_conversations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE chat_participant_state (
  conversation_id UUID NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  participant_role chat_participant_role NOT NULL,
  unread_count INTEGER NOT NULL DEFAULT 0,
  last_read_message_id UUID,
  last_read_at TIMESTAMPTZ,
  typing_until TIMESTAMPTZ,
  active_until TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (conversation_id, user_id),
  UNIQUE (conversation_id, participant_role),
  CONSTRAINT chat_participant_state_unread_count_check CHECK (unread_count >= 0),
  CONSTRAINT chat_participant_state_read_shape_check CHECK (
    (last_read_message_id IS NULL AND last_read_at IS NULL)
    OR (last_read_message_id IS NOT NULL AND last_read_at IS NOT NULL)
  )
);

CREATE INDEX chat_participant_state_user_unread_idx
  ON chat_participant_state (user_id)
  WHERE unread_count > 0;

CREATE TRIGGER chat_participant_state_set_updated_at
BEFORE UPDATE ON chat_participant_state
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE chat_messages (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
  sender_id TEXT NOT NULL,
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chat_messages_sender_participant_fkey
    FOREIGN KEY (conversation_id, sender_id)
    REFERENCES chat_participant_state(conversation_id, user_id)
    ON DELETE RESTRICT,
  CONSTRAINT chat_messages_text_not_blank CHECK (length(btrim(text)) > 0),
  CONSTRAINT chat_messages_text_length_check CHECK (char_length(text) <= 4000)
);

CREATE INDEX chat_messages_conversation_page_idx
  ON chat_messages (conversation_id, created_at DESC, id DESC);

ALTER TABLE chat_messages
  ADD CONSTRAINT chat_messages_conversation_id_id_key UNIQUE (conversation_id, id);

ALTER TABLE chat_conversations
  ADD CONSTRAINT chat_conversations_last_message_fkey
  FOREIGN KEY (id, last_message_id) REFERENCES chat_messages(conversation_id, id)
  DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE chat_participant_state
  ADD CONSTRAINT chat_participant_state_last_read_message_fkey
  FOREIGN KEY (conversation_id, last_read_message_id) REFERENCES chat_messages(conversation_id, id)
  DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE chat_push_outbox (
  message_id UUID PRIMARY KEY REFERENCES chat_messages(id) ON DELETE CASCADE,
  recipient_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status chat_push_status NOT NULL DEFAULT 'pending',
  attempts SMALLINT NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  processing_started_at TIMESTAMPTZ,
  sent_at TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chat_push_outbox_attempts_check CHECK (attempts >= 0),
  CONSTRAINT chat_push_outbox_error_length_check CHECK (char_length(last_error) <= 1000)
);

CREATE INDEX chat_push_outbox_pending_idx
  ON chat_push_outbox (available_at, created_at)
  WHERE status IN ('pending', 'failed');
CREATE INDEX chat_push_outbox_processing_idx
  ON chat_push_outbox (processing_started_at)
  WHERE status = 'processing';

CREATE TRIGGER chat_push_outbox_set_updated_at
BEFORE UPDATE ON chat_push_outbox
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- The payload deliberately contains only identifiers. API listeners reload
-- authorized state from PostgreSQL and message text never enters NOTIFY logs.
CREATE FUNCTION notify_chat_changed() RETURNS TRIGGER
LANGUAGE plpgsql
SET search_path = public
AS $$
DECLARE
  changed_conversation_id UUID;
  changed_customer_id TEXT;
  changed_business_owner_id TEXT;
BEGIN
  IF TG_TABLE_NAME = 'chat_conversations' THEN
    changed_conversation_id := COALESCE(NEW.id, OLD.id);
    changed_customer_id := COALESCE(NEW.customer_id, OLD.customer_id);
    changed_business_owner_id := COALESCE(NEW.business_owner_id, OLD.business_owner_id);
  ELSE
    changed_conversation_id := COALESCE(NEW.conversation_id, OLD.conversation_id);
    SELECT customer_id,business_owner_id
      INTO changed_customer_id,changed_business_owner_id
      FROM chat_conversations WHERE id=changed_conversation_id;
  END IF;
  PERFORM pg_notify('chat_changed', json_build_object(
    'conversationId', changed_conversation_id,
    'participantIds', json_build_array(changed_customer_id, changed_business_owner_id)
  )::TEXT);
  RETURN NULL;
END;
$$;

CREATE TRIGGER chat_conversations_notify_changed
AFTER INSERT OR UPDATE ON chat_conversations
FOR EACH ROW EXECUTE FUNCTION notify_chat_changed();

CREATE TRIGGER chat_messages_notify_changed
AFTER INSERT ON chat_messages
FOR EACH ROW EXECUTE FUNCTION notify_chat_changed();

CREATE TRIGGER chat_participant_state_notify_changed
AFTER INSERT OR UPDATE ON chat_participant_state
FOR EACH ROW EXECUTE FUNCTION notify_chat_changed();

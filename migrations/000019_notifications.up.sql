CREATE TABLE notification_devices (
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  device_id TEXT NOT NULL,
  token TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, device_id)
);

CREATE INDEX notification_devices_enabled_user_idx ON notification_devices(user_id) WHERE enabled;

CREATE TABLE in_app_notifications (
  id TEXT PRIMARY KEY,
  recipient_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  read_at TIMESTAMPTZ
);

CREATE INDEX in_app_notifications_recipient_created_idx ON in_app_notifications(recipient_id, created_at DESC, id DESC);
CREATE INDEX in_app_notifications_recipient_unread_idx ON in_app_notifications(recipient_id) WHERE read_at IS NULL;

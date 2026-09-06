-- +goose Up
CREATE TABLE user_settings (
 user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 theme TEXT NOT NULL DEFAULT 'system' CHECK(theme IN ('light','dark','system')),
 notifications BOOLEAN NOT NULL DEFAULT true,
 typing BOOLEAN NOT NULL DEFAULT true,
 read_receipts BOOLEAN NOT NULL DEFAULT true
);
CREATE TABLE messages (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
 sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 client_id UUID NOT NULL,
 body TEXT NOT NULL CHECK(char_length(body) BETWEEN 1 AND 4000),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(sender_id,client_id)
);
CREATE INDEX messages_connection_order ON messages(connection_id,id DESC);
CREATE TABLE message_receipts (
 connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 delivered_id BIGINT NOT NULL DEFAULT 0,
 read_id BIGINT NOT NULL DEFAULT 0,
 PRIMARY KEY(connection_id,user_id)
);
CREATE TABLE notifications (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
 kind TEXT NOT NULL CHECK(kind IN ('message','call','connection')),
 reference TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 read_at TIMESTAMPTZ,
 UNIQUE(user_id,kind,reference)
);
CREATE INDEX notifications_owner_order ON notifications(user_id,id DESC);

-- +goose StatementBegin
CREATE FUNCTION notify_connection() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO notifications(user_id,connection_id,kind,reference)
 VALUES(NEW.user_low,NEW.id,'connection',NEW.id::text),(NEW.user_high,NEW.id,'connection',NEW.id::text)
 ON CONFLICT DO NOTHING;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER connection_notification AFTER INSERT ON connections FOR EACH ROW EXECUTE FUNCTION notify_connection();

-- +goose Down
DROP TRIGGER connection_notification ON connections;
DROP FUNCTION notify_connection();
DROP TABLE notifications;
DROP TABLE message_receipts;
DROP TABLE messages;
DROP TABLE user_settings;

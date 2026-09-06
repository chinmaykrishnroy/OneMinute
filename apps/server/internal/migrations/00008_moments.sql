-- +goose Up
CREATE TABLE moments (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 body TEXT NOT NULL CHECK(char_length(body) BETWEEN 1 AND 600),
 tone TEXT NOT NULL DEFAULT 'lilac' CHECK(tone IN ('lilac','mint','sand')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 expires_at TIMESTAMPTZ NOT NULL DEFAULT now() + interval '24 hours'
);
CREATE INDEX moments_expiry ON moments(expires_at);
CREATE INDEX moments_owner ON moments(user_id,created_at DESC);
CREATE TABLE moment_audience (
 moment_id UUID NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
 connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
 PRIMARY KEY(moment_id,connection_id)
);
CREATE INDEX moment_audience_connection ON moment_audience(connection_id,moment_id);
-- +goose Down
DROP TABLE moment_audience;
DROP TABLE moments;

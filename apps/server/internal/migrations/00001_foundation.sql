-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'banned', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ
);

-- Profile data belongs in PostgreSQL; Redis must never own it.
CREATE TABLE profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 80),
    avatar_url TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
    country_code TEXT CHECK (country_code ~ '^[A-Z]{2}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE profiles;
DROP TABLE users;
-- Do not drop a potentially shared extension on rollback.

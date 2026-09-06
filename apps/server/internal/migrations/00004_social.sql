-- +goose Up
CREATE TABLE profile_interests (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    interest TEXT NOT NULL CHECK (interest ~ '^[a-z][a-z0-9_]{0,39}$'),
    PRIMARY KEY (user_id, interest)
);

CREATE TABLE profile_languages (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language_code TEXT NOT NULL CHECK (language_code ~ '^[a-z]{2,3}(-[A-Z]{2})?$'),
    PRIMARY KEY (user_id, language_code)
);

CREATE TABLE encounters (
    id UUID PRIMARY KEY,
    user_low UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    user_high UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    intent TEXT NOT NULL CHECK (char_length(intent) BETWEEN 1 AND 40),
    shared_interests TEXT[] NOT NULL DEFAULT '{}',
    outcome TEXT NOT NULL CHECK (outcome IN ('connected', 'reported')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (user_low < user_high),
    CHECK (ended_at >= started_at)
);

CREATE TABLE connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_low UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    user_high UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    encounter_id UUID NOT NULL UNIQUE REFERENCES encounters(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    CHECK (user_low < user_high)
);
CREATE UNIQUE INDEX connections_active_pair_idx ON connections(user_low, user_high) WHERE ended_at IS NULL;
CREATE INDEX connections_user_high_idx ON connections(user_high, user_low) WHERE ended_at IS NULL;

CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reported_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    encounter_id UUID REFERENCES encounters(id) ON DELETE RESTRICT,
    connection_id UUID REFERENCES connections(id) ON DELETE RESTRICT,
    category TEXT NOT NULL CHECK (category IN ('spam','harassment','sexual_content','hate','violence','underage','other')),
    details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (reporter_user_id <> reported_user_id),
    CHECK (encounter_id IS NOT NULL OR connection_id IS NOT NULL)
);
CREATE UNIQUE INDEX reports_encounter_reporter_idx ON reports(reporter_user_id, encounter_id) WHERE encounter_id IS NOT NULL;
CREATE INDEX reports_reported_user_idx ON reports(reported_user_id, created_at DESC);

-- +goose Down
DROP TABLE reports;
DROP TABLE connections;
DROP TABLE encounters;
DROP TABLE profile_languages;
DROP TABLE profile_interests;

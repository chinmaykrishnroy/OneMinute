-- +goose Up
ALTER TABLE profiles ADD COLUMN discovery_intent TEXT NOT NULL DEFAULT 'surprise_me'
    CHECK (discovery_intent IN ('surprise_me','new_friends','dating','gaming','language_exchange','tech_ideas','professional_networking'));

-- +goose Down
ALTER TABLE profiles DROP COLUMN discovery_intent;

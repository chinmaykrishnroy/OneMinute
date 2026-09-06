-- +goose Up
ALTER TABLE reports
    ADD CONSTRAINT reports_single_context CHECK (num_nonnulls(encounter_id, connection_id) = 1);
CREATE UNIQUE INDEX reports_connection_reporter_idx
    ON reports(reporter_user_id, connection_id) WHERE connection_id IS NOT NULL;

-- +goose Down
DROP INDEX reports_connection_reporter_idx;
ALTER TABLE reports DROP CONSTRAINT reports_single_context;

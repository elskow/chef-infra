-- +goose Up
-- +goose StatementBegin
-- Remove columns related to account locking
ALTER TABLE users
    DROP COLUMN IF EXISTS failed_login_count,
    DROP COLUMN IF EXISTS last_login_attempt,
    DROP COLUMN IF EXISTS locked,
    DROP COLUMN IF EXISTS lock_until;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Restore columns related to account locking
ALTER TABLE users
    ADD COLUMN failed_login_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN last_login_attempt TIMESTAMP,
    ADD COLUMN locked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN lock_until TIMESTAMP;
-- +goose StatementEnd
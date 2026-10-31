-- +goose Up
ALTER TABLE app_user ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE app_user DROP COLUMN updated_at;

-- +goose Up
ALTER TABLE app_user ADD COLUMN staff_no TEXT NOT NULL DEFAULT '';
UPDATE app_user SET staff_no = '0001' WHERE username = 'admin';

-- +goose Down
ALTER TABLE app_user DROP COLUMN staff_no;

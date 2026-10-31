-- +goose Up
-- Area tables are modelled as batch items + labels; expose labels and a scan screen instead.
UPDATE menu SET title = '标签管理', path = '/production/labels' WHERE id = 401;
INSERT INTO menu (id, parent_id, title, path, icon, sort) VALUES (403, 400, '扫码报工', '/production/scan', '', 3);

-- +goose Down
DELETE FROM menu WHERE id = 403;
UPDATE menu SET title = '面积表查询', path = '/production/area-tables' WHERE id = 401;

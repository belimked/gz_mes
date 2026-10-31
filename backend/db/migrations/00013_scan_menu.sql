-- +goose Up
INSERT INTO menu (id, parent_id, title, path, icon, sort) VALUES
    (1004, 1000, '扫描记录', '/report/scans', 'barcode', 4);
-- Roles that can open 标签管理 (production data) also get the scan list.
INSERT INTO role_menu (role_id, menu_id)
SELECT rm.role_id, 1004 FROM role_menu rm WHERE rm.menu_id = 401
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id = 1004;
DELETE FROM menu WHERE id = 1004;

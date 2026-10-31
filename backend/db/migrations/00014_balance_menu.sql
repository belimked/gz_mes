-- +goose Up
INSERT INTO menu (id, parent_id, title, path, icon, sort) VALUES
    (1005, 1000, '生产平衡表', '/report/production-balance', 'calculator', 5);
INSERT INTO role_menu (role_id, menu_id)
SELECT rm.role_id, 1005 FROM role_menu rm WHERE rm.menu_id = 401
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id = 1005;
DELETE FROM menu WHERE id = 1005;

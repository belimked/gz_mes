-- +goose Up
INSERT INTO menu (id, parent_id, title, path, icon, sort) VALUES
    (1000, NULL, '生产报表', '/report',                'bar-chart',  100),
    (1001, 1000, '工程日报', '/report/batch-daily',    'file-text',  1),
    (1002, 1000, '工程进度', '/report/batch-progress', 'line-chart', 2),
    (1003, 1000, '员工产量', '/report/staff-output',   'team',       3);
-- Roles that can open 标签管理 (production data) also get the report screens.
INSERT INTO role_menu (role_id, menu_id)
SELECT rm.role_id, m.id FROM role_menu rm CROSS JOIN (VALUES (1001), (1002), (1003)) AS m (id)
WHERE rm.menu_id = 401
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (1001, 1002, 1003);
DELETE FROM menu WHERE id IN (1001, 1002, 1003, 1000);

-- +goose Up
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES
 (1103,1100,'工单执行看板','/board/work-orders','dashboard',3),
 (1104,1100,'办公室驾驶舱','/board/cockpit','dashboard',4);
INSERT INTO role_menu(role_id,menu_id)
 SELECT DISTINCT rm.role_id,m.id FROM role_menu rm CROSS JOIN (VALUES (1103),(1104)) m(id)
 WHERE rm.menu_id=1101 ON CONFLICT DO NOTHING;
-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (1103,1104);
DELETE FROM menu WHERE id IN (1103,1104);

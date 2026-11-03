-- +goose Up
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES
 (1100,NULL,'看板管理','/board','dashboard',5),
 (1101,1100,'喷涂日排产看板','/board/spray-daily','dashboard',1),
 (1102,1100,'钣金日排程看板','/board/sheet-daily','dashboard',2);
INSERT INTO role_menu(role_id,menu_id)
 SELECT rm.role_id,m.id FROM role_menu rm CROSS JOIN (VALUES (1100),(1101),(1102)) m(id)
 WHERE rm.menu_id=301 ON CONFLICT DO NOTHING;
-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (1100,1101,1102);
DELETE FROM menu WHERE id IN (1101,1102,1100);

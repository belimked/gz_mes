-- +goose Up
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES
 (1010,1000,'项目汇总','/report/project-summary','bar-chart',10),
 (1300,NULL,'库存管理','/inventory','database',110),
 (1301,1300,'半成品库存','/inventory/semi','',1),
 (1302,1300,'成品库存','/inventory/finished','',2);
INSERT INTO role_menu(role_id,menu_id)
 SELECT rm.role_id,m.id FROM role_menu rm CROSS JOIN (VALUES (1010),(1300),(1301),(1302)) m(id)
 WHERE rm.menu_id=1004 ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (1010,1300,1301,1302);
DELETE FROM menu WHERE id IN (1010,1301,1302,1300);

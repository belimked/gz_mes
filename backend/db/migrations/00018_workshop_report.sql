-- +goose Up
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES
 (1008,1000,'车间产量日报','/report/workshop-daily','bar-chart',8),
 (1009,1000,'车间产量月报','/report/workshop-monthly','bar-chart',9);
INSERT INTO role_menu(role_id,menu_id) SELECT role_id,1008 FROM role_menu WHERE menu_id=401 ON CONFLICT DO NOTHING;
INSERT INTO role_menu(role_id,menu_id) SELECT role_id,1009 FROM role_menu WHERE menu_id=401 ON CONFLICT DO NOTHING;
-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (1008,1009);
DELETE FROM menu WHERE id IN (1008,1009);

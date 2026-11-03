-- +goose Up
CREATE TABLE dict_item (
 id BIGSERIAL PRIMARY KEY,
 kind TEXT NOT NULL CHECK (kind IN ('workshop','paint_line','shift')),
 name TEXT NOT NULL CHECK (name = btrim(name) AND length(name) BETWEEN 1 AND 50),
 sort INT NOT NULL DEFAULT 0,
 start_time TIME,
 end_time TIME,
 is_stop BOOL NOT NULL DEFAULT false,
 remark TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(kind,name),
 CHECK ((start_time IS NULL AND end_time IS NULL) OR (kind='shift' AND start_time IS NOT NULL AND end_time IS NOT NULL))
);
INSERT INTO dict_item(kind,name,sort,start_time,end_time) VALUES
 ('workshop','钣金一车间',1,NULL,NULL),('workshop','钣金二车间',2,NULL,NULL),
 ('paint_line','1号线',1,NULL,NULL),('paint_line','2号线',2,NULL,NULL),
 ('shift','早班',1,'08:00','17:00'),('shift','中班',2,'16:00','00:00'),('shift','晚班',3,'00:00','08:00');
INSERT INTO dict_item(kind,name)
 SELECT kind,btrim(name) FROM (
 SELECT 'workshop' AS kind,workshop AS name FROM batch
 UNION SELECT 'workshop',workshop FROM plan
 UNION SELECT 'paint_line',paint_line FROM plan
 UNION SELECT 'shift',shift FROM plan
 ) existing WHERE btrim(name) <> '' AND length(btrim(name)) <= 50
 ON CONFLICT DO NOTHING;
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES (104,100,'基础字典','/system/dicts','book',4);
INSERT INTO role_menu(role_id,menu_id) SELECT role_id,104 FROM role_menu WHERE menu_id=103 ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id=104;
DELETE FROM menu WHERE id=104;
DROP TABLE dict_item;

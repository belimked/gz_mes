-- +goose Up
ALTER TABLE role ADD COLUMN is_personal BOOL NOT NULL DEFAULT false,
 ADD COLUMN remark TEXT NOT NULL DEFAULT '',
 ADD COLUMN is_stop BOOL NOT NULL DEFAULT false,
 ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE role SET is_personal=true WHERE code LIKE 'user:%';
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES (105,100,'角色管理','/system/roles','team',5);
INSERT INTO role_menu(role_id,menu_id) SELECT role_id,105 FROM role_menu WHERE menu_id=102 ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id=105;
DELETE FROM menu WHERE id=105;
ALTER TABLE role DROP COLUMN is_personal, DROP COLUMN remark, DROP COLUMN is_stop, DROP COLUMN created_at;

-- +goose Up
CREATE TABLE label_log (
 id BIGSERIAL PRIMARY KEY, label_id BIGINT NOT NULL REFERENCES label(id) ON DELETE CASCADE,
 op_type TEXT NOT NULL CHECK (op_type IN ('generate','cancel','reprint')),
 reason TEXT NOT NULL DEFAULT '', operator_id BIGINT NOT NULL REFERENCES app_user(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX label_log_created_idx ON label_log(created_at DESC);
CREATE INDEX label_log_label_idx ON label_log(label_id);
INSERT INTO label_log(label_id,op_type,operator_id,created_at) SELECT id,'generate',created_by,created_at FROM label;
INSERT INTO label_log(label_id,op_type,reason,operator_id,created_at) SELECT id,'cancel','迁移补录',created_by,created_at FROM label WHERE cancelled;
CREATE TABLE audit_log (
 id BIGSERIAL PRIMARY KEY, user_id BIGINT REFERENCES app_user(id) ON DELETE SET NULL,
 username TEXT NOT NULL DEFAULT '', display_name TEXT NOT NULL DEFAULT '',
 method TEXT NOT NULL, route TEXT NOT NULL, path TEXT NOT NULL,
 status INT NOT NULL, duration_ms INT NOT NULL, ip TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_created_idx ON audit_log(created_at DESC);
CREATE INDEX audit_log_user_created_idx ON audit_log(user_id,created_at DESC);
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES (404,400,'标签日志','/production/label-logs','history',4),(106,100,'操作日志','/system/audit-logs','audit',6);
INSERT INTO role_menu(role_id,menu_id) SELECT role_id,404 FROM role_menu WHERE menu_id=401 ON CONFLICT DO NOTHING;
INSERT INTO role_menu(role_id,menu_id) SELECT role_id,106 FROM role_menu WHERE menu_id=102 ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (404,106);
DELETE FROM menu WHERE id IN (404,106);
DROP TABLE audit_log;
DROP TABLE label_log;

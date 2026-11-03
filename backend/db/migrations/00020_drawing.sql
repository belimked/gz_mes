-- +goose Up
CREATE TABLE drawing (
 id BIGSERIAL PRIMARY KEY,
 batch_id BIGINT NOT NULL REFERENCES batch(id) ON DELETE CASCADE,
 kind TEXT NOT NULL CHECK (kind IN ('unfold','forming','nesting')),
 drawing_no TEXT NOT NULL DEFAULT '', remark TEXT NOT NULL DEFAULT '',
 original_name TEXT NOT NULL, content_type TEXT NOT NULL, size_bytes BIGINT NOT NULL,
 sha256 TEXT NOT NULL, storage_key TEXT NOT NULL UNIQUE,
 uploaded_by BIGINT NOT NULL REFERENCES app_user(id), uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX drawing_batch_kind_idx ON drawing(batch_id,kind);
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES
 (1200,NULL,'图纸管理','/drawing','radar-chart',6),
 (1201,1200,'图纸查询','/drawing/list','file-search',1);
INSERT INTO role_menu(role_id,menu_id)
 SELECT rm.role_id,m.id FROM role_menu rm CROSS JOIN (VALUES (1200),(1201)) m(id)
 WHERE rm.menu_id=301 ON CONFLICT DO NOTHING;
-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (1200,1201);
DELETE FROM menu WHERE id IN (1201,1200);
DROP TABLE drawing;

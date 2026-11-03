-- +goose Up
CREATE TABLE machine (
 id BIGSERIAL PRIMARY KEY,
 factory_id BIGINT NOT NULL REFERENCES factory(id),
 code TEXT NOT NULL,
 name TEXT NOT NULL,
 job_id BIGINT NOT NULL REFERENCES job(id),
 capacity_qty INT NOT NULL DEFAULT 0 CHECK (capacity_qty >= 0),
 capacity_area NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (capacity_area >= 0),
 is_stop BOOL NOT NULL DEFAULT false,
 remark TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(factory_id,code)
);
ALTER TABLE scan_record ADD COLUMN machine_id BIGINT REFERENCES machine(id);
CREATE INDEX scan_record_machine_scanned_idx ON scan_record(machine_id,scanned_at);
INSERT INTO menu(id,parent_id,title,path,icon,sort) VALUES
 (103,100,'机台管理','/system/machines','deployment-unit',3),
 (1007,1000,'机台产量','/report/machine-output','bar-chart',7);
INSERT INTO role_menu(role_id,menu_id) SELECT role_id,103 FROM role_menu WHERE menu_id=101 ON CONFLICT DO NOTHING;
INSERT INTO role_menu(role_id,menu_id) SELECT role_id,1007 FROM role_menu WHERE menu_id=401 ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (103,1007);
DELETE FROM menu WHERE id IN (103,1007);
DROP INDEX scan_record_machine_scanned_idx;
ALTER TABLE scan_record DROP COLUMN machine_id;
DROP TABLE machine;

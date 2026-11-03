-- +goose Up
INSERT INTO menu (id,parent_id,title,path,icon,sort) VALUES
 (305,300,'外协单查看','/scheduling/outsource-orders','file-search',5),
 (802,800,'发货计划','/delivery/plans','calendar',2);
INSERT INTO role_menu(role_id,menu_id)
SELECT role_id,305 FROM role_menu WHERE menu_id=301 ON CONFLICT DO NOTHING;
INSERT INTO role_menu(role_id,menu_id)
SELECT role_id,802 FROM role_menu WHERE menu_id=801 ON CONFLICT DO NOTHING;

CREATE TABLE delivery_plan (
 id BIGSERIAL PRIMARY KEY,
 plan_no TEXT NOT NULL UNIQUE,
 factory_id BIGINT NOT NULL REFERENCES factory(id),
 logistics_company TEXT NOT NULL DEFAULT '', contact TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL DEFAULT '',
 shipping_address TEXT NOT NULL DEFAULT '', receiving_address TEXT NOT NULL DEFAULT '',
 car_model TEXT NOT NULL DEFAULT '', plate_no TEXT NOT NULL DEFAULT '', driver TEXT NOT NULL DEFAULT '', driver_phone TEXT NOT NULL DEFAULT '',
 cost NUMERIC(12,2) NOT NULL DEFAULT 0, extra_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
 estimated_departure_date DATE NOT NULL, confirmed_departure_date DATE,
 confirmed_by BIGINT REFERENCES app_user(id), confirmed_at TIMESTAMPTZ,
 remark TEXT NOT NULL DEFAULT '', created_by BIGINT NOT NULL REFERENCES app_user(id), created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE delivery_note ADD COLUMN plan_id BIGINT REFERENCES delivery_plan(id);
CREATE INDEX delivery_note_plan_id_idx ON delivery_note(plan_id);

-- +goose Down
DROP INDEX delivery_note_plan_id_idx;
ALTER TABLE delivery_note DROP COLUMN plan_id;
DROP TABLE delivery_plan;
DELETE FROM role_menu WHERE menu_id IN (305,802);
DELETE FROM menu WHERE id IN (305,802);

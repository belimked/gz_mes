-- +goose Up
CREATE TABLE job (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    seq INT NOT NULL UNIQUE,
    stage TEXT NOT NULL CHECK (stage IN ('sheet_metal', 'spray', 'finished'))
);
INSERT INTO job (code, name, seq, stage) VALUES
('10','开料',1,'sheet_metal'), ('15','折弯',2,'sheet_metal'),
('20','钣金下线',3,'sheet_metal'), ('25','钣金质检',4,'sheet_metal'),
('30','喷涂接收',5,'spray'), ('35','上挂喷涂',6,'spray'),
('40','下挂冷却',7,'spray'), ('45','喷涂质检',8,'spray'),
('50','成品接收',9,'finished'), ('55','成品装架',10,'finished'),
('60','发货对账',11,'finished');

CREATE TABLE batch (
    id BIGSERIAL PRIMARY KEY,
    factory_id BIGINT NOT NULL REFERENCES factory (id),
    contract_id BIGINT NOT NULL REFERENCES contract (id),
    lot_no TEXT NOT NULL,
    client_order_no TEXT NOT NULL DEFAULT '',
    thickness NUMERIC(8,2) NOT NULL DEFAULT 0,
    color TEXT NOT NULL DEFAULT '', color_no TEXT NOT NULL DEFAULT '',
    texture TEXT NOT NULL DEFAULT '', spraying_type TEXT NOT NULL DEFAULT '',
    delivery_address TEXT NOT NULL DEFAULT '', special_request TEXT NOT NULL DEFAULT '',
    workshop TEXT NOT NULL DEFAULT '', remark TEXT NOT NULL DEFAULT '',
    order_qty INT NOT NULL DEFAULT 0, order_area NUMERIC(14,4) NOT NULL DEFAULT 0,
    required_date DATE,
    urgency TEXT NOT NULL DEFAULT 'normal' CHECK (urgency IN ('normal','urgent')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','scheduled','in_production','done')),
    item_count INT NOT NULL DEFAULT 0, item_qty INT NOT NULL DEFAULT 0,
    item_area NUMERIC(14,4) NOT NULL DEFAULT 0,
    plan_tech_date DATE, plan_drawing_date DATE, plan_cut_date DATE,
    plan_sheet_start DATE, plan_sheet_delivery DATE,
    plan_spray_start DATE, plan_spray_delivery DATE,
    outsource_factory_id BIGINT REFERENCES factory (id),
    created_by BIGINT NOT NULL REFERENCES app_user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (factory_id, lot_no)
);
CREATE INDEX batch_factory_created_idx ON batch (factory_id, created_at DESC);
CREATE INDEX batch_contract_idx ON batch (contract_id);
CREATE INDEX batch_status_idx ON batch (status);

CREATE TABLE batch_item (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES batch (id) ON DELETE CASCADE,
    seq INT NOT NULL,
    piece_no TEXT NOT NULL,
    drawing_no TEXT NOT NULL DEFAULT '', pattern TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT '', remark TEXT NOT NULL DEFAULT '',
    thickness NUMERIC(8,2) NOT NULL DEFAULT 0,
    length NUMERIC(10,2) NOT NULL DEFAULT 0, width NUMERIC(10,2) NOT NULL DEFAULT 0,
    piece_area NUMERIC(12,4) NOT NULL DEFAULT 0, spraying_area NUMERIC(12,4) NOT NULL DEFAULT 0,
    weld_line NUMERIC(10,2) NOT NULL DEFAULT 0,
    qty INT NOT NULL, package_qty INT NOT NULL DEFAULT 0, bending_times INT NOT NULL DEFAULT 0,
    UNIQUE (batch_id, piece_no)
);
CREATE INDEX batch_item_batch_seq_idx ON batch_item (batch_id, seq);

-- +goose Down
DROP TABLE batch_item;
DROP TABLE batch;
DROP TABLE job;

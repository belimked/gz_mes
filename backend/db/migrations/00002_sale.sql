-- +goose Up
CREATE TABLE customer (
    id BIGSERIAL PRIMARY KEY,
    factory_id BIGINT NOT NULL REFERENCES factory (id),
    name TEXT NOT NULL,
    contacts TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    bank TEXT NOT NULL DEFAULT '',
    account TEXT NOT NULL DEFAULT '',
    eas_code TEXT NOT NULL DEFAULT '',
    eas_name TEXT NOT NULL DEFAULT '',
    is_internal BOOLEAN NOT NULL DEFAULT false,
    confirmed BOOLEAN NOT NULL DEFAULT false,
    created_by BIGINT NOT NULL REFERENCES app_user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (factory_id, name)
);
CREATE INDEX customer_factory_created_idx ON customer (factory_id, created_at DESC);

CREATE TABLE contract (
    id BIGSERIAL PRIMARY KEY,
    factory_id BIGINT NOT NULL REFERENCES factory (id),
    contract_no TEXT NOT NULL,
    customer_contract_no TEXT NOT NULL DEFAULT '',
    inner_contract_no TEXT NOT NULL DEFAULT '',
    customer_id BIGINT NOT NULL REFERENCES customer (id),
    project_id BIGINT NOT NULL REFERENCES project (id),
    salesman TEXT NOT NULL DEFAULT '',
    pricing_manner TEXT NOT NULL DEFAULT '',
    payment TEXT NOT NULL DEFAULT '',
    payment_point TEXT NOT NULL DEFAULT '',
    texture TEXT NOT NULL DEFAULT '',
    paint_type TEXT NOT NULL DEFAULT '',
    packing TEXT NOT NULL DEFAULT '',
    receiver TEXT NOT NULL DEFAULT '',
    receiver_phone TEXT NOT NULL DEFAULT '',
    receiver_address TEXT NOT NULL DEFAULT '',
    remark TEXT NOT NULL DEFAULT '',
    order_qty INT NOT NULL DEFAULT 0,
    order_area NUMERIC(14,4) NOT NULL DEFAULT 0,
    customer_valuation_audited BOOLEAN NOT NULL DEFAULT false,
    outsource_valuation_audited BOOLEAN NOT NULL DEFAULT false,
    created_by BIGINT NOT NULL REFERENCES app_user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (factory_id, contract_no)
);
CREATE INDEX contract_factory_created_idx ON contract (factory_id, created_at DESC);
CREATE INDEX contract_customer_idx ON contract (customer_id);
CREATE INDEX contract_project_idx ON contract (project_id);

-- +goose Down
DROP TABLE contract;
DROP TABLE customer;

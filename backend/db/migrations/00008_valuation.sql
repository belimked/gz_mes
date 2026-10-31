-- +goose Up
CREATE TABLE batch_valuation (
 batch_id BIGINT PRIMARY KEY REFERENCES batch(id),
 basis TEXT NOT NULL CHECK (basis IN ('order_area','item_area','delivered_area')),
 basis_area NUMERIC(12,4) NOT NULL,
 unit_price NUMERIC(12,4) NOT NULL,
 ingot_price NUMERIC(12,2) NOT NULL DEFAULT 0,
 amount NUMERIC(14,2) NOT NULL,
 remark TEXT NOT NULL DEFAULT '',
 audited BOOL NOT NULL DEFAULT false,
 audited_by BIGINT REFERENCES app_user(id),
 audited_at TIMESTAMPTZ,
 updated_by BIGINT NOT NULL REFERENCES app_user(id),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE batch_valuation;

-- +goose Up
CREATE TABLE handover (
 id BIGSERIAL PRIMARY KEY, handover_no TEXT NOT NULL UNIQUE,
 factory_id BIGINT NOT NULL REFERENCES factory(id), batch_id BIGINT NOT NULL REFERENCES batch(id), job_id BIGINT NOT NULL REFERENCES job(id),
 qty INT NOT NULL, area NUMERIC(12,4) NOT NULL, remark TEXT NOT NULL DEFAULT '',
 packed_by BIGINT NOT NULL REFERENCES app_user(id), packed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 received BOOL NOT NULL DEFAULT false, received_by BIGINT REFERENCES app_user(id), received_at TIMESTAMPTZ
);
CREATE TABLE handover_label (
 handover_id BIGINT NOT NULL REFERENCES handover(id) ON DELETE CASCADE,
 label_id BIGINT NOT NULL REFERENCES label(id), job_id BIGINT NOT NULL REFERENCES job(id),
 PRIMARY KEY(handover_id,label_id), UNIQUE(label_id,job_id)
);
CREATE INDEX handover_factory_idx ON handover(factory_id,packed_at DESC);
CREATE TABLE delivery_note (
 id BIGSERIAL PRIMARY KEY, note_no TEXT NOT NULL UNIQUE,
 factory_id BIGINT NOT NULL REFERENCES factory(id), batch_id BIGINT NOT NULL REFERENCES batch(id), customer_id BIGINT NOT NULL REFERENCES customer(id),
 delivery_batch TEXT NOT NULL DEFAULT '', qty INT NOT NULL, area NUMERIC(12,4) NOT NULL,
 logistics_company TEXT NOT NULL DEFAULT '', plate_no TEXT NOT NULL DEFAULT '', driver TEXT NOT NULL DEFAULT '', driver_phone TEXT NOT NULL DEFAULT '',
 delivered_at DATE NOT NULL, remark TEXT NOT NULL DEFAULT '', account_id BIGINT,
 created_by BIGINT NOT NULL REFERENCES app_user(id), created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE delivery_note_label (
 note_id BIGINT NOT NULL REFERENCES delivery_note(id) ON DELETE CASCADE,
 label_id BIGINT NOT NULL UNIQUE REFERENCES label(id), PRIMARY KEY(note_id,label_id)
);
CREATE INDEX delivery_note_factory_idx ON delivery_note(factory_id,created_at DESC);
CREATE TABLE deliver_account (
 id BIGSERIAL PRIMARY KEY, account_no TEXT NOT NULL UNIQUE,
 factory_id BIGINT NOT NULL REFERENCES factory(id), customer_id BIGINT NOT NULL REFERENCES customer(id),
 note_count INT NOT NULL, total_qty INT NOT NULL, total_area NUMERIC(12,4) NOT NULL,
 unit_price NUMERIC(12,4) NOT NULL, extra_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
 extra_remark TEXT NOT NULL DEFAULT '', amount NUMERIC(14,2) NOT NULL,
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','confirmed')),
 remark TEXT NOT NULL DEFAULT '', created_by BIGINT NOT NULL REFERENCES app_user(id), created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 confirmed_by BIGINT REFERENCES app_user(id), confirmed_at TIMESTAMPTZ
);
ALTER TABLE delivery_note ADD CONSTRAINT delivery_note_account_fk FOREIGN KEY(account_id) REFERENCES deliver_account(id);
CREATE INDEX deliver_account_factory_idx ON deliver_account(factory_id,created_at DESC);
CREATE INDEX delivery_note_account_idx ON delivery_note(account_id);
CREATE TABLE doc_seq (kind TEXT NOT NULL, day DATE NOT NULL, n INT NOT NULL, PRIMARY KEY(kind,day));

-- +goose Down
DROP TABLE doc_seq;
ALTER TABLE delivery_note DROP CONSTRAINT delivery_note_account_fk;
DROP TABLE deliver_account;
DROP TABLE delivery_note_label;
DROP TABLE delivery_note;
DROP TABLE handover_label;
DROP TABLE handover;

-- +goose Up
CREATE TABLE label (
    id BIGSERIAL PRIMARY KEY,
    bar_code TEXT NOT NULL UNIQUE,
    factory_id BIGINT NOT NULL REFERENCES factory (id),
    batch_id BIGINT NOT NULL REFERENCES batch (id),
    batch_item_id BIGINT NOT NULL REFERENCES batch_item (id) ON DELETE CASCADE,
    piece_index INT NOT NULL,
    cancelled BOOL NOT NULL DEFAULT false,
    created_by BIGINT NOT NULL REFERENCES app_user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (batch_item_id, piece_index)
);
CREATE INDEX label_batch_idx ON label (batch_id);
CREATE INDEX label_factory_created_idx ON label (factory_id, created_at DESC);

CREATE TABLE scan_record (
    id BIGSERIAL PRIMARY KEY,
    label_id BIGINT NOT NULL REFERENCES label (id) ON DELETE CASCADE,
    batch_id BIGINT NOT NULL REFERENCES batch (id),
    job_id BIGINT NOT NULL REFERENCES job (id),
    scanned_by BIGINT NOT NULL REFERENCES app_user (id),
    scanned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    device TEXT NOT NULL DEFAULT '',
    UNIQUE (label_id, job_id)
);
CREATE INDEX scan_record_batch_job_idx ON scan_record (batch_id, job_id);
CREATE INDEX scan_record_scanned_at_idx ON scan_record (scanned_at DESC);

-- +goose Down
DROP TABLE scan_record;
DROP TABLE label;

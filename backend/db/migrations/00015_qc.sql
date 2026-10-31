-- +goose Up
ALTER TABLE job ADD COLUMN is_qc BOOLEAN NOT NULL DEFAULT false;
UPDATE job SET is_qc = true WHERE code IN ('25', '45');

-- One row per inspection. A pass also inserts the flow scan_record; a fail does not, so the piece
-- stays blocked at the QC job until a later inspection passes (复检).
CREATE TABLE qc_record (
    id BIGSERIAL PRIMARY KEY,
    label_id BIGINT NOT NULL REFERENCES label (id) ON DELETE CASCADE,
    batch_id BIGINT NOT NULL REFERENCES batch (id),
    job_id BIGINT NOT NULL REFERENCES job (id),
    result TEXT NOT NULL CHECK (result IN ('pass', 'fail')),
    defect_job_id BIGINT REFERENCES job (id),
    defect_category TEXT NOT NULL DEFAULT '',
    defect_reason TEXT NOT NULL DEFAULT '',
    inspected_by BIGINT NOT NULL REFERENCES app_user (id),
    inspected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    device TEXT NOT NULL DEFAULT ''
);
CREATE INDEX qc_record_label_job_idx ON qc_record (label_id, job_id);
CREATE INDEX qc_record_batch_job_idx ON qc_record (batch_id, job_id);
CREATE INDEX qc_record_inspected_at_idx ON qc_record (inspected_at DESC);

INSERT INTO menu (id, parent_id, title, path, icon, sort) VALUES
    (700, NULL, '质量管理', '/quality',          'audit',     70),
    (701, 700,  '质检扫描记录', '/quality/qc-scans', 'scan',   1),
    (702, 700,  '质量报表',   '/quality/report',   'bar-chart', 2),
    (1006, 1000, '不良品日报', '/report/defect-daily', 'file-text', 6);
INSERT INTO role_menu (role_id, menu_id)
SELECT rm.role_id, m.id FROM role_menu rm CROSS JOIN (VALUES (701), (702), (1006)) AS m (id)
WHERE rm.menu_id = 401
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (701, 702, 1006);
DELETE FROM menu WHERE id IN (701, 702, 1006, 700);
DROP TABLE qc_record;
ALTER TABLE job DROP COLUMN is_qc;

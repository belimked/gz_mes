-- +goose Up
CREATE TABLE plan (
    id          BIGSERIAL PRIMARY KEY,
    kind        TEXT        NOT NULL CHECK (kind IN ('sheet_week', 'sheet_daily', 'spray_daily')),
    factory_id  BIGINT      NOT NULL REFERENCES factory (id),
    batch_id    BIGINT      NOT NULL REFERENCES batch (id),
    start_date  DATE        NOT NULL,
    end_date    DATE        NOT NULL,
    workshop    TEXT        NOT NULL DEFAULT '',
    job_id      BIGINT REFERENCES job (id),
    paint_line  TEXT        NOT NULL DEFAULT '',
    shift       TEXT        NOT NULL DEFAULT '',
    spray_type  TEXT        NOT NULL DEFAULT '',
    urgency     TEXT        NOT NULL DEFAULT 'normal' CHECK (urgency IN ('normal', 'urgent')),
    audit_state TEXT        NOT NULL DEFAULT 'draft' CHECK (audit_state IN ('draft', 'audited')),
    audited_by  BIGINT REFERENCES app_user (id),
    audited_at  TIMESTAMPTZ,
    stop_state  TEXT        NOT NULL DEFAULT 'normal' CHECK (stop_state IN ('normal', 'paused', 'stopped')),
    stop_reason TEXT        NOT NULL DEFAULT '',
    remark      TEXT        NOT NULL DEFAULT '',
    created_by  BIGINT      NOT NULL REFERENCES app_user (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date),
    CHECK (kind <> 'sheet_daily' OR job_id IS NOT NULL)
);
CREATE INDEX plan_kind_dates_idx ON plan (kind, start_date, end_date);
CREATE INDEX plan_batch_idx ON plan (batch_id);

INSERT INTO menu (id, parent_id, title, path, icon, sort) VALUES
    (302, 300, '钣金周计划', '/scheduling/sheet-week',  'calendar',  2),
    (303, 300, '钣金日计划', '/scheduling/sheet-daily', 'schedule',  3),
    (304, 300, '喷涂日计划', '/scheduling/spray-daily', 'bg-colors', 4);
-- Roles that can already open 订单批次排程 also get the new plan screens.
INSERT INTO role_menu (role_id, menu_id)
SELECT rm.role_id, m.id FROM role_menu rm CROSS JOIN (VALUES (302), (303), (304)) AS m (id)
WHERE rm.menu_id = 301
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_menu WHERE menu_id IN (302, 303, 304);
DELETE FROM menu WHERE id IN (302, 303, 304);
DROP TABLE plan;

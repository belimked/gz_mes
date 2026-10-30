-- +goose Up
CREATE TABLE factory (
    id         BIGSERIAL PRIMARY KEY,
    code       TEXT        NOT NULL UNIQUE,
    name       TEXT        NOT NULL,
    kind       TEXT        NOT NULL CHECK (kind IN ('own', 'outsource')),
    is_stop    BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app_user (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    display_name  TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    is_admin      BOOLEAN     NOT NULL DEFAULT false,
    enabled       BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Factory scope of a non-admin user (outsource accounts see only their own factory).
CREATE TABLE user_factory (
    user_id    BIGINT NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    factory_id BIGINT NOT NULL REFERENCES factory (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, factory_id)
);

CREATE TABLE role (
    id   BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

CREATE TABLE menu (
    id        BIGINT PRIMARY KEY,
    parent_id BIGINT REFERENCES menu (id),
    title     TEXT NOT NULL,
    path      TEXT NOT NULL UNIQUE,
    icon      TEXT NOT NULL DEFAULT '',
    sort      INT  NOT NULL DEFAULT 0
);

CREATE TABLE role_menu (
    role_id BIGINT NOT NULL REFERENCES role (id) ON DELETE CASCADE,
    menu_id BIGINT NOT NULL REFERENCES menu (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, menu_id)
);

CREATE TABLE user_role (
    user_id BIGINT NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES role (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE project (
    id         BIGSERIAL PRIMARY KEY,
    factory_id BIGINT      NOT NULL REFERENCES factory (id),
    code       TEXT        NOT NULL,
    full_name  TEXT        NOT NULL,
    short_name TEXT        NOT NULL DEFAULT '',
    address    TEXT        NOT NULL DEFAULT '',
    kind       TEXT        NOT NULL DEFAULT 'offline' CHECK (kind IN ('online', 'offline')),
    confirmed  BOOLEAN     NOT NULL DEFAULT false,
    created_by BIGINT      NOT NULL REFERENCES app_user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (factory_id, code)
);
CREATE INDEX project_factory_created_idx ON project (factory_id, created_at DESC);

INSERT INTO menu (id, parent_id, title, path, icon, sort) VALUES
    (100, NULL, '系统管理',     '/system',                   'setting',      10),
    (101, 100,  '厂区管理',     '/system/factories',         '',             1),
    (102, 100,  '用户管理',     '/system/users',             '',             2),
    (200, NULL, '销售管理',     '/sale',                     'shop',         20),
    (201, 200,  '项目管理',     '/sale/projects',            '',             1),
    (202, 200,  '客户管理',     '/sale/customers',           '',             2),
    (203, 200,  '合同查询',     '/sale/contracts',           '',             3),
    (300, NULL, '排程管理',     '/scheduling',               'schedule',     30),
    (301, 300,  '订单批次排程', '/scheduling/batches',       '',             1),
    (400, NULL, '生产管理',     '/production',               'build',        40),
    (401, 400,  '面积表查询',   '/production/area-tables',   '',             1),
    (402, 400,  '交接单管理',   '/production/handovers',     '',             2),
    (800, NULL, '发货管理',     '/delivery',                 'car',          80),
    (801, 800,  '发货查询',     '/delivery/notes',           '',             1),
    (900, NULL, '财务管理',     '/finance',                  'account-book', 90),
    (901, 900,  '工程批次对账', '/finance/batch-valuation',  '',             1),
    (902, 900,  '发货对账查询', '/finance/deliver-accounts', '',             2);

-- +goose Down
DROP TABLE project;
DROP TABLE user_role;
DROP TABLE role_menu;
DROP TABLE menu;
DROP TABLE role;
DROP TABLE user_factory;
DROP TABLE app_user;
DROP TABLE factory;

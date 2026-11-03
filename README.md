# GZ MES

铝单板工厂生产执行系统（MES）：销售合同 → 生产批次 → 排程 → 面积表/标签 → 工序扫描 → 发货 → 对账。

- **在线演示**：<https://gzmes.saulxj.ggff.net>（备用 <https://gz-mes-demo.pages.dev>）（纯前端演示，数据为虚构，仅保存在浏览器；账号 `admin / admin123`、`waixie / demo123`）
- 后端：Go（chi + oapi-codegen + pgx + goose），PostgreSQL 16
- 前端：Vue 3 + Vite + TypeScript + Ant Design Vue
- 接口契约：[`api/openapi.yaml`](api/openapi.yaml) 是唯一来源，Go 服务端接口与前端 TS 类型均由它生成

## 目录

```
api/openapi.yaml        接口定义（改接口先改这里，再 make gen）
backend/                Go API（cmd/server、internal/*、db/migrations）
frontend/               Vue 前端（src/theme.ts 统一管理主题与布局）
  src/api/demo/         演示模式：浏览器内模拟后端（行为与 Go 后端一致）
deploy/                 docker compose（db + api + web）
.github/workflows/      CI、演示部署、发版
```

## 本地开发

需要 Go 1.27+、Node 24（22.22+ 也可）、Docker。

```bash
make dev-db                       # PostgreSQL → 127.0.0.1:15432
make api                          # 后端 → 127.0.0.1:18000（首次启动创建 admin / admin123）
cd frontend && npm ci && npm run dev   # 前端 → http://127.0.0.1:15173
```

只看前端、不跑后端：`cd frontend && npx vite --mode demo`。

## 测试

```bash
make test        # 后端集成测试（真实 PostgreSQL）+ 前端类型检查/lint/单测
make e2e         # Playwright：演示构建上的端到端测试
```

## 部署

```bash
cd deploy && cp .env.example .env   # 修改所有密码/密钥
docker compose up -d                # 默认拉取 ghcr.io/belimked/gz-mes-{api,web}:latest
```

API 使用 `FILE_STORAGE_DIR`（默认 `/data/files`）保存图纸，`MAX_UPLOAD_MB`（默认 50）限制上传大小；启动时目录必须可写。Compose 将命名卷 `file-data` 挂载到 `/data/files`，备份数据库时请同时备份该卷。开发时可用 `FILE_STORAGE_DIR=$PWD/.tmp/files make api`。

打 `v*` 标签会构建并推送镜像到 GHCR，并在 GitHub Release 附上锁定版本的 `docker-compose.yml`。

## CI / CD

| 工作流 | 触发 | 内容 |
|---|---|---|
| CI | push main / PR | 生成代码一致性、Go lint + 集成测试、前端 typecheck/lint/单测/构建、演示版 E2E、docker compose 全栈 E2E |
| Demo | push main / PR | 构建演示版并部署到 Cloudflare Pages（PR 有独立预览地址） |
| Release | tag `v*` | 推送镜像到 GHCR，创建 GitHub Release |

# 开发指南

## 1. 前置要求

- Docker Desktop（Windows/macOS）或 Docker Engine（Linux）。
- Docker Compose v2。
- Git。
- 不需要在宿主机安装 PostgreSQL、Redis、pgvector 或 Go 开发依赖。

> Wails 桌面窗口依赖操作系统 WebView，最终调试窗口仍需在桌面系统运行；但后端、数据库和依赖服务全部在 Docker 中。

## 2. 初始化本地配置

仓库只提交示例配置，不提交真实配置和 `.env`：

```bash
cp backend/config/config.example.yml backend/config/config.yml
cp .env.example .env
```

`config.yml` 和 `.env` 已被 `.gitignore` 忽略，不要提交。

如果忘记复制，`docker compose` 启动后端时也会自动从 `config.example.yml` 生成 `config.yml`。

## 3. 一键启动

```bash
docker compose up -d --build
```

查看状态：

```bash
docker compose ps
```

查看后端日志：

```bash
docker compose logs -f backend
```

健康检查：

```text
http://127.0.0.1:18080/healthz
```

停止服务：

```bash
docker compose down
```

删除数据库和 Redis 数据（谨慎）：

```bash
docker compose down -v
```

## 4. 服务与端口

| 服务 | 地址 | 说明 |
|---|---|---|
| backend | http://127.0.0.1:18080 | Gin API |
| postgres | 127.0.0.1:5432 | PostgreSQL 16 + pgvector |
| redis | 127.0.0.1:6379 | Redis 7 |

默认数据库：

```text
database: agent_coder_team
user: act
password: act_dev_password
```

## 5. 配置文件

后端配置：

```text
backend/config/config.yml
```

容器内通过环境变量加载：

```text
CONFIG_PATH=/app/config/config.yml
```

配置分为：

- `app`：名称、版本、环境、Gin 模式。
- `server`：监听地址、端口、超时。
- `database`：PostgreSQL 连接。
- `redis`：Redis 连接。
- `queue`：Asynq 队列和并发数。
- `vector`：pgvector 配置。
- `sandbox`：Docker 沙箱限制。
- `auth`：会话有效期。
- `security`：HTTPS 与 Origin。
- `scheduler`：开工窗口。
- `i18n`：默认语言和支持语言。

规则：

- 不要把生产密钥提交到仓库。
- 本地覆盖使用 `config.local.yml`，该文件已被 `.gitignore` 忽略。
- `.env` 只用于 Docker Compose 的端口和数据库账号变量，示例见 `.env.example`。
- 修改端口或数据库密码时，同时修改 `docker-compose.yml` 和配置文件，或使用环境变量覆盖 compose 默认值。

## 6. 常用命令

进入后端容器：

```bash
docker compose exec backend sh
```

执行 Go 命令：

```bash
docker compose exec backend go test ./...
docker compose exec backend go vet ./...
docker compose exec backend go build ./...
```

格式化：

```bash
docker compose exec backend gofmt -w .
```

进入 PostgreSQL：

```bash
docker compose exec postgres psql -U act -d agent_coder_team
```

检查 pgvector：

```sql
SELECT extname FROM pg_extension WHERE extname = 'vector';
```

进入 Redis：

```bash
docker compose exec redis redis-cli
```

## 7. 目录结构

```text
agent-coder-team/
├─ backend/
│  ├─ cmd/app/              # 服务入口
│  ├─ internal/config/      # 配置加载
│  ├─ internal/logger/      # 日志
│  ├─ config/config.example.yml  # 提交的示例配置
│  ├─ config/config.yml          # 本地配置，不提交
│  ├─ Dockerfile.dev        # 开发镜像
│  └─ go.mod
├─ deploy/
│  └─ postgres/init/        # 数据库初始化 SQL
├─ docs/
│  ├─ README.md             # 产品与范围
│  ├─ tech-stack.md         # 技术选型
│  ├─ architecture.md       # 架构与关键设计
│  └─ development.md        # 本文件
├─ .env.example            # 提交的示例环境变量
├─ .env                    # 本地环境变量，不提交
├─ docker-compose.yml
├─ AGENTS.md
└─ README.md
```

后续新增前端时，再增加 `apps/desktop` 和 `apps/mobile`，不要提前创建空目录。

## 8. 开发规则

1. 先读 `AGENTS.md` 和本文件，再改代码。
2. 生产代码不做 GORM `AutoMigrate`，数据库变更使用迁移文件。
3. 领域层不依赖 Gin、GORM、Docker、模型 SDK。
4. 外部服务通过接口适配，便于测试和替换。
5. 所有 Agent 代码执行必须走 Docker 沙箱。
6. 合并、部署、删除等高风险操作必须人工审批。
7. 不提交密钥、数据库文件、构建产物和日志。
8. 新增依赖前先确认是否必要，保持技术栈精简。
9. 提交前至少运行 `go test ./...` 和 `go vet ./...`。
10. 文档只维护 `README.md`、`docs/README.md`、`docs/architecture.md`、`docs/development.md` 和 `AGENTS.md`，不要新增零散文档。

## 9. 故障排查

### 端口被占用

修改 `docker-compose.yml` 中的端口映射，或设置环境变量：

```bash
BACKEND_PORT=18081 docker compose up -d
```

### 数据库初始化失败

查看日志：

```bash
docker compose logs postgres
```

确认 `deploy/postgres/init/001_extensions.sql` 被执行。它只在数据卷首次创建时运行。

### 后端无法连接数据库

确认容器内使用的主机名是：

```text
postgres
redis
```

而不是 `localhost`。

### 修改配置后未生效

```bash
docker compose restart backend
```

### 彻底重置开发环境

```bash
docker compose down -v
docker compose up -d --build
```

## 10. Git 工作流

建议提交粒度：

- `feat:` 新功能
- `fix:` 修复
- `refactor:` 重构
- `docs:` 文档
- `chore:` 构建、依赖、配置
- `test:` 测试

提交前检查：

```bash
git status
git diff
docker compose config --quiet
docker compose exec backend go test ./...
```

## 11. 当前阶段

已完成：

- Docker 开发环境、PostgreSQL/pgvector、Redis。
- 配置加载、启动校验、结构化日志。
- PostgreSQL 和 Redis 连接、就绪检查。
- 请求 ID 和访问日志中间件。
- 优雅关闭。
- 基础单元测试。

可用端点：

```text
GET /healthz
GET /readyz
GET /api/v1/meta
```

下一步：

1. 数据库迁移和基础表。
2. 本地账号、会话和认证中间件。
3. 项目创建、导入和项目列表。
4. 项目群聊和消息存储。
5. M0 技术验证：Wails IPC、Docker 沙箱、pgvector、Tailscale/TLS。

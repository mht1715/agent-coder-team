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
- `workspace`：允许的项目工作目录根路径和扫描文件上限。
- `vector`：pgvector 配置。
- `sandbox`：Docker 沙箱限制、允许镜像、并发数、输出上限和开发环境工作区锚点。
- `auth`：会话有效期。
- `security`：HTTPS 与 Origin。
- `scheduler`：开工窗口。
- `i18n`：默认语言和支持语言。

规则：

- 不要把生产密钥提交到仓库。
- 本地覆盖使用 `config.local.yml`，该文件已被 `.gitignore` 忽略。
- `.env` 只用于 Docker Compose 的端口和数据库账号变量，示例见 `.env.example`。
- 修改端口或数据库密码时，同时修改 `docker-compose.yml` 和配置文件，或使用环境变量覆盖 compose 默认值。

## 6. Docker 沙箱执行

开发环境由 `docker compose` 提供三个与沙箱相关的组件：

- `workspace-anchor`：仅挂载 `./workspaces` 的轻量容器，作为项目工作区的共享锚点。
- 后端容器：安装 Docker CLI，并通过本机 Docker Socket 创建一次性兄弟沙箱。Socket 不会进入沙箱。
- 沙箱容器：每次任务执行单独创建，使用 `--volumes-from workspace-anchor` 只共享工作区，默认 `network=none`、`cap-drop=ALL`、`no-new-privileges`，并限制 CPU、内存、PID 和超时。

任务执行命令默认从项目画像的测试命令、构建命令推导，镜像按主语言从 `sandbox.images` 白名单选择。执行结果会写入 `agent_runs` 的日志、退出码和变更文件；成功后任务进入 `review`，失败后进入 `blocked`。

本地配置可覆盖：

```yaml
sandbox:
  workspace_anchor: act-workspace-anchor
  container_workspace_root: /workspaces
  max_concurrent: 1
  timeout_seconds: 1800
  default_image: alpine:3.20
  images:
    - alpine:3.20
    - golang:1.27-alpine
    - node:22-alpine
```

安全边界：

- 不把 Docker Socket、用户主目录或系统目录挂载进沙箱。
- 沙箱根文件系统是临时的，项目工作区可写。
- 沙箱执行的是用户审批过的命令，不是模型生成的任意宿主机命令。
- 生产环境应把 Docker 访问限制到专用 runner，而不是直接暴露给通用后端进程。
- 当前服务重启不会自动恢复运行中的沙箱任务，需要将任务重新排队后再执行。

## 7. 常用命令

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

## 8. 目录结构

```text
agent-coder-team/
├─ apps/
│  └─ desktop/             # Wails 3 + Vue 3 + Element Plus
│     └─ frontend/         # 桌面 UI
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

## 9. 开发规则

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

## 10. 故障排查

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

## 11. Git 工作流

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

## 12. 当前阶段

已完成：

- Docker 开发环境、PostgreSQL/pgvector、Redis。
- 配置加载、启动校验、结构化日志。
- PostgreSQL 和 Redis 连接、就绪检查。
- 请求 ID 和访问日志中间件。
- 优雅关闭。
- 数据库迁移和基础表。
- 本地账号、登录、刷新令牌、退出。
- 项目创建、列表、详情和更新。
- 项目群聊、消息分页、消息发送和 SSE 实时事件。
- 项目工作目录校验、技术栈识别和项目画像。
- Wails 3 桌面端：登录、项目列表、主 Agent 群聊、需求、任务看板、审批与执行记录。
- 主 Agent 流式回复、PRD 草稿生成和需求批准。
- 任务 DAG 生成、依赖解锁、任务状态机和计划审批。
- 任务执行审批、Docker 沙箱任务执行、实时日志、退出码和变更文件记录。
- 基础单元测试。

可用端点：

```text
GET /healthz
GET /readyz
GET /api/v1/meta
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET /api/v1/me
POST /api/v1/projects
GET /api/v1/projects
GET /api/v1/projects/:id
PATCH /api/v1/projects/:id
GET /api/v1/projects/:id/profile
POST /api/v1/projects/:id/profile/scan
GET /api/v1/projects/:id/conversations
GET /api/v1/conversations/:id/messages
POST /api/v1/conversations/:id/messages
GET /api/v1/conversations/:id/stream
GET /api/v1/projects/:id/requirement
POST /api/v1/projects/:id/requirement/generate
PATCH /api/v1/projects/:id/requirement
POST /api/v1/projects/:id/requirement/approve
GET /api/v1/projects/:id/tasks
POST /api/v1/projects/:id/tasks/generate
POST /api/v1/tasks/:id/transition
POST /api/v1/tasks/:id/request-execution
GET /api/v1/projects/:id/approvals
POST /api/v1/approvals/:id/decision
GET /api/v1/projects/:id/runs
POST /api/v1/runs/:id/finish
```

下一步：

1. 模型网关、可替换主 Agent 生成器和执行 Agent。
2. pgvector 索引、上下文预算和长期记忆。
3. Tailscale 移动端派活、设备配对和 TLS。

## 13. 开发账号

本地开发默认账号由 seed 命令创建：

```text
username: admin
password: admin12345
```

如果数据库被重置，执行：

```bash
docker compose exec backend go run ./cmd/seed
```

可以通过环境变量覆盖：

```bash
docker compose exec -e SEED_USERNAME=admin -e SEED_PASSWORD=admin12345 backend go run ./cmd/seed
```

该账号只用于本地开发，不会进入生产环境。

## 14. 桌面端开发

安装 Wails 3 CLI：

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
```

确保 `$(go env GOPATH)/bin` 在 PATH 中。

启动后端：

```bash
docker compose up -d backend
```

启动桌面端开发模式：

```bash
cd apps/desktop
wails3 dev
```

桌面端 WebView 通过 Wails IPC 调用 Go 桥接层。桥接层默认访问：

```text
http://127.0.0.1:18080/api/v1
```

可通过环境变量 `ACT_CORE_API_URL` 覆盖。登录令牌只保存在 Wails Go 进程内存中，不会下发到 WebView；Go 层负责鉴权、刷新和实时事件转发。

浏览器调试模式仍可通过 `.env.development` 和 Vite `/api` 代理访问后端。

构建桌面端：

```bash
cd apps/desktop
wails3 build
```

构建产物：

```text
apps/desktop/bin/
```

当前桌面端使用 Wails IPC 安全适配器。WebView 不直接访问 HTTP，项目消息实时事件由 Go 桥接层通过 Wails Events 推送。

桌面端当前页面：

- 登录 / 首次创建本地账号。
- 项目列表和创建项目。
- 主 Agent 群聊、流式回复和 Wails Events 实时消息。
- 需求与 PRD 生成、编辑和批准。
- 任务 DAG、看板、依赖解锁和审批。
- 审批中心、执行记录和任务验收。
- 项目画像查看与扫描。
- 深色 / 浅色 / 跟随系统主题。
- 中文 / 英文切换。
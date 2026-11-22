# Agent Coder Team

本地优先的 "Agent 软件公司"：用户提出需求并验收，主 Agent 负责澄清和规划，多角色 Agent 在 Docker 沙箱中完成开发。

## 状态

当前已完成可运行的 MVP 核心闭环：

- 本地账号、项目和工作区管理。
- Wails IPC 安全桌面端。
- 主 Agent 流式回复与需求澄清。
- PRD 生成、编辑和批准。
- 任务 DAG 生成、审批和状态机。
- 项目任务看板、审批中心和执行记录。
- Docker 沙箱任务执行、实时日志、退出码和代码变更文件记录。
- Docker 开发环境与 PostgreSQL/Redis 依赖。

## 快速开始

前置要求：Docker Desktop 或 Docker Engine + Docker Compose v2。

```bash
docker compose up -d --build
docker compose ps
```

健康检查：

```text
http://127.0.0.1:18080/healthz
```

常用命令：

```bash
docker compose logs -f backend
docker compose exec backend go test ./...
docker compose down
```

## 桌面端开发

安装 Wails 3 CLI：

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
```

启动后端后，在 `apps/desktop` 运行：

```bash
wails3 dev
```

当前桌面端包含登录、项目列表、主 Agent 群聊、流式回复、项目画像、需求与 PRD、任务看板、审批与执行记录、Docker 沙箱执行日志和主题切换。

本地开发账号：

```text
admin / admin12345
```

数据库重置后可执行：

```bash
docker compose exec backend go run ./cmd/seed
```

## 文档

- [产品范围与决策](./docs/README.md)
- [技术选型](./docs/tech-stack.md)
- [架构设计](./docs/architecture.md)
- [开发指南](./docs/development.md)
- [AI Agent 开发约定](./AGENTS.md)

## 技术基线

- 后端：Go + Gin + GORM + Viper
- 数据：PostgreSQL 16 + pgvector；Redis 7 + Asynq
- 桌面：Wails + Vue 3 + Element Plus
- 移动：优先 Wails + Vue 3 + Vant
- 沙箱：Docker
- 远程：Tailscale + HTTPS + 证书固定
- Agent 目标技术栈：Go、Vue、React、Java、Python、Node、.NET、uni-app

## 当前限制

主 Agent 仍使用本地确定性规划器，尚未接入模型网关。Docker 沙箱已经可以执行审批后的测试/构建命令并记录日志、退出码和工作区变更，但还没有由模型生成代码修改。

## 当前优先级

下一步接入可替换模型网关与执行 Agent，再做 pgvector 检索、上下文预算和长期记忆。详见 [产品范围与决策](./docs/README.md)。
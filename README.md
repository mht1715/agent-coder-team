# Agent Coder Team

本地优先的 "Agent 软件公司"：用户提出需求并验收，主 Agent 负责澄清和规划，多角色 Agent 在 Docker 沙箱中完成开发。

## 状态

产品与架构设计完成，仓库当前包含：

- Go 后端骨架。
- Docker 开发环境。
- 精简后的产品、架构和开发文档。

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

当前桌面端包含登录、项目列表、项目群聊、SSE 消息、项目画像和主题切换。

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

## 当前优先级

先完成 M0 技术验证，再进入业务开发。详见 [产品范围与决策](./docs/README.md)。
# AGENTS.md

本文件面向在本仓库工作的 AI 编码 Agent。开始修改前必须阅读本文件、`docs/tech-stack.md`、`docs/architecture.md` 和 `docs/development.md`。

## 1. 项目目标

Agent Coder Team 是一款本地优先的 "Agent 软件公司"：

- 用户提出需求，主 Agent 澄清并规划。
- 多角色 Agent 在隔离的 Git worktree 和 Docker 沙箱中开发。
- 用户审批需求、计划、合并和部署。
- 数据和密钥默认留在用户电脑。
- 移动端通过 Tailscale + HTTPS 远程查看、派活和审批。

## 2. 当前仓库状态

- 当前已有 Go 后端基础骨架、数据库迁移、本地认证、项目 CRUD、项目群聊、消息 SSE、项目画像、Wails 3 桌面端骨架、Docker 开发环境和健康检查。
- 不要提前创建空的 `apps/desktop`、`apps/mobile` 或前端代码。
- 不要假设未实现的功能已经存在。
- 需要新增目录时，先确认它服务于当前里程碑。

## 3. 技术基线

- 后端：Go、Gin、GORM、Viper、zerolog。
- 数据库：PostgreSQL 16 + pgvector。
- 队列/缓存：Redis 7 + Asynq。
- 桌面：Wails + Vue 3 + Element Plus。
- 移动：优先 Wails + Vue 3 + Vant，Capacitor 仅作兜底。
- 前端产品自身固定 Vue 3，不使用 React/uni-app。
- Agent 目标技术栈：Go、Vue、React、Java、Python、Node、.NET、uni-app。
- 不引入 Tauri、Kubernetes、消息中间件或重型框架。

## 4. 架构约束

1. 依赖方向：`transport -> application -> domain <- infrastructure`。
2. `domain` 不得 import Gin、GORM、Docker、模型 SDK 或数据库驱动。
3. Handler 只做参数解析、鉴权、调用用例和响应映射。
4. 外部服务必须通过接口适配。
5. 事务边界放在 application 层。
6. 不使用 GORM AutoMigrate 代替正式迁移。
7. 桌面 WebView 通过 Wails IPC 访问核心服务，不直接访问本地 HTTP。
8. 移动端 API 必须使用 HTTPS/TLS。
9. 所有 Agent 代码执行必须进 Docker 沙箱。
10. 不把整个仓库、全部聊天记录或密钥放入模型上下文。

## 5. 开发命令

开发环境全部通过 Docker：

```bash
docker compose up -d --build
docker compose ps
docker compose logs -f backend
docker compose exec backend go test ./...
docker compose exec backend go vet ./...
docker compose exec backend gofmt -w .
docker compose down
docker compose exec backend go run ./cmd/seed
```

本地开发账号：`admin / admin12345`。

桌面端：

```bash
cd apps/desktop
wails3 dev
wails3 build
```

验证 Compose：

```bash
docker compose config --quiet
```

配置文件：

```text
backend/config/config.example.yml  # 提交
backend/config/config.yml          # 本地生成，不提交
.env.example                       # 提交
.env                               # 本地生成，不提交
```

容器内配置路径：

```text
CONFIG_PATH=/app/config/config.yml
```

不要在宿主机安装或启动 PostgreSQL/Redis 作为开发依赖。

## 6. 安全规则

- 不提交 `config.yml`、`.env`、密钥、Token、数据库密码、私钥和本地配置；只提交 `config.example.yml` 和 `.env.example`。
- 不在日志中输出模型 Key、用户密钥或完整敏感请求。
- 不执行 Agent 生成的任意宿主机命令。
- 不挂载 Docker Socket、用户主目录或系统目录到沙箱。
- 不自动合并主分支、自动部署或自动删除项目。
- 高风险操作必须经过审批。
- 远程接口必须校验设备身份、令牌和权限。

## 7. 文档规则

文档保持少而清晰，只维护：

- `README.md`
- `docs/README.md`
- `docs/tech-stack.md`
- `docs/architecture.md`
- `docs/development.md`
- `AGENTS.md`

不要新增零散设计文档、会议记录或临时说明。需要记录新决策时，直接更新上述文件对应章节。

## 8. 代码质量

- 新增业务逻辑必须有测试。
- 错误带上下文，但不泄露敏感数据。
- 公开 API 使用稳定错误码。
- 日志包含 `request_id`、`project_id`、`task_id` 等必要字段。
- 避免全局可变状态和隐式 `init()` 注册。
- 新增依赖前确认是否必要，优先使用标准库和已有依赖。

## 9. 完成定义

提交前至少确认：

- [ ] 功能符合当前里程碑，不提前扩展。
- [ ] `go test ./...` 通过。
- [ ] `go vet ./...` 通过。
- [ ] `docker compose config --quiet` 通过。
- [ ] 没有新增密钥或敏感配置。
- [ ] 没有破坏文档、迁移或 Docker 开发环境。
- [ ] 文档只在既有文件中更新。

## 10. 下一步优先级

1. 完成 M0 技术验证。
2. 建立数据库迁移和基础仓储层。
3. 实现认证、项目和群聊骨架。
4. 再进入 Agent 编排和 Docker 沙箱执行。

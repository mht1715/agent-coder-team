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

- 当前已有 Go 后端、数据库迁移、本地认证、项目 CRUD、项目群聊、消息 SSE、项目画像、Wails 3 桌面端、Docker 开发环境和健康检查。
- 已实现主 Agent 流式回复、PRD 生成与批准、任务 DAG、任务状态机、任务看板、计划/执行审批和运行记录。
- 任务执行审批已接入 Docker 沙箱执行器：执行计划中的测试/构建命令，记录实时日志、退出码、变更文件，并按结果推进任务状态。
- 主 Agent 仍是本地确定性规划器；模型网关和由模型生成代码修改尚未实现，不要把它描述成已完成。
- 移动端尚未创建，不要提前生成空的移动端目录。
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
- 沙箱容器不得挂载 Docker Socket、用户主目录或系统目录；开发后端容器为创建隔离的兄弟沙箱可访问 Docker Socket，仅限本地开发环境。
- 沙箱必须保持 `network=none`、`cap-drop=ALL`、`no-new-privileges`、CPU/内存/PID/超时限制，并且只共享项目工作区。
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

1. 模型网关、可替换的主 Agent 生成器和执行 Agent。
2. pgvector 索引、上下文预算和长期记忆。
3. Tailscale 移动端派活、设备配对和 TLS。
4. Git worktree 隔离、合并审批和部署流程。

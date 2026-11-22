# 技术选型基线

本文件是项目的最终技术选型。除非出现明确的技术阻塞，否则开发期间不替换核心组件。

## 1. 选型原则

1. 主流开源、文档充足、社区活跃。
2. 本地资源占用低，避免为单用户产品引入重型基础设施。
3. 技术栈数量少，产品自身只使用 Go + Vue。
4. 所有版本锁定，升级通过独立提交完成。
5. 领域层与具体框架解耦，保留可替换性，但不提前抽象。
6. 能用标准库和现有依赖解决的，不新增框架。

## 2. 后端

| 类别 | 选型 | 说明 |
|---|---|---|
| 语言 | Go 1.27 | 与 `go.mod` 一致 |
| Web 框架 | Gin 1.12 | 路由、中间件、HTTP 处理 |
| ORM | GORM v2 | 常规 CRUD；复杂查询写原生 SQL |
| PostgreSQL 驱动 | pgx v5 | GORM postgres driver 底层 |
| 配置 | Viper | YAML + 环境变量 |
| 日志 | zerolog | 结构化 JSON、低分配 |
| 参数校验 | go-playground/validator v10 | Gin 默认集成 |
| 密码哈希 | `golang.org/x/crypto/argon2` | Argon2id |
| 会话 | 不透明 refresh token | 只存哈希，支持设备撤销；不把 JWT 当唯一方案 |
| 任务队列 | Asynq | Redis 持久任务、延迟任务、重试 |
| Redis 客户端 | go-redis v9 | 队列、缓存、限流 |
| 数据库迁移 | golang-migrate | SQL migration，禁用生产 AutoMigrate |
| UUID | google/uuid | UUIDv7，时间有序 |
| CLI | Cobra | `agentd serve/migrate/doctor` |
| API 契约 | OpenAPI 3 + oapi-codegen | 生成 Go 类型和 Gin 接口 |
| HTTP 客户端 | 标准库 `net/http` | 模型适配器统一封装 |
| 模型 SDK | OpenAI-compatible + 各厂商适配器 | DeepSeek/OpenAI 兼容，支持本地 Ollama |
| MCP | modelcontextprotocol/go-sdk | 外部工具接入 |
| 容器 | Docker Engine SDK | 沙箱生命周期管理 |
| 依赖注入 | 显式构造函数 | 不引入 Wire/Fx，减少生成代码和隐式行为 |
| 测试 | Go test + testify + testcontainers-go | 单元 + PostgreSQL/Redis 集成测试 |
| 代码质量 | golangci-lint + gofumpt + govulncheck | CI 必跑 |

### 后端明确不做

- 不引入 Fiber、Echo 等第二套 Web 框架。
- 不引入 GraphQL、gRPC；REST + SSE 足够。
- 不引入 Kafka、RabbitMQ、NATS。
- 不引入 Elasticsearch/OpenSearch；使用 PostgreSQL 全文检索 + pgvector。
- 不引入 Wire/Fx。
- 不引入 Kubernetes。

## 3. 前端

产品自身前端固定 Vue 3。React 和 uni-app 只是 Agent 可开发的目标技术栈。

| 类别 | 选型 | 说明 |
|---|---|---|
| 运行时 | Node.js 22 LTS | 构建和开发 |
| 包管理 | pnpm 10 | workspace + lockfile |
| 框架 | Vue 3 + TypeScript 5 | Composition API |
| 构建 | Vite 7 | 开发与打包 |
| 状态 | Pinia + pinia-plugin-persistedstate | 主题/偏好持久化 |
| 路由 | Vue Router 4 | 桌面/移动路由分离 |
| HTTP | Axios | 移动端和可选 Web 端 |
| 流式事件 | 原生 EventSource/fetch | SSE |
| 桌面 UI | Element Plus | 工作台、表格、树、抽屉 |
| 移动 UI | Vant 4 | 触控和底部导航 |
| 样式 | SCSS 嵌套 + CSS Variables | 主题 Token，统一组件内嵌套写法 |
| 国际化 | vue-i18n | 懒加载语言包 + Intl |
| 工具库 | VueUse | 常用组合式能力 |
| 校验 | Zod | 运行时 Schema 校验 |
| Markdown | markdown-it + Shiki + DOMPurify | 渲染、高亮、防 XSS |
| Diff | diff2html | Diff 审核展示 |
| 虚拟列表 | vue-virtual-scroller | 长聊天/任务列表 |
| 日期 | dayjs | 轻量时间处理 |
| 图标 | Lucide + Vant Icons | Lucide 为 ISC 许可，免费商用；不使用 emoji 代替图标 |
| 单元测试 | Vitest + Vue Test Utils | 组件和组合式函数 |
| E2E | Playwright | 关键用户流程 |
| Mock | MSW | API mock |
| 代码质量 | ESLint 9 + Prettier 3 + Stylelint + vue-tsc | CI 必跑 |

### 前端明确不做

- 产品自身不使用 React。
- 不引入 Nx、Turborepo，使用 pnpm workspace + scripts。
- 不引入 UnoCSS/Tailwind，统一 SCSS + CSS Variables。
- 不引入多套组件库。
- 不引入 Redux 等非 Vue 状态方案。

## 4. 数据与基础设施

| 类别 | 选型 |
|---|---|
| 关系数据库 | PostgreSQL 16 |
| 向量检索 | pgvector（MVP），Milvus 后续 Full 档 |
| 缓存/队列 | Redis 7 + Asynq |
| 文件与产物 | 本地文件系统 |
| 开发环境 | Docker Compose v2 |
| 沙箱 | 非 root Docker 容器 |
| 远程组网 | Tailscale；完全自托管可用 Headscale |
| 传输 | 移动端 HTTPS/TLS 1.3；桌面 Wails IPC |
| CI | GitHub Actions |
| 发布 | Wails build + GoReleaser |
| 依赖安全 | govulncheck + Trivy + Dependabot |

### 明确不做

- 不使用 SQLite 代替 PostgreSQL。
- 不使用 MongoDB。
- MVP 不部署 Milvus。
- 不使用 MinIO/S3，文件先落本地。
- 不使用 Nginx/Caddy 作为本地入口；Tailscale 负责私网访问。
## 5. 桌面与移动

| 端 | 选型 |
|---|---|
| 桌面壳 | Wails 3.x，精确版本写入 `wails.json` 和 `go.mod` |
| 桌面 UI | Vue 3 + Element Plus |
| 桌面通信 | Wails IPC/事件，不开放本地明文 HTTP |
| 移动壳 | 优先 Wails 移动端 |
| 移动备选 | Capacitor，仅在 M0 证明 Wails 移动端不可用时启用 |
| 移动 UI | Vue 3 + Vant |
| 移动通信 | HTTPS + REST + SSE |
| 证书 | 应用本地 CA 或 Tailscale 证书 + 证书固定 |
| 设备认证 | 设备密钥签名 + 短期 access token |

### 移动端验证门槛

M0 必须验证 Wails 移动端的：

- 扫码/相机权限。
- Keychain/Keystore。
- 后台切换和重连。
- SSE 或事件流。
- 应用签名、安装和升级。
- iOS/Android 发布流程。

不达标才启用 Capacitor，不提前引入第二套移动壳。

## 6. Agent 目标技术栈

Agent 支持的技术栈不是产品自身依赖。Agent 通过 Stack Profile 和 Tech Pack 适配目标项目。

| 技术栈 | 常用框架/工具 | 支持范围 |
|---|---|---|
| Go | Gin、GORM、标准库、Go test | 完整 |
| Vue 3 | Vite、Pinia、Vue Router、Element Plus/Vant、Vitest | 完整 |
| React | Vite、Next.js、React Router、Zustand/Redux、Vitest | 完整 |
| Node.js | Express、NestJS、Next.js、pnpm/npm、Jest/Vitest | 完整 |
| Python | uv/pip、FastAPI、Django、pytest | 完整 |
| Java | Maven/Gradle、Spring Boot、JUnit 5 | 完整 |
| .NET | dotnet CLI、ASP.NET Core、xUnit | 基础到完整 |
| uni-app | Vue 语法、CLI、H5 构建 | 基础；原生打包后续 |

每个 Stack Profile 至少包含：

- 项目检测规则。
- 构建、测试、格式化、Lint 命令。
- 目录约定和框架约定。
- 沙箱镜像。
- 常见错误和安全注意事项。
- 推荐的 Tech Pack。

## 7. 版本与升级策略

- `go.mod`、`go.sum`、`pnpm-lock.yaml`、`wails.json` 必须提交。
- 不在配置中使用 `latest` 镜像；生产镜像固定 digest。
- 依赖升级使用独立 PR，并跑完整测试。
- 升级 Go、Node、Wails、PostgreSQL 大版本前先写迁移方案。
- 不频繁替换框架；确需替换时更新本文件和 `docs/architecture.md`。
- 新依赖必须说明：解决什么问题、为什么标准库/已有依赖不够、资源开销和维护状态。

## 8. 选型总结

最终主线只有四条：

```text
Go + Gin + GORM + PostgreSQL/pgvector + Redis/Asynq
Vue 3 + TypeScript + Vite + Element Plus/Vant
Wails 桌面/移动
Docker + Tailscale
```

React、Java、Python、Node、.NET、uni-app 是 Agent 要支持的目标项目技术栈，不是产品自身运行时。这样既覆盖常见开发场景，又不让产品本身变成多技术栈拼盘。
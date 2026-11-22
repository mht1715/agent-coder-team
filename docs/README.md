# Agent Coder Team

本地优先的 "Agent 软件公司"：用户提出需求并验收，主 Agent 负责澄清和规划，多角色 Agent 在 Docker 沙箱中完成开发。

## 1. 核心目标

- 降低 token 成本：增量索引、按需上下文、分层记忆、模型路由。
- 降低使用门槛：Docker 一键启动开发依赖，桌面端本地使用。
- 提高可控性：需求、计划、合并、部署均有审批门禁。
- 提高安全性：代码只在沙箱执行，远程传输必须加密。
- 支持远程：手机通过 Tailscale 查看进度、派活和审批，不需要公网 IP。

## 2. 产品原则

1. **状态机优先，不做自由 Agent 群聊。** 主 Agent 独占用户对话，执行 Agent 只接收结构化任务。
2. **本地优先。** 数据、密钥、代码默认只留在用户电脑。
3. **预算先行。** 每个任务都有 token、步数、时间和文件访问范围。
4. **增量优先。** 索引、摘要、记忆、构建都尽量增量，禁止反复全仓扫描。
5. **执行与决策分离。** 代码在沙箱运行，合并和部署必须人工确认。
6. **少技术栈。** 产品自身前端固定 Vue 3；不为每个目标语言复制一套 Agent。

## 3. 核心流程

```text
用户提需求
  -> 主 Agent 澄清
  -> 生成 PRD 和验收标准
  -> 用户确认需求
  -> 架构 Agent 生成任务 DAG
  -> 用户确认计划
  -> 到达开工窗口或手动开工
  -> 执行 Agent 在 worktree + Docker 沙箱中开发
  -> 测试 Agent / 审查 Agent 验证
  -> 用户查看 Diff 和测试报告
  -> 用户批准合并
  -> 本地预览
```

## 4. MVP 范围

### 必须完成

- 本地账号、个人资料、偏好、主题。
- 项目列表、创建项目、导入已有 Git 项目。
- 项目群聊、流式回复、原文/摘要双视图。
- 主 Agent 需求澄清、PRD、任务 DAG。
- 计划审批、任务状态、失败重试、运行详情。
- 至少一个完整全栈示例：Go + Vue。
- Docker 沙箱执行构建和测试。
- 本地预览。
- pgvector 增量索引、混合检索、基础记忆。
- 桌面端深色/浅色/自定义主题。
- 中文/英文 i18n 骨架。
- Tailscale + HTTPS + 设备配对原型。

### 暂不实现

- 多人协作、组织/租户、复杂 RBAC。
- 真实支付和套餐计费。
- Milvus 默认部署。
- 自动合并到主分支。
- 云服务器自动部署。
- uni-app 原生打包。

## 5. 最终决策基线

| 事项 | 决策 |
|---|---|
| Docker | 开发环境必须使用 Docker；引导安装 Docker Desktop，兼容 Docker Engine/Podman |
| 数据库 | PostgreSQL |
| 向量检索 | MVP 使用 pgvector；Milvus 后续作为 Full 档 |
| 缓存/队列 | Redis 7 + Asynq |
| 桌面 | Wails 3 + Vue 3 + Element Plus |
| 移动 | 优先 Wails 3 + Vue 3 + Vant；Capacitor 仅作兜底 |
| 后端 | Go + Gin + GORM + Viper |
| 目标技术栈 | Go、Vue、React、Java、Python、Node、.NET、uni-app |
| 产品自身前端 | 固定 Vue 3，不使用 React/uni-app |
| 远程访问 | Tailscale/Headscale + TLS 1.3 + 证书固定 + 设备签名 |
| 桌面传输 | Wails IPC，不开放本地明文 HTTP |
| 商业模式 | 免费版 + Pro 订阅/本地授权 + 托管服务的混合模式；支持 BYOK |
| 开工时间 | 自动执行窗口与模型低价时段解耦，支持成本优先调度 |
| 团队消息 | 可生成角色化消息，但必须绑定真实任务/事件 |

## 6. 技术栈摘要

| 层 | 技术                                                                |
|---|---------------------------------------------------------------------|
| 后端 | Gin、GORM、Viper、zerolog、validator、Asynq、Cobra                  |
| 数据 | PostgreSQL 16 + pgvector、Redis 7 + Asynq、Docker                   |
| 前端 | Vue 3、TypeScript、Vite、Pinia、Vue Router、Axios、Scss             |
| 桌面 UI | Element Plus                                                        |
| 移动 UI | Vant                                                                |
| 国际化 | vue-i18n + Intl + 后端错误码                                        |
| 测试 | Go test、testify、testcontainers-go、Vitest、Playwright             |
| 工程化 | golangci-lint、gofumpt、ESLint、Prettier、Stylelint、GitHub Actions |

完整技术选型、版本策略和 Agent 目标技术栈见 [tech-stack.md](./tech-stack.md)。

## 7. Agent 团队

| 角色 | 职责 |
|---|---|
| 主 Agent / PM | 澄清需求、PRD、任务分解、汇总和审批 |
| 架构 Agent | 方案、接口、模块边界、任务 DAG |
| UI/UX Agent | 信息架构、交互和设计 Token |
| 数据库 Agent | 模型、迁移、索引和查询 |
| 后端 Agent | Go/Java/Python/Node/.NET 后端任务 |
| 前端 Agent | Vue/React/uni-app 前端任务 |
| 测试 Agent | 测试生成、执行和失败定位 |
| 审查 Agent | 代码、安全和验收标准审查 |
| DevOps Agent | Dockerfile、Compose、CI 和预览 |

角色是 "权限 + 提示词 + 模型策略 + 工具集" 的组合，不是不同的人格。

## 8. 路线图

| 阶段 | 目标 |
|---|---|
| M0 | 技术验证：Wails IPC、Docker 沙箱、pgvector、Tailscale/TLS |
| M1 | 认证、项目、群聊、主题、i18n 骨架、Docker 开发环境 |
| M2 | 主 Agent、PRD、任务 DAG、审批、任务状态机 |
| M3 | 增量索引、Git worktree、沙箱执行、Diff、本地预览 |
| M4 | 移动远程、设备配对、证书固定、审批 |
| M5 | 稳定性、备份、诊断、安装包和发布 |

## 9. 待确认

1. 原文和摘要的保留策略。
2. 首批国际化语言是否只有中文、英文。
3. Pro 版本的具体功能边界和定价。
4. uni-app 原生打包的平台范围和时间点。

## 10. 文档

- [技术选型](./tech-stack.md)
- [架构设计](./architecture.md)
- [开发指南](./development.md)
- [AGENTS.md](../AGENTS.md)

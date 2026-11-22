package agent

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"backend/internal/domain"

	"github.com/google/uuid"
)

type plannedTask struct {
	Key                string
	Title              string
	Description        string
	AcceptanceCriteria string
	AgentRole          string
	Priority           string
	DependsOn          []string
}

func isEnglishProject(project *domain.Project) bool {
	return project != nil && strings.HasPrefix(strings.ToLower(project.OutputLanguage), "en")
}

func detectIntent(content string) (wantRequirement, wantPlan bool) {
	value := strings.ToLower(content)
	requirementKeywords := []string{
		"prd", "需求文档", "产品需求", "生成需求", "需求说明", "需求规格",
		"requirement document", "product requirement", "generate requirement",
	}
	planKeywords := []string{
		"拆解任务", "生成任务", "任务计划", "任务分解", "任务清单", "plan tasks",
		"task breakdown", "generate tasks", "task plan", "break down tasks",
	}
	for _, keyword := range requirementKeywords {
		if strings.Contains(value, keyword) {
			wantRequirement = true
			break
		}
	}
	for _, keyword := range planKeywords {
		if strings.Contains(value, keyword) {
			wantPlan = true
			break
		}
	}
	return wantRequirement, wantPlan
}

func composeReply(
	project *domain.Project,
	latest string,
	requirement *domain.RequirementDocument,
	tasks []domain.Task,
	approval *domain.Approval,
	wantRequirement, wantPlan bool,
) string {
	english := isEnglishProject(project)
	summary := strings.TrimSpace(latest)
	if utf8.RuneCountInString(summary) > 180 {
		summary = string([]rune(summary)[:180]) + "..."
	}

	if english {
		var builder strings.Builder
		builder.WriteString("I have recorded your latest input:\n\n> ")
		builder.WriteString(strings.ReplaceAll(summary, "\n", " "))
		builder.WriteString("\n\n")
		if wantRequirement && requirement != nil {
			builder.WriteString(fmt.Sprintf("The product requirement document **%s** (version %d) has been generated as a draft. Review it in the Requirements tab and approve it before execution.\n\n", requirement.Title, requirement.Version))
		}
		if wantPlan && len(tasks) > 0 {
			builder.WriteString(fmt.Sprintf("I generated a task DAG with **%d tasks**. It is waiting for plan approval in the Approvals tab. After approval, tasks with satisfied dependencies become ready.\n\n", len(tasks)))
			if approval != nil {
				builder.WriteString(fmt.Sprintf("Approval: **%s**.\n\n", approval.Title))
			}
		} else if wantPlan {
			builder.WriteString("The task plan could not be generated. Check whether a plan already exists and review the project status.\n\n")
		}
		if !wantRequirement && !wantPlan {
			builder.WriteString("To move from conversation to execution, provide the target users, core scenarios, constraints and acceptance criteria. You can then say **generate PRD** or use the action in the Requirements tab.\n\n")
			builder.WriteString("Recommended next steps:\n\n1. Confirm the first milestone and explicit non-goals.\n2. Confirm permissions, data retention and integration constraints.\n3. Generate and approve the PRD.\n4. Generate the task DAG and approve the plan.")
		} else {
			builder.WriteString("Next step: review the generated artifacts, close open questions, and approve the plan before starting a task.")
		}
		return builder.String()
	}

	var builder strings.Builder
	builder.WriteString("我已记录你最新的输入：\n\n> ")
	builder.WriteString(strings.ReplaceAll(summary, "\n", " "))
	builder.WriteString("\n\n")
	if wantRequirement && requirement != nil {
		builder.WriteString(fmt.Sprintf("产品需求文档《%s》第 %d 版已生成草稿。请在需求页审阅，确认后批准，再进入任务拆解。\n\n", requirement.Title, requirement.Version))
	}
	if wantPlan && len(tasks) > 0 {
		builder.WriteString(fmt.Sprintf("我已生成包含 **%d 个任务** 的任务 DAG，并在审批与执行页提交了计划审批。批准后，依赖已满足的任务会自动进入可执行状态。\n\n", len(tasks)))
		if approval != nil {
			builder.WriteString(fmt.Sprintf("待审批事项：**%s**。\n\n", approval.Title))
		}
	} else if wantPlan {
		builder.WriteString("任务计划未能生成。请确认当前项目是否已有计划，或先完善需求文档。\n\n")
	}
	if !wantRequirement && !wantPlan {
		builder.WriteString("要把对话推进到可执行计划，请继续补充目标用户、核心场景、业务约束和验收标准。完成后可以回复生成 PRD，或直接在需求页生成需求文档。\n\n")
		builder.WriteString("建议下一步：\n\n1. 确认首个里程碑和明确不做的范围。\n2. 确认权限、数据保留和外部集成约束。\n3. 生成并批准 PRD。\n4. 生成任务 DAG 并批准计划。")
	} else {
		builder.WriteString("下一步：审阅生成的文档和任务，关闭待确认问题，并在开始执行前批准计划。")
	}
	return builder.String()
}

func buildRequirementDocument(
	project *domain.Project,
	profile *domain.ProjectProfile,
	messages []domain.Message,
) (string, string) {
	english := isEnglishProject(project)
	userInputs := make([]string, 0)
	for i := len(messages) - 1; i >= 0 && len(userInputs) < 12; i-- {
		if messages[i].SenderType != domain.SenderTypeUser {
			continue
		}
		value := strings.TrimSpace(messages[i].Content)
		if value == "" {
			continue
		}
		if utf8.RuneCountInString(value) > 600 {
			value = string([]rune(value)[:600]) + "..."
		}
		userInputs = append(userInputs, value)
	}
	for left, right := 0, len(userInputs)-1; left < right; left, right = left+1, right-1 {
		userInputs[left], userInputs[right] = userInputs[right], userInputs[left]
	}

	stack := "待补充"
	if profile != nil {
		parts := make([]string, 0)
		if profile.PrimaryLanguage != "" {
			parts = append(parts, profile.PrimaryLanguage)
		}
		parts = append(parts, profile.Frameworks...)
		if profile.PackageManager != "" {
			parts = append(parts, profile.PackageManager)
		}
		if len(parts) > 0 {
			stack = strings.Join(parts, " / ")
		}
	}

	if english {
		title := project.Name + " Product Requirements Document"
		var b strings.Builder
		b.WriteString("# " + title + "\n\n")
		b.WriteString("## 1. Document status\n\n")
		b.WriteString("- Status: draft for review\n")
		b.WriteString("- Generated by: Main Agent\n")
		b.WriteString(fmt.Sprintf("- Project mode: %s\n", project.ProjectMode))
		b.WriteString(fmt.Sprintf("- Technical baseline: %s\n\n", stack))
		b.WriteString("## 2. Product context and goal\n\n")
		b.WriteString(project.Description)
		if strings.TrimSpace(project.Description) == "" {
			b.WriteString("The project goal is derived from the requirement conversation and must be confirmed by the product owner.")
		}
		b.WriteString("\n\n")
		b.WriteString("## 3. Target users\n\n")
		b.WriteString("- Primary users: to be confirmed during requirement review.\n")
		b.WriteString("- Secondary users and administrators: to be confirmed.\n\n")
		b.WriteString("## 4. Scope\n\n")
		b.WriteString("### In scope for the first milestone\n\n")
		if len(userInputs) == 0 {
			b.WriteString("- Deliver the first verified user journey described in the project brief.\n")
		} else {
			for i, input := range userInputs {
				b.WriteString(fmt.Sprintf("%d. %s\n", i+1, oneLine(input)))
			}
		}
		b.WriteString("\n### Explicit non-goals for the first milestone\n\n")
		b.WriteString("- Advanced automation and integrations not listed in the confirmed scope.\n")
		b.WriteString("- Large-scale migration and multi-region deployment unless separately approved.\n\n")
		b.WriteString("## 5. Functional requirements\n\n")
		b.WriteString("| ID | Requirement | Priority | Acceptance evidence |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		if len(userInputs) == 0 {
			b.WriteString("| FR-01 | Complete the primary user journey | P0 | Product owner validates end-to-end behavior |\n")
		} else {
			for i, input := range userInputs {
				b.WriteString(fmt.Sprintf("| FR-%02d | %s | P0 | Verifiable in the UI/API and covered by acceptance checks |\n", i+1, escapeTable(oneLine(input))))
			}
		}
		b.WriteString("\n## 6. Non-functional requirements\n\n")
		b.WriteString("- Security: authenticated access, least privilege, protected secrets and auditable mutations.\n")
		b.WriteString("- Reliability: deterministic error handling, idempotent retries and recoverable job states.\n")
		b.WriteString("- Performance: define measurable latency and throughput targets before release.\n")
		b.WriteString("- Observability: structured logs, run records and actionable failure messages.\n\n")
		b.WriteString("## 7. Acceptance criteria\n\n")
		b.WriteString("1. Every P0 functional requirement has a reproducible acceptance check.\n")
		b.WriteString("2. Permission boundaries and failure paths are verified.\n")
		b.WriteString("3. Build, automated tests and packaging complete from a clean checkout.\n")
		b.WriteString("4. The owner can review execution records and approve completion.\n\n")
		b.WriteString("## 8. Open questions\n\n")
		appendOpenQuestions(&b, userInputs, true)
		return title, b.String()
	}
	title := project.Name + " 产品需求文档"
	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	b.WriteString("## 1. 文档状态\n\n")
	b.WriteString("- 状态：待评审草稿\n")
	b.WriteString("- 生成方：主 Agent\n")
	b.WriteString("- 项目类型：" + projectModeLabel(project.ProjectMode) + "\n")
	b.WriteString("- 技术基线：" + stack + "\n\n")
	b.WriteString("## 2. 产品背景与目标\n\n")
	if strings.TrimSpace(project.Description) != "" {
		b.WriteString(project.Description)
	} else {
		b.WriteString("项目目标来自当前需求对话，需要由项目负责人最终确认。")
	}
	b.WriteString("\n\n## 3. 目标用户\n\n")
	b.WriteString("- 主要用户：待需求评审确认。\n")
	b.WriteString("- 次要用户与管理员：待确认。\n\n")
	b.WriteString("## 4. 范围\n\n")
	b.WriteString("### 首个里程碑包含\n\n")
	if len(userInputs) == 0 {
		b.WriteString("- 交付项目简介中描述的首个可验证用户旅程。\n")
	} else {
		for i, input := range userInputs {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, oneLine(input)))
		}
	}
	b.WriteString("\n### 首期明确不做\n\n")
	b.WriteString("- 未在确认范围中列出的高级自动化和外部集成。\n")
	b.WriteString("- 未经单独审批的大规模数据迁移和多区域部署。\n\n")
	b.WriteString("## 5. 功能需求\n\n")
	b.WriteString("| 编号 | 需求 | 优先级 | 验收证据 |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	if len(userInputs) == 0 {
		b.WriteString("| FR-01 | 完成核心用户旅程 | P0 | 项目负责人完成端到端验证 |\n")
	} else {
		for i, input := range userInputs {
			b.WriteString(fmt.Sprintf("| FR-%02d | %s | P0 | 可在界面或接口中验证，并有验收检查覆盖 |\n", i+1, escapeTable(oneLine(input))))
		}
	}
	b.WriteString("\n## 6. 非功能需求\n\n")
	b.WriteString("- 安全：登录鉴权、最小权限、密钥保护和关键变更审计。\n")
	b.WriteString("- 可靠性：错误处理确定、重试幂等、任务状态可恢复。\n")
	b.WriteString("- 性能：发布前明确可测量的延迟和吞吐目标。\n")
	b.WriteString("- 可观测性：结构化日志、执行记录和可操作失败信息。\n\n")
	b.WriteString("## 7. 验收标准\n\n")
	b.WriteString("1. 所有 P0 功能需求都有可复现的验收检查。\n")
	b.WriteString("2. 权限边界和主要失败路径已经验证。\n")
	b.WriteString("3. 干净检出后可以完成构建、自动化测试和打包。\n")
	b.WriteString("4. 项目负责人可以查看执行记录并审批完成。\n\n")
	b.WriteString("## 8. 待确认问题\n\n")
	appendOpenQuestions(&b, userInputs, false)
	return title, b.String()
}

func appendOpenQuestions(b *strings.Builder, userInputs []string, english bool) {
	hasUser := containsAny(userInputs, "用户", "角色", "user", "persona", "customer")
	hasAcceptance := containsAny(userInputs, "验收", "acceptance", "done", "完成标准")
	hasNonGoal := containsAny(userInputs, "不做", "范围外", "non-goal", "out of scope")
	hasTimeline := containsAny(userInputs, "里程碑", "上线", "期限", "milestone", "deadline", "release")
	questions := make([]string, 0)
	if !hasUser {
		if english {
			questions = append(questions, "Who are the primary and secondary users, and what permissions do they need?")
		} else {
			questions = append(questions, "主要用户和次要用户分别是谁？各自需要哪些权限？")
		}
	}
	if !hasAcceptance {
		if english {
			questions = append(questions, "What observable evidence proves each P0 requirement is complete?")
		} else {
			questions = append(questions, "每个 P0 需求用什么可观察的证据证明已经完成？")
		}
	}
	if !hasNonGoal {
		if english {
			questions = append(questions, "Which capabilities are explicitly out of scope for the first milestone?")
		} else {
			questions = append(questions, "首个里程碑明确不做哪些能力？")
		}
	}
	if !hasTimeline {
		if english {
			questions = append(questions, "What is the target milestone date or release window?")
		} else {
			questions = append(questions, "目标里程碑日期或发布窗口是什么？")
		}
	}
	if len(questions) == 0 {
		if english {
			b.WriteString("- No blocking questions detected. Confirm data retention and production operations before approval.\n")
		} else {
			b.WriteString("- 暂未识别到阻塞性问题。批准前请确认数据保留策略和生产运维方式。\n")
		}
		return
	}
	for i, question := range questions {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, question))
	}
}

func buildTaskPlan(
	project *domain.Project,
	profile *domain.ProjectProfile,
	requirement *domain.RequirementDocument,
) []plannedTask {
	english := isEnglishProject(project)
	stack := "项目现有技术栈"
	if profile != nil {
		parts := make([]string, 0)
		if profile.PrimaryLanguage != "" {
			parts = append(parts, profile.PrimaryLanguage)
		}
		parts = append(parts, profile.Frameworks...)
		if len(parts) > 0 {
			stack = strings.Join(parts, " / ")
		}
	}
	requirementRef := requirement.Title
	if english {
		return []plannedTask{
			{Key: "T-001", Title: "Confirm requirement baseline and acceptance criteria", Description: "Review \"" + requirementRef + "\", close blocking questions and freeze the first milestone scope. Technical baseline: " + stack + ".", AcceptanceCriteria: "The PRD is approved; every P0 item has an observable acceptance check and explicit non-goals.", AgentRole: "product_manager", Priority: "critical"},
			{Key: "T-002", Title: "Design architecture and delivery boundaries", Description: "Translate the approved PRD into module boundaries, data ownership, API contracts, security boundaries and rollout strategy.", AcceptanceCriteria: "Architecture notes and interface contracts are reviewed; unresolved technical risks have owners.", AgentRole: "architect", Priority: "critical", DependsOn: []string{"T-001"}},
			{Key: "T-003", Title: "Implement data model and migrations", Description: "Create the persisted entities, constraints, indexes and reversible migrations required by the approved scope.", AcceptanceCriteria: "Migrations apply and roll back cleanly; model constraints are covered by automated checks.", AgentRole: "backend_engineer", Priority: "high", DependsOn: []string{"T-002"}},
			{Key: "T-004", Title: "Define API contracts and backend foundation", Description: "Implement authentication-aware API contracts, validation, error envelopes and service boundaries for the first milestone.", AcceptanceCriteria: "Core endpoints match the contract and pass integration checks for success and failure paths.", AgentRole: "backend_engineer", Priority: "high", DependsOn: []string{"T-003"}},
			{Key: "T-005", Title: "Build frontend shell and state foundation", Description: "Implement the page shell, routing, state management, API adapter and reusable visual primitives for the approved journey.", AcceptanceCriteria: "The primary navigation and state restoration work across reloads and back navigation.", AgentRole: "frontend_engineer", Priority: "high", DependsOn: []string{"T-002"}},
			{Key: "T-006", Title: "Implement backend core business flows", Description: "Implement the P0 business operations, permission checks, persistence and event publication required by the PRD.", AcceptanceCriteria: "P0 backend flows have integration coverage and produce auditable state changes.", AgentRole: "backend_engineer", Priority: "critical", DependsOn: []string{"T-004"}},
			{Key: "T-007", Title: "Implement core product experience", Description: "Connect the frontend to the approved APIs and deliver the primary user journey with loading, empty, error and permission states.", AcceptanceCriteria: "The primary user journey completes end-to-end without manual data repair.", AgentRole: "frontend_engineer", Priority: "critical", DependsOn: []string{"T-004", "T-005"}},
			{Key: "T-008", Title: "Integrate, harden and audit the workflow", Description: "Integrate backend and frontend flows, add permission and failure-path handling, and verify execution records and approvals.", AcceptanceCriteria: "Cross-layer integration checks pass; security and failure paths are documented.", AgentRole: "fullstack_engineer", Priority: "high", DependsOn: []string{"T-006", "T-007"}},
			{Key: "T-009", Title: "Run acceptance tests and release verification", Description: "Execute automated and manual acceptance checks derived from the PRD, package the application and record evidence.", AcceptanceCriteria: "Acceptance evidence is attached; build and test commands pass from a clean checkout.", AgentRole: "qa_engineer", Priority: "critical", DependsOn: []string{"T-008"}},
			{Key: "T-010", Title: "Publish delivery documentation and handoff", Description: "Update setup, operations, architecture and user-facing documentation; record known limitations and next actions.", AcceptanceCriteria: "A new operator can start the local stack and complete the primary journey from the documentation.", AgentRole: "tech_writer", Priority: "medium", DependsOn: []string{"T-009"}},
		}
	}
	return []plannedTask{
		{Key: "T-001", Title: "确认需求基线与验收标准", Description: "评审《" + requirementRef + "》，关闭阻塞性问题，冻结首个里程碑范围。技术基线：" + stack + "。", AcceptanceCriteria: "PRD 已批准；每个 P0 需求都有可观察的验收检查，并明确首期不做的范围。", AgentRole: "product_manager", Priority: "critical"},
		{Key: "T-002", Title: "设计系统架构与交付边界", Description: "把已批准的需求转化为模块边界、数据归属、接口契约、安全边界和发布策略。", AcceptanceCriteria: "架构说明和接口契约完成评审；未解决的技术风险都有负责人。", AgentRole: "architect", Priority: "critical", DependsOn: []string{"T-001"}},
		{Key: "T-003", Title: "实现数据模型与数据库迁移", Description: "为已批准范围创建持久化实体、约束、索引和可回滚的数据库迁移。", AcceptanceCriteria: "迁移可以正向执行和回滚；关键模型约束有自动化检查覆盖。", AgentRole: "backend_engineer", Priority: "high", DependsOn: []string{"T-002"}},
		{Key: "T-004", Title: "定义 API 契约与后端基础能力", Description: "实现鉴权后的接口契约、参数校验、错误响应和首个里程碑所需的服务边界。", AcceptanceCriteria: "核心接口符合契约，并通过成功路径和失败路径的集成检查。", AgentRole: "backend_engineer", Priority: "high", DependsOn: []string{"T-003"}},
		{Key: "T-005", Title: "搭建前端壳层与状态基础", Description: "实现页面框架、路由、状态管理、API 适配器和首个用户旅程所需的基础组件。", AcceptanceCriteria: "主要导航、状态恢复和返回行为在刷新与路由切换后保持正确。", AgentRole: "frontend_engineer", Priority: "high", DependsOn: []string{"T-002"}},
		{Key: "T-006", Title: "实现后端核心业务闭环", Description: "实现 PRD 中的 P0 业务操作、权限检查、持久化和事件发布。", AcceptanceCriteria: "P0 后端流程有集成测试覆盖，并产生可审计的状态变更。", AgentRole: "backend_engineer", Priority: "critical", DependsOn: []string{"T-004"}},
		{Key: "T-007", Title: "实现核心产品交互与页面", Description: "将前端接入已批准的 API，完成主要用户旅程，并覆盖加载、空状态、错误和权限场景。", AcceptanceCriteria: "主要用户旅程可以端到端完成，无需手工修复数据。", AgentRole: "frontend_engineer", Priority: "critical", DependsOn: []string{"T-004", "T-005"}},
		{Key: "T-008", Title: "联调、加固与流程审计", Description: "完成前后端联调，补齐权限和失败路径处理，并验证审批与执行记录链路。", AcceptanceCriteria: "跨层集成检查通过；安全和失败路径已记录并有对应验证。", AgentRole: "fullstack_engineer", Priority: "high", DependsOn: []string{"T-006", "T-007"}},
		{Key: "T-009", Title: "执行验收测试与发布验证", Description: "执行从 PRD 衍生的自动化和人工验收检查，完成应用打包并记录验收证据。", AcceptanceCriteria: "验收证据已归档；干净检出后构建和测试命令通过。", AgentRole: "qa_engineer", Priority: "critical", DependsOn: []string{"T-008"}},
		{Key: "T-010", Title: "完成交付文档与交接", Description: "更新安装、运维、架构和用户文档，记录已知限制和后续行动项。", AcceptanceCriteria: "新的操作者可以依据文档启动本地环境并完成主要用户旅程。", AgentRole: "tech_writer", Priority: "medium", DependsOn: []string{"T-009"}},
	}
}

func buildDomainTasks(
	projectID uuid.UUID,
	requirement *domain.RequirementDocument,
	raw []plannedTask,
) ([]domain.Task, error) {
	now := time.Now().UTC()
	idsByKey := make(map[string]uuid.UUID, len(raw))
	for _, item := range raw {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		idsByKey[item.Key] = id
	}
	tasks := make([]domain.Task, 0, len(raw))
	for index, item := range raw {
		dependencies := make([]uuid.UUID, 0, len(item.DependsOn))
		for _, key := range item.DependsOn {
			id, ok := idsByKey[key]
			if !ok {
				return nil, fmt.Errorf("%w: task %s depends on unknown task %s", ErrValidation, item.Key, key)
			}
			dependencies = append(dependencies, id)
		}
		tasks = append(tasks, domain.Task{
			ID:                 idsByKey[item.Key],
			ProjectID:          projectID,
			TaskKey:            item.Key,
			Title:              item.Title,
			Description:        item.Description,
			AcceptanceCriteria: item.AcceptanceCriteria,
			AgentRole:          item.AgentRole,
			Priority:           item.Priority,
			Status:             domain.TaskStatusBacklog,
			Position:           index,
			Dependencies:       dependencies,
			CreatedAt:          now,
			UpdatedAt:          now,
		})
	}
	if err := validateTaskDAG(tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func validateTaskDAG(tasks []domain.Task) error {
	byID := make(map[uuid.UUID]domain.Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}
	visiting := make(map[uuid.UUID]bool)
	visited := make(map[uuid.UUID]bool)
	var visit func(uuid.UUID) error
	visit = func(id uuid.UUID) error {
		if visiting[id] {
			return fmt.Errorf("%w: task dependency cycle detected", ErrValidation)
		}
		if visited[id] {
			return nil
		}
		task, ok := byID[id]
		if !ok {
			return fmt.Errorf("%w: task dependency refers to a missing task", ErrValidation)
		}
		visiting[id] = true
		for _, dependency := range task.Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		delete(visiting, id)
		visited[id] = true
		return nil
	}
	for _, task := range tasks {
		if err := visit(task.ID); err != nil {
			return err
		}
	}
	return nil
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func escapeTable(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func containsAny(values []string, needles ...string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, needle := range needles {
			if strings.Contains(lower, strings.ToLower(needle)) {
				return true
			}
		}
	}
	return false
}

func projectModeLabel(mode string) string {
	if mode == domain.ProjectModeExisting {
		return "导入已有项目"
	}
	return "新建项目"
}

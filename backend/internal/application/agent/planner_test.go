package agent

import (
	"errors"
	"testing"

	"backend/internal/domain"

	"github.com/google/uuid"
)

func TestDetectIntent(t *testing.T) {
	tests := []struct {
		content     string
		requirement bool
		plan        bool
	}{
		{content: "请生成 PRD", requirement: true, plan: false},
		{content: "请拆解任务并生成任务计划", requirement: false, plan: true},
		{content: "generate task breakdown", requirement: false, plan: true},
		{content: "先聊聊目标用户", requirement: false, plan: false},
	}
	for _, test := range tests {
		requirement, plan := detectIntent(test.content)
		if requirement != test.requirement || plan != test.plan {
			t.Fatalf("detectIntent(%q) = (%v, %v), want (%v, %v)", test.content, requirement, plan, test.requirement, test.plan)
		}
	}
}

func TestBuildDomainTasksValidatesDAG(t *testing.T) {
	projectID := uuid.New()
	requirement := &domain.RequirementDocument{Title: "Test PRD"}
	valid := []plannedTask{
		{Key: "T-001", Title: "Requirement", AgentRole: "product_manager"},
		{Key: "T-002", Title: "Design", AgentRole: "architect", DependsOn: []string{"T-001"}},
		{Key: "T-003", Title: "Build", AgentRole: "backend_engineer", DependsOn: []string{"T-002"}},
	}
	tasks, err := buildDomainTasks(projectID, requirement, valid)
	if err != nil {
		t.Fatalf("buildDomainTasks() error = %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(tasks))
	}
	if len(tasks[2].Dependencies) != 1 {
		t.Fatalf("dependency count = %d, want 1", len(tasks[2].Dependencies))
	}

	cycle := []plannedTask{
		{Key: "T-001", DependsOn: []string{"T-002"}},
		{Key: "T-002", DependsOn: []string{"T-001"}},
	}
	if _, err := buildDomainTasks(projectID, requirement, cycle); !errors.Is(err, ErrValidation) {
		t.Fatalf("cycle error = %v, want ErrValidation", err)
	}

	missing := []plannedTask{{Key: "T-001", DependsOn: []string{"T-404"}}}
	if _, err := buildDomainTasks(projectID, requirement, missing); !errors.Is(err, ErrValidation) {
		t.Fatalf("missing dependency error = %v, want ErrValidation", err)
	}
}

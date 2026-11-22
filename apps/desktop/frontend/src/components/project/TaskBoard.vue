<script setup lang="ts">
import { computed, onActivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CheckCheck, CirclePlay, GitBranch, ListTree, RefreshCw, Undo2 } from 'lucide-vue-next'
import {
  generatePlan,
  listApprovals,
  listTasks,
  requestTaskExecution,
  transitionTask,
} from '../../api/agent'
import { translateApiError } from '../../utils/error'
import type { Approval, Task, TaskStatus } from '../../types/api'

const props = defineProps<{ projectId: string }>()
const emit = defineEmits<{ (event: 'open-tab', tab: 'requirements' | 'activity'): void }>()
const { t } = useI18n()

const tasks = ref<Task[]>([])
const pendingApprovals = ref<Approval[]>([])
const loading = ref(false)
const generating = ref(false)
const actionTaskId = ref('')

const columns: { status: TaskStatus; label: string }[] = [
  { status: 'backlog', label: 'task.columnBacklog' },
  { status: 'ready', label: 'task.columnReady' },
  { status: 'in_progress', label: 'task.columnInProgress' },
  { status: 'review', label: 'task.columnReview' },
  { status: 'done', label: 'task.columnDone' },
  { status: 'blocked', label: 'task.columnBlocked' },
]

const taskByID = computed(() => new Map(tasks.value.map((task) => [task.id, task])))
const hasPendingPlanApproval = computed(() =>
  pendingApprovals.value.some((approval) => approval.kind === 'plan_review'),
)
const taskCounts = computed(() => {
  const result: Record<string, number> = {}
  for (const task of tasks.value) result[task.status] = (result[task.status] || 0) + 1
  return result
})

function tasksByStatus(status: TaskStatus): Task[] {
  return tasks.value.filter((task) => task.status === status)
}

function dependencyLabels(task: Task): string[] {
  return task.dependencies.map((id) => taskByID.value.get(id)?.task_key || id.slice(0, 8))
}

function hasPendingExecution(task: Task): boolean {
  return pendingApprovals.value.some(
    (approval) => approval.kind === 'task_execution' && approval.task_id === task.id,
  )
}

async function load(silent = false) {
  if (!silent) loading.value = true
  try {
    const [taskItems, approvals] = await Promise.all([
      listTasks(props.projectId),
      listApprovals(props.projectId, 'pending'),
    ])
    tasks.value = taskItems
    pendingApprovals.value = approvals
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    if (!silent) loading.value = false
  }
}

async function regeneratePlan() {
  if (tasks.value.length > 0) {
    const confirmed = await ElMessageBox.confirm(
      t('task.regenerateConfirm'),
      t('common.confirm'),
      { type: 'warning' },
    ).catch(() => false)
    if (!confirmed) return
  }
  generating.value = true
  try {
    await generatePlan(props.projectId, tasks.value.length > 0)
    await load(true)
    ElMessage.success(t('task.generated'))
    if (hasPendingPlanApproval.value) emit('open-tab', 'activity')
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    generating.value = false
  }
}

async function requestExecution(task: Task) {
  actionTaskId.value = task.id
  try {
    await requestTaskExecution(task.id)
    await load(true)
    ElMessage.success(t('task.executionRequested'))
    emit('open-tab', 'activity')
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    actionTaskId.value = ''
  }
}

async function move(task: Task, status: TaskStatus) {
  actionTaskId.value = task.id
  try {
    await transitionTask(task.id, status)
    await load(true)
    ElMessage.success(t('task.updated'))
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    actionTaskId.value = ''
  }
}

function priorityType(priority: string): 'danger' | 'warning' | 'info' {
  if (priority === 'critical') return 'danger'
  if (priority === 'high') return 'warning'
  return 'info'
}

function formatRole(role: string): string {
  return t(`task.role.${role}`) === `task.role.${role}` ? role : t(`task.role.${role}`)
}

onMounted(() => void load())
onActivated(() => void load(true))
</script>

<template>
  <div v-loading="loading" class="task-panel">
    <header class="panel-head">
      <div>
        <h3>{{ t('task.title') }}</h3>
        <p>{{ t('task.description') }}</p>
      </div>
      <div class="head-actions">
        <el-button @click="load()">
          <RefreshCw :size="15" />
          <span>{{ t('common.refresh') }}</span>
        </el-button>
        <el-button type="primary" :loading="generating" @click="regeneratePlan">
          <ListTree :size="15" />
          <span>{{ tasks.length ? t('task.regenerate') : t('task.generate') }}</span>
        </el-button>
      </div>
    </header>

    <el-alert
      v-if="hasPendingPlanApproval"
      class="plan-alert"
      type="warning"
      :closable="false"
      show-icon
      :title="t('task.planPending')"
    >
      <template #default>
        <el-button link type="primary" @click="emit('open-tab', 'activity')">{{ t('task.reviewPlan') }}</el-button>
      </template>
    </el-alert>

    <div v-if="tasks.length === 0 && !loading" class="empty-state">
      <el-empty :description="t('task.empty')">
        <el-button type="primary" :loading="generating" @click="regeneratePlan">{{ t('task.generate') }}</el-button>
      </el-empty>
    </div>

    <div v-else class="board-scroll">
      <div class="board">
        <section v-for="column in columns" :key="column.status" class="board-column">
          <header class="column-head">
            <span>{{ t(column.label) }}</span>
            <el-tag size="small" effect="plain">{{ taskCounts[column.status] || 0 }}</el-tag>
          </header>

          <div class="column-body">
            <el-empty v-if="tasksByStatus(column.status).length === 0" :image-size="42" :description="t('task.noTasks')" />
            <article v-for="task in tasksByStatus(column.status)" :key="task.id" class="task-card">
              <div class="task-top">
                <span class="task-key">{{ task.task_key }}</span>
                <el-tag :type="priorityType(task.priority)" size="small" effect="plain">{{ task.priority }}</el-tag>
              </div>
              <h4>{{ task.title }}</h4>
              <p>{{ task.description }}</p>

              <div class="task-role">
                <GitBranch :size="13" />
                <span>{{ formatRole(task.agent_role) }}</span>
              </div>

              <div v-if="task.dependencies.length" class="dependencies">
                <span>{{ t('task.dependsOn') }}</span>
                <el-tag v-for="dependency in dependencyLabels(task)" :key="dependency" size="small" effect="plain">
                  {{ dependency }}
                </el-tag>
              </div>

              <div class="task-actions">
                <el-button
                  v-if="task.status === 'ready'"
                  type="primary"
                  size="small"
                  :loading="actionTaskId === task.id"
                  :disabled="hasPendingExecution(task)"
                  @click="requestExecution(task)"
                >
                  <CirclePlay :size="14" />
                  <span>{{ hasPendingExecution(task) ? t('task.awaitingApproval') : t('task.requestExecution') }}</span>
                </el-button>
                <template v-if="task.status === 'review'">
                  <el-button type="success" size="small" :loading="actionTaskId === task.id" @click="move(task, 'done')">
                    <CheckCheck :size="14" />
                    <span>{{ t('task.accept') }}</span>
                  </el-button>
                  <el-button size="small" :loading="actionTaskId === task.id" @click="move(task, 'in_progress')">
                    <Undo2 :size="14" />
                    <span>{{ t('task.reopen') }}</span>
                  </el-button>
                </template>
                <el-button
                  v-if="task.status === 'blocked'"
                  size="small"
                  :loading="actionTaskId === task.id"
                  @click="move(task, 'ready')"
                >
                  <Undo2 :size="14" />
                  <span>{{ t('task.requeue') }}</span>
                </el-button>
                <el-button
                  v-if="task.status === 'in_progress'"
                  size="small"
                  @click="emit('open-tab', 'activity')"
                >
                  <CirclePlay :size="14" />
                  <span>{{ t('task.viewRun') }}</span>
                </el-button>
              </div>
            </article>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.task-panel {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  padding: var(--act-space-5) var(--act-space-6);

  .panel-head {
    display: flex;
    flex: 0 0 auto;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--act-space-5);
    margin-bottom: var(--act-space-4);

    h3 { margin: 0; font-size: 16px; font-weight: 600; }
    p { margin: 5px 0 0; color: var(--act-text-muted); font-size: 12px; }
    .head-actions { display: flex; gap: var(--act-space-2); flex: 0 0 auto; }
  }

  .plan-alert {
    flex: 0 0 auto;
    margin-bottom: var(--act-space-3);
  }

  .empty-state {
    display: grid;
    min-height: 0;
    flex: 1;
    place-items: center;
  }

  .board-scroll {
    min-height: 0;
    flex: 1;
    overflow-x: auto;
    overflow-y: hidden;

    .board {
      display: grid;
      height: 100%;
      min-width: 1080px;
      gap: var(--act-space-3);
      grid-template-columns: repeat(6, minmax(180px, 1fr));
    }
  }

  .board-column {
    display: flex;
    min-height: 0;
    flex-direction: column;
    border: 1px solid var(--act-border);
    border-radius: var(--act-radius-md);
    background: var(--act-bg-soft);

    .column-head {
      display: flex;
      flex: 0 0 auto;
      align-items: center;
      justify-content: space-between;
      padding: var(--act-space-3);
      border-bottom: 1px solid var(--act-border);
      font-size: 12px;
      font-weight: 600;
    }

    .column-body {
      display: flex;
      min-height: 0;
      flex: 1;
      flex-direction: column;
      gap: var(--act-space-2);
      padding: var(--act-space-2);
      overflow-y: auto;
    }
  }

  .task-card {
    padding: var(--act-space-3);
    border: 1px solid var(--act-border);
    border-radius: var(--act-radius-sm);
    background: var(--act-surface);

    .task-top {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: var(--act-space-2);

      .task-key { color: var(--act-text-muted); font-family: var(--act-font-mono); font-size: 10px; }
    }

    h4 { margin: var(--act-space-2) 0 6px; font-size: 12px; font-weight: 600; line-height: 1.45; }
    p {
      display: -webkit-box;
      margin: 0;
      overflow: hidden;
      color: var(--act-text-muted);
      font-size: 11px;
      line-height: 1.55;
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 3;
    }

    .task-role {
      display: flex;
      gap: 5px;
      align-items: center;
      margin-top: var(--act-space-2);
      color: var(--act-text-secondary);
      font-size: 10px;
    }

    .dependencies {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      align-items: center;
      margin-top: var(--act-space-2);
      color: var(--act-text-muted);
      font-size: 10px;
    }

    .task-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
      margin-top: var(--act-space-3);
      padding-top: var(--act-space-2);
      border-top: 1px solid var(--act-border);
    }
  }
}
</style>

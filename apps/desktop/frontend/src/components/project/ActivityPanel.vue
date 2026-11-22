<script setup lang="ts">
import { computed, onActivated, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, ClipboardList, Play, RefreshCw, X } from 'lucide-vue-next'
import {
  decideApproval,
  finishRun,
  listApprovals,
  listRuns,
  listTasks,
} from '../../api/agent'
import { translateApiError } from '../../utils/error'
import type { AgentRun, Approval, Task } from '../../types/api'

const props = defineProps<{ projectId: string }>()
const { t, locale } = useI18n()

const approvals = ref<Approval[]>([])
const runs = ref<AgentRun[]>([])
const tasks = ref<Task[]>([])
const loading = ref(false)
const decidingId = ref('')
const finishDialogVisible = ref(false)
const finishing = ref(false)
const finishForm = reactive({
  runId: '',
  status: 'succeeded' as 'succeeded' | 'failed' | 'cancelled',
  output_summary: '',
})

const taskByID = computed(() => new Map(tasks.value.map((task) => [task.id, task])))
const pendingApprovals = computed(() => approvals.value.filter((item) => item.status === 'pending'))
const activeRuns = computed(() => runs.value.filter((item) => item.status === 'queued' || item.status === 'running'))

async function load(silent = false) {
  if (!silent) loading.value = true
  try {
    const [approvalItems, runItems, taskItems] = await Promise.all([
      listApprovals(props.projectId),
      listRuns(props.projectId),
      listTasks(props.projectId),
    ])
    approvals.value = approvalItems
    runs.value = runItems
    tasks.value = taskItems
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    if (!silent) loading.value = false
  }
}

async function decide(approval: Approval, decision: 'approve' | 'reject') {
  let note = ''
  if (decision === 'reject') {
    const result = await ElMessageBox.prompt(t('activity.rejectReason'), t('activity.reject'), {
      type: 'warning',
      inputPlaceholder: t('activity.rejectReasonPlaceholder'),
      confirmButtonText: t('activity.reject'),
      cancelButtonText: t('common.cancel'),
    }).catch(() => null)
    if (!result) return
    note = result.value || ''
  }
  decidingId.value = approval.id
  try {
    await decideApproval(approval.id, decision, note)
    await load(true)
    ElMessage.success(decision === 'approve' ? t('activity.approved') : t('activity.rejected'))
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    decidingId.value = ''
  }
}

function openFinish(run: AgentRun) {
  finishForm.runId = run.id
  finishForm.status = 'succeeded'
  finishForm.output_summary = ''
  finishDialogVisible.value = true
}

async function submitFinish() {
  finishing.value = true
  try {
    await finishRun(finishForm.runId, finishForm.status, finishForm.output_summary)
    finishDialogVisible.value = false
    await load(true)
    ElMessage.success(t('activity.runUpdated'))
  } catch (error) {
    ElMessage.error(translateApiError(error, t))
  } finally {
    finishing.value = false
  }
}

function approvalTask(approval: Approval): Task | undefined {
  return approval.task_id ? taskByID.value.get(approval.task_id) : undefined
}

function statusType(status: string): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'approved' || status === 'succeeded' || status === 'done') return 'success'
  if (status === 'pending' || status === 'queued' || status === 'running') return 'warning'
  if (status === 'rejected' || status === 'failed') return 'danger'
  return 'info'
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

onMounted(() => void load())
onActivated(() => void load(true))
</script>

<template>
  <div v-loading="loading" class="activity-panel">
    <header class="panel-head">
      <div>
        <h3>{{ t('activity.title') }}</h3>
        <p>{{ t('activity.description') }}</p>
      </div>
      <el-button @click="load()">
        <RefreshCw :size="15" />
        <span>{{ t('common.refresh') }}</span>
      </el-button>
    </header>

    <div class="activity-grid">
      <section class="activity-column">
        <div class="section-head">
          <div class="section-title"><ClipboardList :size="16" /><span>{{ t('activity.approvals') }}</span></div>
          <el-tag v-if="pendingApprovals.length" type="warning" effect="plain" size="small">
            {{ t('activity.pendingCount', { count: pendingApprovals.length }) }}
          </el-tag>
        </div>

        <div class="section-body">
          <el-empty v-if="approvals.length === 0" :image-size="60" :description="t('activity.noApprovals')" />
          <article v-for="approval in approvals" :key="approval.id" class="record-card">
            <div class="record-top">
              <div>
                <el-tag size="small" effect="plain" :type="statusType(approval.status)">
                  {{ t(`activity.approvalStatus.${approval.status}`) }}
                </el-tag>
                <span class="record-kind">{{ t(`activity.approvalKind.${approval.kind}`) }}</span>
              </div>
              <time>{{ formatTime(approval.created_at) }}</time>
            </div>
            <h4>{{ approval.title }}</h4>
            <p>{{ approval.description }}</p>
            <div v-if="approvalTask(approval)" class="record-link">
              <span>{{ t('task.task') }}</span>
              <el-tag size="small" effect="plain">{{ approvalTask(approval)?.task_key }}</el-tag>
              <span>{{ approvalTask(approval)?.title }}</span>
            </div>
            <el-text v-if="approval.decision_note" size="small" type="info">
              {{ t('activity.decisionNote') }}: {{ approval.decision_note }}
            </el-text>
            <div v-if="approval.status === 'pending'" class="record-actions">
              <el-button type="success" size="small" :loading="decidingId === approval.id" @click="decide(approval, 'approve')">
                <Check :size="14" />
                <span>{{ t('activity.approve') }}</span>
              </el-button>
              <el-button type="danger" plain size="small" :loading="decidingId === approval.id" @click="decide(approval, 'reject')">
                <X :size="14" />
                <span>{{ t('activity.reject') }}</span>
              </el-button>
            </div>
          </article>
        </div>
      </section>

      <section class="activity-column">
        <div class="section-head">
          <div class="section-title"><Play :size="16" /><span>{{ t('activity.runs') }}</span></div>
          <el-tag v-if="activeRuns.length" type="warning" effect="plain" size="small">
            {{ t('activity.activeCount', { count: activeRuns.length }) }}
          </el-tag>
        </div>

        <div class="section-body">
          <el-empty v-if="runs.length === 0" :image-size="60" :description="t('activity.noRuns')" />
          <article v-for="run in runs" :key="run.id" class="record-card">
            <div class="record-top">
              <div>
                <el-tag size="small" effect="plain" :type="statusType(run.status)">
                  {{ t(`activity.runStatus.${run.status}`) }}
                </el-tag>
                <span class="record-kind">{{ t(`activity.runKind.${run.kind}`) }}</span>
              </div>
              <time>{{ formatTime(run.created_at) }}</time>
            </div>
            <h4>{{ run.title }}</h4>
            <p v-if="run.output_summary">{{ run.output_summary }}</p>
            <p v-else-if="run.input_summary">{{ run.input_summary }}</p>
            <div class="run-logs">
              <div v-for="(log, index) in run.logs.slice(-4)" :key="`${run.id}-${index}`" class="run-log">
                <span>{{ formatTime(log.at) }}</span>
                <p>{{ log.message }}</p>
              </div>
            </div>
            <div v-if="run.status === 'queued' || run.status === 'running'" class="record-actions">
              <el-button size="small" type="primary" plain @click="openFinish(run)">
                {{ t('activity.recordResult') }}
              </el-button>
            </div>
          </article>
        </div>
      </section>
    </div>

    <el-dialog v-model="finishDialogVisible" :title="t('activity.recordResult')" width="520px" append-to-body>
      <el-form label-position="top">
        <el-form-item :label="t('activity.resultStatus')">
          <el-select v-model="finishForm.status" class="full-width">
            <el-option :label="t('activity.runStatus.succeeded')" value="succeeded" />
            <el-option :label="t('activity.runStatus.failed')" value="failed" />
            <el-option :label="t('activity.runStatus.cancelled')" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('activity.outputSummary')">
          <el-input
            v-model="finishForm.output_summary"
            type="textarea"
            :rows="4"
            :placeholder="t('activity.outputSummaryPlaceholder')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="finishDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="finishing" @click="submitFinish">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.activity-panel {
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
  }

  .activity-grid {
    display: grid;
    min-height: 0;
    flex: 1;
    gap: var(--act-space-4);
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .activity-column {
    display: flex;
    min-height: 0;
    flex-direction: column;
    border: 1px solid var(--act-border);
    border-radius: var(--act-radius-lg);
    background: var(--act-bg-soft);

    .section-head {
      display: flex;
      flex: 0 0 auto;
      align-items: center;
      justify-content: space-between;
      padding: var(--act-space-3) var(--act-space-4);
      border-bottom: 1px solid var(--act-border);

      .section-title { display: flex; gap: var(--act-space-2); align-items: center; font-size: 13px; font-weight: 600; }
    }

    .section-body {
      display: flex;
      min-height: 0;
      flex: 1;
      flex-direction: column;
      gap: var(--act-space-2);
      padding: var(--act-space-3);
      overflow-y: auto;
    }
  }

  .record-card {
    padding: var(--act-space-3);
    border: 1px solid var(--act-border);
    border-radius: var(--act-radius-md);
    background: var(--act-surface);

    .record-top {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: var(--act-space-2);

      .record-kind { margin-left: var(--act-space-2); color: var(--act-text-muted); font-size: 10px; }
      time { color: var(--act-text-muted); font-size: 10px; }
    }

    h4 { margin: var(--act-space-2) 0 6px; font-size: 13px; font-weight: 600; }
    p { margin: 0; color: var(--act-text-muted); font-size: 11px; line-height: 1.6; white-space: pre-wrap; }

    .record-link { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; margin-top: var(--act-space-2); font-size: 10px; color: var(--act-text-secondary); }

    .run-logs {
      display: flex;
      flex-direction: column;
      gap: 5px;
      margin-top: var(--act-space-2);
      padding: var(--act-space-2);
      border-radius: var(--act-radius-sm);
      background: var(--act-bg-soft);

      .run-log {
        display: grid;
        gap: 6px;
        grid-template-columns: 94px minmax(0, 1fr);
        color: var(--act-text-muted);
        font-size: 10px;

        p { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      }
    }

    .record-actions { display: flex; gap: var(--act-space-2); margin-top: var(--act-space-3); padding-top: var(--act-space-2); border-top: 1px solid var(--act-border); }
  }
}

@media (max-width: 1100px) {
  .activity-grid { grid-template-columns: 1fr; }
}
</style>
